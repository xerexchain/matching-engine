package orderbook

import (
	"bytes"
	"fmt"
	"log"

	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/order/action"
	"github.com/xerexchain/matching-engine/orderbook/bucket"
	"github.com/xerexchain/matching-engine/orderbook/event"
	resultcode "github.com/xerexchain/matching-engine/result_code"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/symbol"
)

// FIX rece condition, concurrency
// TODO role of symbol
// TODO implement stateHash according to java impl in IOrderBook.java
// TODO static CommandResultCode processCommand
// TODO static IOrderBook create
// TODO logging
// TODO IOC_BUDGET and FOK support

const (
	Naive  byte = 0
	Direct byte = 2
)

type OrderBook interface {
	// state.Hashable
	serialization.Marshalable
	Symbol() symbol.Symbol
	NumAskBuckets() int32
	NumBidBuckets() int32
	PlaceGTCOrder(order.Order) *MatcherResult
	PlaceIOCOrder(order.Order) *MatcherResult
	PlaceFOKBudgetOrder(order.Order) *MatcherResult
	MoveOrder(order.Move) *MatcherResult     // TODO adjust balance
	ReduceOrder(order.Reduce) *MatcherResult // TODO adjust balance // Decrease the size of the order by specific number of lots
	CancelOrder(order.Cancel) *MatcherResult // TODO adjust balance
	UserOrders(userId int64) []order.Order
	AskOrders() []interface{} // TODO How to return []order.Order
	BidOrders() []interface{} // TODO How to return []order.Order
	FillAsks(size int32, data *L2MarketData)
	FillBids(size int32, data *L2MarketData)
	ValidateInternalState()
}

type NaiveOrderBook interface {
	OrderBook
}

type MatcherResult struct {
	EventHead event.Event
	EventTail event.Event
	resultcode.ResultCode
	_ struct{}
}

type naiveOrderBook struct {
	askBuckets *btree.BTree
	bidBuckets *btree.BTree
	symbol     symbol.Symbol
	orders     map[int64]order.Order // used for reverse lookup
	_          struct{}
}

func (n *naiveOrderBook) sameBuckets(
	act action.Action,
) *btree.BTree {
	if act == action.Ask {
		return n.askBuckets
	} else {
		return n.bidBuckets
	}
}

func (n *naiveOrderBook) oppositeBuckets(
	act action.Action,
) *btree.BTree {
	if act == action.Ask {
		return n.bidBuckets
	} else {
		return n.askBuckets
	}
}

func (n *naiveOrderBook) findBucket(
	price int64,
	buckets *btree.BTree,
) (bucket.NaiveOrderBucket, bool) {
	var res bucket.NaiveOrderBucket
	pivot := bucket.NewDumpNaiveOrderBucket(price)

	buckets.AscendGreaterOrEqual(
		pivot,
		func(item btree.Item) bool {
			buck := item.(bucket.NaiveOrderBucket)

			if price == buck.Price() {
				res = buck
			}

			return false
		},
	)

	ok := res != nil

	return res, ok
}

