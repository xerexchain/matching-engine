package orderbook

import (
	"bytes"
	"fmt"
	"log"

	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/orderbook/orderbucket"
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
	Naive_  int8 = 0
	Direct_ int8 = 2
)

type OrderBook interface {
	// state.Hashable
	serialization.Marshalable
	Symbol() symbol.Symbol
	NumAskBuckets() int32
	NumBidBuckets() int32
	PlaceGTC(*order.Place) *MatcherResult
	PlaceIOC(*order.Place) *MatcherResult
	PlaceFOKBudget(*order.Place) *MatcherResult
	Move(*order.Move) *MatcherResult     // TODO adjust balance
	Reduce(*order.Reduce) *MatcherResult // TODO adjust balance // Decrease the size of the order by specific number of lots
	Cancel(*order.Cancel) *MatcherResult // TODO adjust balance
	UserOrders(int64) []*order.Order
	AskOrders() []interface{} // TODO How to return []order.Order
	BidOrders() []interface{} // TODO How to return []order.Order
	FillAsks(int32, *L2MarketData)
	FillBids(int32, *L2MarketData)
	ValidateInternalState()
}

type Naive interface {
	OrderBook
}

type MatcherResult struct {
	EventHead event.Event
	EventTail event.Event
	resultcode.ResultCode
	_ struct{}
}

type naive struct {
	askBuckets *btree.BTree
	bidBuckets *btree.BTree
	symbol     symbol.Symbol
	orders     map[int64]*order.Order // used for reverse lookup
	_          struct{}
}

func (n *naive) sameBucketsAs(
	action order.Action,
) *btree.BTree {
	if action == order.Ask {
		return n.askBuckets
	} else {
		return n.bidBuckets
	}
}

func (n *naive) oppositeBucketsTo(
	action order.Action,
) *btree.BTree {
	if action == order.Ask {
		return n.bidBuckets
	} else {
		return n.askBuckets
	}
}

func (n *naive) findBucket(
	price int64,
	buckets *btree.BTree,
) (orderbucket.Naive, bool) {
	var res orderbucket.Naive
	pivot := orderbucket.NewDumpNaive(price)

	buckets.AscendGreaterOrEqual(
		pivot,
		func(item btree.Item) bool {
			buck := item.(orderbucket.Naive)

			if price == buck.Price() {
				res = buck
			}

			return false
		},
	)

	ok := res != nil

	return res, ok
}

