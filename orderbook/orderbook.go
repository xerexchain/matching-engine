package orderbook

import (
	"bytes"
	"fmt"
	"log"

	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/orderbook/bucket"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/resultcode"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/symbol"
)

// FIX rece condition, concurrency
// TODO implement stateHash according to java impl in IOrderBook.java, unexported fields
// TODO static CommandResultCode processCommand
// TODO static IOrderBook create
// TODO logging
// TODO IOC_BUDGET and FOK support

const (
	_naiveOrderBook int8 = iota + 1
	_directOrderBook
	_btreeDegree = 4 // TODO adjust
)

var _ _OrderBook = (*Naive)(nil)

type MatcherResult struct {
	Head event.Event
	Tail event.Event
	Code resultcode.ResultCode
	_    struct{}
}

type _OrderBook interface {
	NumAskBuckets() int32
	NumBidBuckets() int32
	FillAsks(int32, *L2MarketData)
	FillBids(int32, *L2MarketData)
}

// TODO rename `Naive` and `_naiveOrderBook`
type Naive struct {
	askBuckets *btree.BTree
	bidBuckets *btree.BTree
	symbol     symbol.Symbol
	orders     map[int64]*order.Order // used for reverse lookup
	_          struct{}
}

func NewNaive(symbol_ symbol.Symbol) *Naive {
	return &Naive{
		askBuckets: btree.New(_btreeDegree),
		bidBuckets: btree.New(_btreeDegree),
		symbol:     symbol_,
		orders:     make(map[int64]*order.Order),
	}
}

func (n *Naive) sameBucketsAs(
	action order.Action,
) *btree.BTree {
	if action == order.Ask {
		return n.askBuckets
	} else {
		return n.bidBuckets
	}
}

func (n *Naive) oppositeBucketsTo(
	action order.Action,
) *btree.BTree {
	if action == order.Ask {
		return n.bidBuckets
	} else {
		return n.askBuckets
	}
}

func (n *Naive) findBucket(
	price int64,
	buckets *btree.BTree,
) (*bucket.Bucket, bool) {
	var (
		bucket_ *bucket.Bucket
		pivot   = bucket.With(price)
	)

	buckets.AscendGreaterOrEqual(
		pivot,
		func(item btree.Item) bool {
			candidate := item.(*bucket.Bucket)

			if price == candidate.Price() {
				bucket_ = candidate
			}

			return false
		},
	)

	ok := bucket_ != nil

	return bucket_, ok
}