func (n *naiveOrderBook) budgetToFill(
	toCollect int64,
	act action.Action,
) (int64, int64) {
	collected := int64(0)
	budget := int64(0)

	f := func(item btree.Item) bool {
		if toCollect == collected {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		totalQuantity := buck.TotalQuantity()
		price := buck.Price()

		if totalQuantity <= toCollect {
			budget += totalQuantity * price
			collected += totalQuantity
		} else {
			budget += toCollect * price
			collected += toCollect
		}

		return true
	}

	if act == action.Ask {
		n.bidBuckets.Descend(f)
	} else {
		n.askBuckets.Ascend(f)
	}

	return budget, collected
}

func (n *naiveOrderBook) tryMatchInstantly(
	ord order.Order,
) *MatcherResult {
	pivot := bucket.NewDumpNaiveOrderBucket(ord.Price())
	emptyBucks := []bucket.NaiveOrderBucket{}
	var head, tail event.TradeEvent

	f := func(item btree.Item) bool {
		if ord.Remained() == 0 {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		res := buck.Match(
			ord.Remained(),
			ord.ReservedBidPrice(),
		)

		for _, orderId := range res.RemovedOrders {
			delete(n.orders, orderId)
		}

		if tail == nil {
			head = res.EventHead
		} else {
			tail.SetNext(res.EventHead)
		}

		tail = res.EventTail

		ord.Fill(res.CollectedQuantity)

		if buck.TotalQuantity() == 0 {
			emptyBucks = append(emptyBucks, buck)
		}

		return true
	}

	if ord.Action() == action.Ask {
		n.bidBuckets.AscendGreaterOrEqual(pivot, f)
	} else {
		n.askBuckets.DescendLessOrEqual(pivot, f)
	}

	targetBucks := n.oppositeBuckets(ord.Action())

	// TODO Is it necessary?
	for _, buck := range emptyBucks {
		targetBucks.Delete(buck)
	}

	return &MatcherResult{
		EventHead:  head,
		EventTail:  tail,
		ResultCode: resultcode.Success,
	}
}

func (n *naiveOrderBook) Symbol() symbol.Symbol {
	return n.symbol
}

func (n *naiveOrderBook) NumAskBuckets() int32 {
	return int32(n.askBuckets.Len())
}

func (n *naiveOrderBook) NumBidBuckets() int32 {
	return int32(n.bidBuckets.Len())
}

func (n *naiveOrderBook) PlaceGTCOrder(
	ord order.Order,
) *MatcherResult {
	res := n.tryMatchInstantly(ord)

	if ord.Remained() == 0 {
		return res
	}

	if _, ok := n.orders[ord.Id()]; ok {
		log.Printf("warn: duplicate order id: %v\n", ord.Id())

		newHead := event.PrependRejectEvent(
			res.EventHead,
			ord.Id(),
			ord.Price(),
			ord.Remained(),
			ord.Action(),
		)
		res.EventHead = newHead

		return res
	}

	targetBucks := n.sameBuckets(ord.Action())

	buck, ok := n.findBucket(ord.Price(), targetBucks)

	if !ok {
		buck = bucket.NewNaiveOrderBucket(ord.Price())
		targetBucks.ReplaceOrInsert(buck)
	}

	buck.Put(ord)
	n.orders[ord.Id()] = ord

	return res
}

func (n *naiveOrderBook) PlaceIOCOrder(
	ord order.Order,
) *MatcherResult {
	res := n.tryMatchInstantly(ord)

	if ord.Remained() == 0 {
		return res
	}

	newHead := event.PrependRejectEvent(
		res.EventHead,
		ord.Id(),
		ord.Price(),
		ord.Remained(),
		ord.Action(),
	)
	res.EventHead = newHead

	return res
}

func (n *naiveOrderBook) PlaceFOKBudgetOrder(
	ord order.Order,
) *MatcherResult {
	budget, collected := n.budgetToFill(ord.Remained(), ord.Action())

	// TODO logic
	if collected == ord.Remained() || ((ord.Price() == budget) ||
		((ord.Action() == action.Ask) && (budget <= ord.Price())) ||
		((ord.Action() == action.Bid) && (budget > ord.Price()))) {
		return n.tryMatchInstantly(ord)
	} else {
		rejectEvent := event.NewRejectEvent(
			ord.Id(),
			ord.Price(),
			ord.Remained(),
			ord.Action(),
		)

		return &MatcherResult{
			EventHead:  rejectEvent,
			EventTail:  rejectEvent,
			ResultCode: resultcode.Success,
		}
	}
}

// TODO order.uid != cmd.uid
func (n *naiveOrderBook) MoveOrder(
	m order.Move,
) *MatcherResult {
	orderId := m.OrderId()
	toPrice := m.ToPrice()
	ord, ok := n.orders[orderId]

	if !ok {
		return &MatcherResult{
			ResultCode: resultcode.MatchingUnknownOrderId,
		}
	}

	if toPrice <= 0 || toPrice == ord.Price() {
		return &MatcherResult{
			ResultCode: resultcode.MatchingMoveFailedPriceInvalid, // TODO proper response code
		}
	}

	// reserved price risk check for exchange bids
	// TODO symbolSpec.type == SymbolType.CURRENCY_EXCHANGE_PAIR
	if ord.Action() == action.Bid && toPrice > ord.ReservedBidPrice() {
		return &MatcherResult{
			ResultCode: resultcode.MatchingMoveFailedPriceOverRiskLimit,
		}
	}

	newOrder := order.New(
		ord.Id(),
		ord.UserId(),
		toPrice,
		ord.Remained(),
		0,
		ord.ReservedBidPrice(), // TODO toPrice?
		ord.Timestamp(),        // TODO current timestamp?
		ord.Action(),
	)

	reduceOrder := order.NewReduceOrder(
		ord.Id(),
		ord.Remained(),
	)

	reduceRes := n.ReduceOrder(reduceOrder)
	gtcRes := n.PlaceGTCOrder(newOrder)

	reduceRes.EventTail.SetNext(gtcRes.EventHead)

	return &MatcherResult{
		EventHead:  reduceRes.EventHead,
		EventTail:  gtcRes.EventTail,
		ResultCode: gtcRes.ResultCode, // TODO success?
	}
}

// TODO order.uid != cmd.uid
func (n *naiveOrderBook) ReduceOrder(
	r order.Reduce,
) *MatcherResult {
	orderId := r.OrderId()
	reduceQuantity := r.ReduceQuantity()

	if reduceQuantity <= 0 {
		return &MatcherResult{
			ResultCode: resultcode.MatchingReduceFailedWrongQuantity,
		}
	}

	ord, ok := n.orders[orderId]

	if !ok {
		return &MatcherResult{
			ResultCode: resultcode.MatchingUnknownOrderId,
		}
	}

	if reduceQuantity > ord.Remained() {
		reduceQuantity = ord.Remained()
	}

	targetBucks := n.sameBuckets(ord.Action())

	buck, ok := n.findBucket(ord.Price(), targetBucks)

	if !ok {
		// not possible state
		panic(
			fmt.Sprintf(
				"warn: can not find bucket for order %s\n",
				ord.String(),
			),
		)
	}

	buck.Reduce(reduceQuantity)
	ord.Reduce(reduceQuantity)

	if ord.Remained() == 0 {
		delete(n.orders, orderId)
		buck.Remove(orderId)

		if buck.TotalQuantity() == 0 {
			targetBucks.Delete(buck)
		}
	}

	reduceEvent := event.NewReduceEvent(
		orderId,
		ord.Remained() == 0,
		ord.Price(),
		reduceQuantity,
		ord.Action(),
	)

	return &MatcherResult{
		EventHead:  reduceEvent,
		EventTail:  reduceEvent,
		ResultCode: resultcode.Success,
	}
}

// TODO order.uid == cmd.uid
func (n *naiveOrderBook) CancelOrder(
	c order.Cancel,
) *MatcherResult {
	orderId := c.OrderId()
	ord, ok := n.orders[orderId]

	if !ok {
		return &MatcherResult{
			ResultCode: resultcode.MatchingUnknownOrderId,
		}
	}

	reduceOrder := order.NewReduceOrder(
		ord.Id(),
		ord.Remained(),
	)

	return n.ReduceOrder(reduceOrder)
}

// TODO performance
func (n *naiveOrderBook) UserOrders(
	userId int64,
) []order.Order {
	res := []order.Order{}

	f := func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)

		buck.ForEachOrder(func(ord order.Order) {
			if userId == ord.UserId() {
				res = append(res, ord)
			}
		})

		return true
	}

	n.askBuckets.Ascend(f)
	n.bidBuckets.Descend(f)

	return res
}