func (n *naive) budgetToFill(
	toCollect int64,
	action order.Action,
) (int64, int64) {
	collected := int64(0)
	budget := int64(0)

	f := func(item btree.Item) bool {
		if toCollect == collected {
			return false
		}

		buck := item.(orderbucket.Naive)
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

	if action == order.Ask {
		n.bidBuckets.Descend(f)
	} else {
		n.askBuckets.Ascend(f)
	}

	return budget, collected
}

func (n *naive) tryMatchInstantly(
	command *order.Place,
) *MatcherResult {
	pivot := orderbucket.NewDumpNaive(command.Price())
	emptyBucks := []orderbucket.Naive{}
	var head, tail event.Trade

	f := func(item btree.Item) bool {
		if command.Quantity() == 0 {
			return false
		}

		buck := item.(orderbucket.Naive)
		res := buck.Match(
			command.Quantity(),
			command.ReservedPrice(),
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

		command.Reduce(res.CollectedQuantity)

		if buck.TotalQuantity() == 0 {
			emptyBucks = append(emptyBucks, buck)
		}

		return true
	}

	if command.Action() == order.Ask {
		n.bidBuckets.AscendGreaterOrEqual(pivot, f)
	} else {
		n.askBuckets.DescendLessOrEqual(pivot, f)
	}

	targetBucks := n.oppositeBucketsTo(command.Action())

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

func (n *naive) Symbol() symbol.Symbol {
	return n.symbol
}

func (n *naive) NumAskBuckets() int32 {
	return int32(n.askBuckets.Len())
}

func (n *naive) NumBidBuckets() int32 {
	return int32(n.bidBuckets.Len())
}

func (n *naive) PlaceGTC(
	gtc *order.Place,
) *MatcherResult {
	res := n.tryMatchInstantly(gtc)

	if gtc.Quantity() == 0 {
		return res
	}

	if _, ok := n.orders[gtc.OrderID()]; ok {
		log.Printf("warn: duplicate order id: %v\n", gtc.OrderID())

		newHead := event.PrependReject(
			res.EventHead,
			gtc.OrderID(),
			gtc.Price(),
			gtc.Quantity(),
			gtc.Action(),
		)
		res.EventHead = newHead

		return res
	}

	targetBucks := n.sameBucketsAs(gtc.Action())

	buck, ok := n.findBucket(gtc.Price(), targetBucks)

	if !ok {
		buck = orderbucket.NewNaive(gtc.Price())
		targetBucks.ReplaceOrInsert(buck)
	}

	ord := order.New(
		gtc.OrderID(),
		gtc.UserID(),
		gtc.Price(),
		gtc.Quantity(),
		0, // TODO
		gtc.ReservedPrice(),
		gtc.Timestamp(),
		gtc.Action(),
	)

	buck.Put(ord)
	n.orders[ord.ID()] = ord

	return res
}

func (n *naive) PlaceIOC(
	ioc *order.Place,
) *MatcherResult {
	res := n.tryMatchInstantly(ioc)

	if ioc.Quantity() == 0 {
		return res
	}

	newHead := event.PrependReject(
		res.EventHead,
		ioc.OrderID(),
		ioc.Price(),
		ioc.Quantity(),
		ioc.Action(),
	)
	res.EventHead = newHead

	return res
}

func (n *naive) PlaceFOKBudget(
	fok *order.Place,
) *MatcherResult {
	budget, collected := n.budgetToFill(fok.Quantity(), fok.Action())

	// TODO logic
	if collected == fok.Quantity() || ((fok.Price() == budget) ||
		((fok.Action() == order.Ask) && (budget <= fok.Price())) ||
		((fok.Action() == order.Bid) && (budget > fok.Price()))) {
		return n.tryMatchInstantly(fok)
	} else {
		e := event.NewReject(
			fok.OrderID(),
			fok.Price(),
			fok.Quantity(),
			fok.Action(),
		)

		return &MatcherResult{
			EventHead:  e,
			EventTail:  e,
			ResultCode: resultcode.Success,
		}
	}
}

// TODO order.uid != cmd.uid
func (n *naive) Move(
	command *order.Move,
) *MatcherResult {
	orderId := command.OrderID()
	toPrice := command.ToPrice()
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
	if ord.Action() == order.Bid && toPrice > ord.ReservedBidPrice() {
		return &MatcherResult{
			ResultCode: resultcode.MatchingMoveFailedPriceOverRiskLimit,
		}
	}

	gtc := order.NewPlace(
		ord.ID(),
		ord.UserID(),
		toPrice,
		ord.Remained(),
		ord.ReservedBidPrice(), // TODO toPrice?
		n.symbol.Id(),
		ord.Timestamp(), // TODO current timestamp?
		ord.Action(),
		order.GTC,
	)

	red := order.NewReduce(
		ord.ID(),
		n.symbol.Id(),
		ord.Remained(),
	)

	reduceRes := n.Reduce(red)
	gtcRes := n.PlaceGTC(gtc)

	reduceRes.EventTail.SetNext(gtcRes.EventHead)

	return &MatcherResult{
		EventHead:  reduceRes.EventHead,
		EventTail:  gtcRes.EventTail,
		ResultCode: gtcRes.ResultCode, // TODO success?
	}
}

// TODO order.uid != cmd.uid
func (n *naive) Reduce(
	command *order.Reduce,
) *MatcherResult {
	orderID := command.OrderID()
	quantity := command.Quantity()

	if quantity <= 0 {
		return &MatcherResult{
			ResultCode: resultcode.MatchingReduceFailedWrongQuantity,
		}
	}

	ord, ok := n.orders[orderID]

	if !ok {
		return &MatcherResult{
			ResultCode: resultcode.MatchingUnknownOrderId,
		}
	}

	if quantity > ord.Remained() {
		quantity = ord.Remained()
	}

	targetBucks := n.sameBucketsAs(ord.Action())

	buck, ok := n.findBucket(ord.Price(), targetBucks)

	if !ok {
		// not possible state
		panic(
			fmt.Sprintf(
				"warn: can not find bucket for order %v\n",
				ord.ID(),
			),
		)
	}

	buck.Reduce(quantity)
	// TODO catch error
	ord.Reduce(quantity)

	if ord.Remained() == 0 {
		delete(n.orders, orderID)
		buck.Remove(orderID)

		if buck.TotalQuantity() == 0 {
			targetBucks.Delete(buck)
		}
	}

	e := event.NewReduce(
		orderID,
		ord.Remained() == 0,
		ord.Price(),
		quantity,
		ord.Action(),
	)

	return &MatcherResult{
		EventHead:  e,
		EventTail:  e,
		ResultCode: resultcode.Success,
	}
}

// TODO order.uid == cmd.uid
func (n *naive) Cancel(
	command *order.Cancel,
) *MatcherResult {
	orderID := command.OrderID()
	ord, ok := n.orders[orderID]

	if !ok {
		return &MatcherResult{
			ResultCode: resultcode.MatchingUnknownOrderId,
		}
	}

	red := order.NewReduce(
		ord.ID(),
		n.symbol.Id(),
		ord.Remained(),
	)

	return n.Reduce(red)
}

// TODO performance
func (n *naive) UserOrders(
	userId int64,
) []*order.Order {
	res := []*order.Order{}

	f := func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)

		buck.ForEachOrder(func(ord *order.Order) {
			if userId == ord.UserID() {
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
func (n *naive) AskOrders() []interface{} {
	res := []interface{}{}

	n.askBuckets.Ascend(func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)
		allOrders := buck.AllOrders()
		res = append(res, allOrders...)

		return true
	})

	return res
}

// TODO performance
// TODO return []order.Order
func (n *naive) BidOrders() []interface{} {
	res := []interface{}{}

	// TODO duplicate code
	n.bidBuckets.Descend(func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)
		allOrders := buck.AllOrders()
		res = append(res, allOrders...)

		return true
	})

	return res
}

func (n *naive) FillAsks(size int32, data *L2MarketData) {
	if size > data.AskSize() {
		size = data.AskSize()
	}

	var i int32 = 0

	n.askBuckets.Ascend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		buck := item.(orderbucket.Naive)
		data.SetAskPriceAt(i, buck.Price())
		data.SetAskQuantityAt(i, buck.TotalQuantity())
		data.SetNumAskOrdersAt(i, buck.NumOrders())
		i++

		return true
	})

	data.LimitAskViewTo(size)
}