func (n *Naive) budgetToFill(
	toCollect int64,
	action order.Action,
) (budget, collected int64) {
	f := func(item btree.Item) bool {
		if toCollect == collected {
			return false
		}

		bucket_ := item.(*bucket.Bucket)
		totalQuantity := bucket_.TotalQuantity()
		price := bucket_.Price()

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

func (n *Naive) match(
	command *order.Place, // TODO rename
) *MatcherResult {
	var (
		head         *event.Trade
		tail         *event.Trade
		emptyBuckets []*bucket.Bucket
		pivot        = bucket.With(command.Price())
	)

	f := func(item btree.Item) bool {
		if command.Quantity() == 0 {
			return false
		}

		bucket_ := item.(*bucket.Bucket)

		res := bucket_.Match(
			command.Quantity(),
			command.ReservedPrice(),
		)

		for _, orderID := range res.RemovedOrders {
			delete(n.orders, orderID)
		}

		if tail == nil {
			head = res.Head
		} else {
			tail.SetNext(res.Head)
		}

		tail = res.Tail

		command.Reduce(res.CollectedQuantity)

		if bucket_.TotalQuantity() == 0 {
			emptyBuckets = append(emptyBuckets, bucket_)
		}

		return true
	}

	if command.Action() == order.Ask {
		n.bidBuckets.AscendGreaterOrEqual(pivot, f)
	} else {
		// FIX iterate from lowest to highest price.
		n.askBuckets.DescendLessOrEqual(pivot, f)
	}

	targetBuckets := n.oppositeBucketsTo(command.Action())

	// TODO Is it necessary?
	for _, bucket_ := range emptyBuckets {
		targetBuckets.Delete(bucket_)
	}

	return &MatcherResult{
		Head: head,
		Tail: tail,
		Code: resultcode.Success,
	}
}

func (n *Naive) Symbol() symbol.Symbol {
	return n.symbol
}

func (n *Naive) NumAskBuckets() int32 {
	return int32(n.askBuckets.Len())
}

func (n *Naive) NumBidBuckets() int32 {
	return int32(n.bidBuckets.Len())
}

func (n *Naive) PlaceGTC(
	gtc *order.Place,
) *MatcherResult {
	res := n.match(gtc)

	if gtc.Quantity() == 0 {
		return res
	}

	if _, ok := n.orders[gtc.OrderID()]; ok {
		log.Printf("duplicate order id: %v", gtc.OrderID())

		e := event.NewReject(
			gtc.OrderID(),
			gtc.Price(),
			gtc.Quantity(),
			gtc.Action(),
		)
		e.SetNext(res.Head)
		res.Head = e

		return res
	}

	targetBuckets := n.sameBucketsAs(gtc.Action())

	bucket_, ok := n.findBucket(gtc.Price(), targetBuckets)

	if !ok {
		bucket_ = bucket.New(gtc.Price())
		targetBuckets.ReplaceOrInsert(bucket_)
	}

	// TODO should set filled = 0 ?
	ord := order.New(
		gtc.OrderID(),
		gtc.UserID(),
		gtc.Price(),
		gtc.Quantity(),
		0,
		gtc.ReservedPrice(),
		gtc.Timestamp(), // TODO current time?
		gtc.Action(),
	)

	bucket_.Put(ord)
	n.orders[ord.ID()] = ord

	return res
}

func (n *Naive) PlaceIOC(
	ioc *order.Place,
) *MatcherResult {
	res := n.match(ioc)

	if ioc.Quantity() == 0 {
		return res
	}

	e := event.NewReject(
		ioc.OrderID(),
		ioc.Price(),
		ioc.Quantity(),
		ioc.Action(),
	)
	e.SetNext(res.Head)
	res.Head = e

	return res
}

func (n *Naive) PlaceFOKBudget(
	fok *order.Place,
) *MatcherResult {
	action := fok.Action()
	price := fok.Price()
	quantity := fok.Quantity()
	budget, collected := n.budgetToFill(quantity, action)

	// TODO logic
	if collected == quantity || ((price == budget) ||
		((action == order.Ask) && (budget <= price)) ||
		((action == order.Bid) && (budget > price))) {
		return n.match(fok)
	} else {
		e := event.NewReject(
			fok.OrderID(),
			fok.Price(),
			fok.Quantity(),
			fok.Action(),
		)

		return &MatcherResult{
			Head: e,
			Tail: e,
			Code: resultcode.Success,
		}
	}
}

// TODO check order.userID == cmd.userID (auth)
func (n *Naive) Move(
	command *order.Move, // TODO rename
) *MatcherResult {
	orderID := command.OrderID()
	toPrice := command.ToPrice()
	ord, ok := n.orders[orderID]

	if !ok {
		return &MatcherResult{
			Code: resultcode.MatchingUnknownOrderID,
		}
	}

	if toPrice <= 0 || toPrice == ord.Price() {
		return &MatcherResult{
			// TODO proper response code
			Code: resultcode.MatchingMoveFailedPriceInvalid,
		}
	}

	// reserved price risk check for exchange bids
	// TODO symbolSpec.type == SymbolType.CURRENCY_EXCHANGE_PAIR
	if ord.Action() == order.Bid && toPrice > ord.ReservedBidPrice() {
		return &MatcherResult{
			Code: resultcode.MatchingMoveFailedPriceOverRiskLimit,
		}
	}

	gtc := order.NewPlace(
		ord.ID(),
		ord.UserID(),
		toPrice,
		ord.Remained(),
		ord.ReservedBidPrice(), // TODO toPrice?
		n.symbol.Id(),
		ord.Timestamp(), // TODO current time?
		ord.Action(),
		order.GTC,
	)

	// TODO rename
	reduce := order.NewReduce(
		ord.ID(),
		n.symbol.Id(),
		ord.Remained(),
	)

	// TODO rename
	reduceResult := n.Reduce(reduce)
	// TODO rename
	placeResult := n.PlaceGTC(gtc)

	reduceResult.Tail.SetNext(placeResult.Head)

	return &MatcherResult{
		Head: reduceResult.Head,
		Tail: placeResult.Tail,
		Code: placeResult.Code, // TODO success?
	}
}

// TODO check order.userID == cmd.userID (auth)
func (n *Naive) Reduce(
	command *order.Reduce, // TODO rename
) *MatcherResult {
	orderID := command.OrderID()
	quantity := command.Quantity()

	if quantity <= 0 {
		return &MatcherResult{
			Code: resultcode.MatchingReduceFailedWrongQuantity,
		}
	}

	ord, ok := n.orders[orderID]

	if !ok {
		return &MatcherResult{
			Code: resultcode.MatchingUnknownOrderID,
		}
	}

	if quantity > ord.Remained() {
		quantity = ord.Remained()
	}

	targetBuckets := n.sameBucketsAs(ord.Action())

	bucket_, ok := n.findBucket(ord.Price(), targetBuckets)

	if !ok {
		// not possible state
		// TODO panic?
	}

	bucket_.Reduce(quantity)

	if err := ord.Reduce(quantity); err != nil {
		// not possible state
		// TODO panic?
	}

	if ord.Remained() == 0 {
		delete(n.orders, orderID)
		bucket_.Remove(orderID)

		if bucket_.TotalQuantity() == 0 {
			targetBuckets.Delete(bucket_)
		}
	}

	e := event.NewReduce(
		orderID,
		ord.Remained() == 0, /*makerOrderCompleted*/
		ord.Price(),
		quantity,
		ord.Action(),
	)

	return &MatcherResult{
		Head: e,
		Tail: e,
		Code: resultcode.Success,
	}
}

// TODO check order.userID == cmd.userID (auth)
func (n *Naive) Cancel(
	command *order.Cancel, // TODO rename
) *MatcherResult {
	orderID := command.OrderID()
	ord, ok := n.orders[orderID]

	if !ok {
		return &MatcherResult{
			Code: resultcode.MatchingUnknownOrderID,
		}
	}

	// TODO rename
	reduce := order.NewReduce(
		ord.ID(),
		n.symbol.Id(),
		ord.Remained(),
	)

	return n.Reduce(reduce)
}

// TODO performance
func (n *Naive) UserOrders(
	userID int64,
) []*order.Order {
	var userOrders []*order.Order

	f := func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)

		bucket_.ForEachOrder(func(ord *order.Order) {
			if userID == ord.UserID() {
				userOrders = append(userOrders, ord)
			}
		})

		return true
	}

	n.askBuckets.Ascend(f)
	n.bidBuckets.Descend(f)

	return userOrders
}