// TODO performance
// TODO return []order.Order
func (n *naiveOrderBook) AskOrders() []interface{} {
	res := []interface{}{}

	n.askBuckets.Ascend(func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)
		allOrders := buck.AllOrders()
		res = append(res, allOrders...)

		return true
	})

	return res
}

// TODO performance
// TODO return []order.Order
func (n *naiveOrderBook) BidOrders() []interface{} {
	res := []interface{}{}

	// TODO duplicate code
	n.bidBuckets.Descend(func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)
		allOrders := buck.AllOrders()
		res = append(res, allOrders...)

		return true
	})

	return res
}

func (n *naiveOrderBook) FillAsks(size int32, data L2MarketData) {
	if size > data.AskSize() {
		size = data.AskSize()
	}

	var i int32 = 0

	n.askBuckets.Ascend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		data.SetAskPriceAt(i, buck.Price())
		data.SetAskQuantityAt(i, buck.TotalQuantity())
		data.SetNumAskOrdersAt(i, buck.NumOrders())
		i++

		return true
	})

	data.LimitAskViewTo(size)
}

func (n *naiveOrderBook) FillBids(size int32, data L2MarketData) {
	if size > data.BidSize() {
		size = data.BidSize()
	}

	var i int32 = 0

	n.bidBuckets.Descend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		data.SetBidPriceAt(i, buck.Price())
		data.SetBidQuantityAt(i, buck.TotalQuantity())
		data.SetNumBidOrdersAt(i, buck.NumOrders())
		i++

		return true
	})

	data.LimitBidViewTo(size)
}

