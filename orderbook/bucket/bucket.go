package bucket

import (
	"bytes"
	"log"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/math"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/serialization"
)

// TODO functions thread safety, calls are subject to race condition.
// TODO Comparable<OrdersBucketNaive>, compareTo, hashCode, equals
// TODO move to `orderbook` package?

type MatcherResult struct {
	Head              *event.Trade
	Tail              *event.Trade
	CollectedQuantity int64
	RemovedOrders     []int64
	_                 struct{}
}

type Bucket struct {
	price int64

	// FIX This field imposes side effects on functions.
	totalQuantity int64
	orders        *linkedhashmap.Map
	_             struct{}
}

func New(price int64) *Bucket {
	return &Bucket{
		price:  price,
		orders: linkedhashmap.New(),
	}
}

// TODO rename
// Bucket with only price set must only be used for btree item comparsion
func With(price int64) *Bucket {
	return &Bucket{
		price: price,
	}
}

func (buc *Bucket) Price() int64 {
	return buc.price
}

func (buc *Bucket) TotalQuantity() int64 {
	return buc.totalQuantity
}

func (buc *Bucket) Put(ord *order.Order) {
	buc.orders.Put(ord.ID(), ord)
	buc.totalQuantity += ord.Remained()
}

func (buc *Bucket) Remove(orderID int64) {
	if ord, ok := buc.Find(orderID); ok {
		buc.Reduce(ord.Remained())
		buc.orders.Remove(orderID)
	}
}

func (buc *Bucket) Reduce(quantity int64) {
	buc.totalQuantity -= quantity
}

func (buc *Bucket) NumOrders() int32 {
	return int32(buc.orders.Size())
}

// TODO How to return `[]*order.Order` without iterating the values? (performance cost of iteration)
// TODO side effects imposed by the caller
// TODO How to preserve execution queue order?
func (buc *Bucket) AllOrders() []interface{} {
	return buc.orders.Values()
}

// TODO side effects imposed by the caller
func (buc *Bucket) Find(orderID int64) (*order.Order, bool) {
	if v, ok := buc.orders.Get(orderID); ok {
		return v.(*order.Order), true
	}

	return nil, false
}

// TODO side effects imposed by the caller
func (buc *Bucket) ForEachOrder(f func(*order.Order)) {
	for _, v := range buc.AllOrders() {
		ord := v.(*order.Order)
		f(ord)
	}
}

func (buc *Bucket) IsValid() bool {
	sum := int64(0)

	accumulator := func(ord *order.Order) {
		sum += ord.Remained()
	}

	buc.ForEachOrder(accumulator)

	return sum == buc.totalQuantity
}

func (buc *Bucket) Match(
	toCollect int64,
	reservedBidPrice int64, // only for bids
) *MatcherResult {
	var (
		collected       int64
		removedOrders   []int64
		head, tail      *event.Trade
		bidderHoldPrice int64
	)

	for _, v := range buc.AllOrders() {
		diff := toCollect - collected

		if diff == 0 {
			break
		}

		ord := v.(*order.Order)
		tradedQuantity := math.Min(ord.Remained(), diff)

		// TODO handle the error properly
		if err := ord.Fill(tradedQuantity); err != nil {
			log.Printf("unexpected: %v", err)
			continue
		}

		buc.Reduce(tradedQuantity)
		collected += tradedQuantity

		if ord.Remained() == 0 {
			buc.Remove(ord.ID())
			removedOrders = append(removedOrders, ord.ID())
		}

		if ord.Action() == order.Ask {
			bidderHoldPrice = reservedBidPrice
		} else {
			bidderHoldPrice = ord.ReservedBidPrice()
		}

		e := event.NewTrade(
			ord.ID(),
			ord.UserID(),
			ord.Remained() == 0,
			collected == toCollect,
			ord.Price(),
			tradedQuantity,
			bidderHoldPrice,
		)

		if tail == nil {
			head = e
		} else {
			tail.SetNext(e)
		}

		tail = e
	}

	return &MatcherResult{
		Head:              head,
		Tail:              tail,
		CollectedQuantity: collected,
		RemovedOrders:     removedOrders,
	}
}

func (buc *Bucket) Less(than btree.Item) bool {
	return buc.price < than.(*Bucket).Price()
}

func (buc *Bucket) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(buc.price, out); err != nil {
		return err
	}

	size := int32(buc.orders.Size())

	if err := serialization.WriteInt32(size, out); err != nil {
		return err
	}

	// TODO handle order of appearance
	for _, k := range buc.orders.Keys() {
		v, _ := buc.orders.Get(k)

		if err := serialization.WriteInt64(k.(int64), out); err != nil {
			return err
		}

		ord := v.(*order.Order)

		if err := ord.Marshal(out); err != nil {
			return err
		}
	}

	if err := serialization.WriteInt64(buc.totalQuantity, out); err != nil {
		return err
	}

	return nil
}

func (buc *Bucket) Unmarshal(in *bytes.Buffer) error {
	price, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	size, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	orders := linkedhashmap.New()

	for ; size > 0; size-- {
		id, err := serialization.ReadInt64(in)

		if err != nil {
			return err
		}

		ord := &order.Order{}

		if err := ord.Unmarshal(in); err != nil {
			return err
		}

		orders.Put(id, ord)
	}

	totalQuantity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	buc.price = price
	buc.orders = orders
	buc.totalQuantity = totalQuantity

	return nil
}