// TODO performance
// TODO return []*order.Order
func (n *Naive) AskOrders() []interface{} {
	var askOrders []interface{}

	n.askBuckets.Ascend(func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)
		allOrders := bucket_.AllOrders()
		askOrders = append(askOrders, allOrders...)

		return true
	})

	return askOrders
}

// TODO performance
// TODO return []*order.Order
func (n *Naive) BidOrders() []interface{} {
	var bidOrders []interface{}

	// TODO duplicate code
	n.bidBuckets.Descend(func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)
		allOrders := bucket_.AllOrders()
		bidOrders = append(bidOrders, allOrders...)

		return true
	})

	return bidOrders
}

func (n *Naive) FillAsks(size int32, marketData *L2MarketData) {
	if size > marketData.AskSize() {
		size = marketData.AskSize()
	}

	var i int32 = 0

	n.askBuckets.Ascend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		bucket_ := item.(*bucket.Bucket)
		marketData.SetAskPriceAt(i, bucket_.Price())
		marketData.SetAskQuantityAt(i, bucket_.TotalQuantity())
		marketData.SetNumAskOrdersAt(i, bucket_.NumOrders())
		i++

		return true
	})

	marketData.LimitAskViewTo(size)
}

func (n *Naive) FillBids(size int32, marketData *L2MarketData) {
	if size > marketData.BidSize() {
		size = marketData.BidSize()
	}

	var i int32 = 0

	n.bidBuckets.Descend(func(item btree.Item) bool {
		if i == size {
			return false
		}

		bucket_ := item.(*bucket.Bucket)
		marketData.SetBidPriceAt(i, bucket_.Price())
		marketData.SetBidQuantityAt(i, bucket_.TotalQuantity())
		marketData.SetNumBidOrdersAt(i, bucket_.NumOrders())
		i++

		return true
	})

	marketData.LimitBidViewTo(size)
}