func (n *naiveOrderBook) ValidateInternalState() {
	f := func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)
		buck.Validate()

		return true
	}

	n.askBuckets.Ascend(f)
	n.askBuckets.Descend(f)
}

func (n *naiveOrderBook) Hash() uint64 {
	// TODO impl
	return 0
}

func (n *naiveOrderBook) Marshal(out *bytes.Buffer) error {
	return MarshalNaiveOrderBook(n, out)
}

func MarshalNaiveOrderBook(in interface{}, out *bytes.Buffer) error {
	n := in.(*naiveOrderBook)

	if err := serialization.MarshalInt8(Naive, out); err != nil {
		return err
	}

	if err := n.symbol.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalBtree(
		n.askBuckets,
		out,
		bucket.MarshalNaiveOrderBucket,
	); err != nil {
		return err
	}

	if err := serialization.MarshalBtree(
		n.bidBuckets,
		out,
		bucket.MarshalNaiveOrderBucket,
	); err != nil {
		return err
	}

	return nil
}

func UnmarshalNaiveOrderBook(b *bytes.Buffer) (interface{}, error) {
	n := naiveOrderBook{}

	if s, err := symbol.UnmarshalSymbol(b); err != nil {
		return nil, err
	} else {
		n.symbol = s.(symbol.Symbol)
	}

	if askBuckets, err := serialization.UnmarshalBtree(
		b,
		bucket.UnmarshalNaiveOrderBucket,
	); err != nil {
		return nil, err
	} else {
		n.askBuckets = askBuckets
	}

	if bidBuckets, err := serialization.UnmarshalBtree(
		b,
		bucket.UnmarshalNaiveOrderBucket,
	); err != nil {
		return nil, err
	} else {
		n.bidBuckets = bidBuckets
	}

	var numOrders int64 = 0

	counter := func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)
		numOrders += int64(buck.NumOrders())

		return true
	}

	n.askBuckets.Ascend(counter)
	n.bidBuckets.Descend(counter)

	n.orders = make(map[int64]order.Order, numOrders)

	appender := func(item btree.Item) bool {
		buck := item.(bucket.NaiveOrderBucket)

		buck.ForEachOrder(func(ord order.Order) {
			n.orders[ord.Id()] = ord
		})

		return true
	}

	n.askBuckets.Ascend(appender)
	n.bidBuckets.Descend(appender)

	return &n, nil
}

func L2MarketDataSnapshot(orderbook OrderBook, limit int32) *L2MarketData {
	askSize := orderbook.NumAskBuckets()
	bidSize := orderbook.NumBidBuckets()

	if limit < askSize {
		askSize = limit
	}
	if limit < bidSize {
		bidSize = limit
	}

	data := NewL2MarketData(askSize, bidSize)
	orderbook.FillAsks(askSize, data)
	orderbook.FillBids(bidSize, data)

	return data
}

func PublishL2MarketDataSnapshot(orderbook OrderBook, data *L2MarketData) {
	orderbook.FillAsks(L2Size, data)
	orderbook.FillBids(L2Size, data)
}