func (n *naive) FillBids(size int32, data *L2MarketData) {
	if size > data.BidSize() {
		size = data.BidSize()
	}

	var i int32 = 0

	n.bidBuckets.Descend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		buck := item.(orderbucket.Naive)
		data.SetBidPriceAt(i, buck.Price())
		data.SetBidQuantityAt(i, buck.TotalQuantity())
		data.SetNumBidOrdersAt(i, buck.NumOrders())
		i++

		return true
	})

	data.LimitBidViewTo(size)
}

func (n *naive) ValidateInternalState() {
	f := func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)
		buck.Validate()

		return true
	}

	n.askBuckets.Ascend(f)
	n.askBuckets.Descend(f)
}

func (n *naive) Hash() uint64 {
	// TODO impl
	return 0
}

func (n *naive) Marshal(out *bytes.Buffer) error {
	return MarshalNaive(n, out)
}

func MarshalNaive(in interface{}, out *bytes.Buffer) error {
	n := in.(*naive)

	if err := serialization.WriteInt8(Naive_, out); err != nil {
		return err
	}

	if err := n.symbol.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalBtree(
		n.askBuckets,
		out,
		orderbucket.MarshalNaive,
	); err != nil {
		return err
	}

	if err := serialization.MarshalBtree(
		n.bidBuckets,
		out,
		orderbucket.MarshalNaive,
	); err != nil {
		return err
	}

	return nil
}

func UnmarshalNaive(b *bytes.Buffer) (interface{}, error) {
	n := naive{}

	if s, err := symbol.UnmarshalSymbol(b); err != nil {
		return nil, err
	} else {
		n.symbol = s.(symbol.Symbol)
	}

	if askBuckets, err := serialization.UnmarshalBtree(
		b,
		orderbucket.UnmarshalNaive,
	); err != nil {
		return nil, err
	} else {
		n.askBuckets = askBuckets
	}

	if bidBuckets, err := serialization.UnmarshalBtree(
		b,
		orderbucket.UnmarshalNaive,
	); err != nil {
		return nil, err
	} else {
		n.bidBuckets = bidBuckets
	}

	var numOrders int64 = 0

	counter := func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)
		numOrders += int64(buck.NumOrders())

		return true
	}

	n.askBuckets.Ascend(counter)
	n.bidBuckets.Descend(counter)

	n.orders = make(map[int64]*order.Order, numOrders)

	appender := func(item btree.Item) bool {
		buck := item.(orderbucket.Naive)

		buck.ForEachOrder(func(ord *order.Order) {
			n.orders[ord.ID()] = ord
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

func NewNaive(sym symbol.Symbol) OrderBook {
	return &naive{
		askBuckets: btree.New(4), // TODO adjust 4
		bidBuckets: btree.New(4), // TODO adjust 4
		symbol:     sym,
		orders:     make(map[int64]*order.Order),
	}
}