func (n *Naive) IsValid() bool {
	ok := true

	f := func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)
		ok = bucket_.IsValid()

		return ok
	}

	n.askBuckets.Ascend(f)

	if !ok {
		return false
	}

	ok = true

	n.askBuckets.Descend(f)

	return ok
}

func (n *Naive) Hash() uint64 {
	// TODO impl
	return 0
}

func (n *Naive) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt8(_naiveOrderBook, out); err != nil {
		return err
	}

	if err := n.symbol.Marshal(out); err != nil {
		return err
	}

	f := func(buckets *btree.BTree) error {
		var (
			size = int32(buckets.Len())
			err  error
		)

		if err := serialization.WriteInt32(size, out); err != nil {
			return err
		}

		buckets.Ascend(func(v btree.Item) bool {
			err = v.(*bucket.Bucket).Marshal(out)

			return err == nil
		})

		return err
	}

	if err := f(n.askBuckets); err != nil {
		return err
	}

	if err := f(n.bidBuckets); err != nil {
		return err
	}

	return nil
}

func (n *Naive) Unmarshal(in *bytes.Buffer) error {
	d, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	if d != _naiveOrderBook {
		return fmt.Errorf("Naive unmarshal: expected %v, got  %v", _naiveOrderBook, d)
	}

	symbol_, err := symbol.UnmarshalSymbol(in)

	if err != nil {
		return err
	}

	f := func() (*btree.BTree, error) {
		var (
			size int32
			err  error
		)

		size, err = serialization.ReadInt32(in)

		if err != nil {
			return nil, err
		}

		buckets := btree.New(_btreeDegree)

		for ; size > 0; size-- {
			bucket_ := &bucket.Bucket{}

			if err = bucket_.Unmarshal(in); err != nil {
				return nil, err
			}

			buckets.ReplaceOrInsert(bucket_)
		}

		return buckets, nil
	}

	askBuckets, err := f()

	if err != nil {
		return err
	}

	bidBuckets, err := f()

	if err != nil {
		return err
	}

	var numOrders int64 = 0

	counter := func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)
		numOrders += int64(bucket_.NumOrders())

		return true
	}

	askBuckets.Ascend(counter)
	bidBuckets.Descend(counter)

	orders := make(map[int64]*order.Order, numOrders)

	appender := func(item btree.Item) bool {
		bucket_ := item.(*bucket.Bucket)

		bucket_.ForEachOrder(func(ord *order.Order) {
			orders[ord.ID()] = ord
		})

		return true
	}

	askBuckets.Ascend(appender)
	bidBuckets.Descend(appender)

	n.askBuckets = askBuckets
	n.bidBuckets = bidBuckets
	n.symbol = symbol_.(symbol.Symbol)
	n.orders = orders

	return nil
}

func L2MarketDataSnapshot(orderbook_ _OrderBook, limit int32) *L2MarketData {
	askSize := orderbook_.NumAskBuckets()
	bidSize := orderbook_.NumBidBuckets()

	if limit < askSize {
		askSize = limit
	}
	if limit < bidSize {
		bidSize = limit
	}

	marketData := NewL2MarketData(askSize, bidSize)
	orderbook_.FillAsks(askSize, marketData)
	orderbook_.FillBids(bidSize, marketData)

	return marketData
}

func PublishL2MarketDataSnapshot(orderbook_ _OrderBook, marketData *L2MarketData) {
	orderbook_.FillAsks(_l2Size, marketData)
	orderbook_.FillBids(_l2Size, marketData)
}
