package orderbucket

import (
	"bytes"
	"fmt"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/math"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/order/action"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/serialization"
)

// TODO functions thread safty, calls are subject to race condition
// TODO Comparable<OrdersBucketNaive>, compareTo, hashCode, equals
type Naive interface {
	btree.Item
	serialization.Marshalable
	Price() int64
	Put(order.Order)
	Remove(int64)
	Reduce(int64)
	NumOrders() int32
	AllOrders() []interface{}
	Find(int64) (order.Order, bool)
	ForEachOrder(func(order.Order))
	TotalQuantity() int64
	Validate()
	Match(int64, int64) *MatcherResult
}

type naive struct {
	price         int64
	totalQuantity int64 // FIX This field imposes side effects on functions.
	orders        *linkedhashmap.Map
	_             struct{}
}

type MatcherResult struct {
	EventHead         event.TradeEvent
	EventTail         event.TradeEvent
	CollectedQuantity int64
	RemovedOrders     []int64
	_                 struct{}
}

func (n *naive) Price() int64 {
	return n.price
}

func (n *naive) Put(ord order.Order) {
	id := ord.Id()
	n.orders.Put(id, ord)
	n.totalQuantity += ord.Remained()
}

func (n *naive) Remove(orderId int64) {
	if ord, ok := n.Find(orderId); ok {
		n.Reduce(ord.Remained())
		n.orders.Remove(orderId)
	}
}

func (n *naive) Reduce(
	quantity int64,
) {
	n.totalQuantity -= quantity
}

func (n *naive) NumOrders() int32 {
	return int32(n.orders.Size())
}

// TODO How to return `[]order.Order` without iterating the values? (performance cost of iteration)
// TODO side effects imposed by the caller
// preserving execution queue order
func (n *naive) AllOrders() []interface{} {
	return n.orders.Values()
}

// TODO side effects imposed by the caller
func (n *naive) Find(orderId int64) (order.Order, bool) {
	if val, ok := n.orders.Get(orderId); ok {
		return val.(order.Order), true
	}

	return nil, false
}

// TODO side effects imposed by the caller
func (n *naive) ForEachOrder(f func(order.Order)) {
	for _, v := range n.AllOrders() {
		ord := v.(order.Order)
		f(ord)
	}
}

func (n *naive) TotalQuantity() int64 {
	return n.totalQuantity
}

func (n *naive) Validate() {
	sum := int64(0)

	accumulator := func(ord order.Order) {
		sum += ord.Remained()
	}

	n.ForEachOrder(accumulator)

	if sum != n.totalQuantity {
		panic(
			fmt.Sprintf(
				"bucket=%v totalQuantity=%v calculated=%v",
				n.price,
				n.totalQuantity,
				sum,
			),
		)
	}
}

func (n *naive) Match(
	toCollect int64,
	reservedBidPrice int64, // only for bids
) *MatcherResult {
	collected := int64(0)
	removedOrders := []int64{}
	var head, tail event.TradeEvent
	var bidderHoldPrice int64

	for _, v := range n.AllOrders() {
		diff := toCollect - collected

		if diff == 0 {
			break
		}

		ord := v.(order.Order)
		tradedQuantity := math.Min(ord.Remained(), diff)
		ord.Fill(tradedQuantity)
		n.Reduce(tradedQuantity)
		collected += tradedQuantity

		if ord.Remained() == 0 {
			n.Remove(ord.Id())
			removedOrders = append(removedOrders, ord.Id())
		}

		if ord.Action() == action.Ask {
			bidderHoldPrice = reservedBidPrice
		} else {
			bidderHoldPrice = ord.ReservedBidPrice()
		}

		tradeEvent := event.NewTradeEvent(
			ord.Id(),
			ord.UserId(),
			ord.Remained() == 0,
			collected == toCollect,
			ord.Price(),
			tradedQuantity,
			bidderHoldPrice,
		)

		if tail == nil {
			head = tradeEvent
		} else {
			tail.SetNext(tradeEvent)
		}

		tail = tradeEvent
	}

	return &MatcherResult{
		EventHead:         head,
		EventTail:         tail,
		CollectedQuantity: collected,
		RemovedOrders:     removedOrders,
	}
}

func (n *naive) Less(than btree.Item) bool {
	return n.price < than.(Naive).Price()
}

func (n *naive) Marshal(out *bytes.Buffer) error {
	return MarshalNaive(n, out)
}

func MarshalNaive(
	in interface{},
	out *bytes.Buffer,
) error {
	n := in.(*naive)

	if err := serialization.MarshalInt64(n.price, out); err != nil {
		return err
	}

	if err := serialization.MarshalLinkedHashMap(
		n.orders,
		out,
		serialization.MarshalInt64,
		order.Marshal,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(n.totalQuantity, out); err != nil {
		return err
	}

	return nil
}

func UnmarshalNaive(
	b *bytes.Buffer,
) (interface{}, error) {
	n := naive{}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		n.price = val.(int64)
	}

	if orders, err := serialization.UnmarshalLinkedHashMap(
		b,
		serialization.UnmarshalInt64,
		order.UnMarshal,
	); err != nil {
		return nil, err
	} else {
		n.orders = orders
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		n.totalQuantity = val.(int64)
	}

	return &n, nil
}

func NewNaive(price int64) Naive {
	return &naive{
		price:  price,
		orders: linkedhashmap.New(),
	}
}

func NewDumpNaive(price int64) Naive {
	return &naive{
		price: price,
	}
}
