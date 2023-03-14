package bucket

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/serialization"
)

// TODO functions thread safty, calls are subject to race condition
// TODO Comparable<OrdersBucketNaive>, compareTo, hashCode, equals
type NaiveOrderBucket interface {
	btree.Item
	serialization.Marshalable
	Price() int64
	Put(order.Order)
	Remove(orderId int64)
	NumOrders() int32
	AllOrders() []interface{}
	FindOrder(orderId int64) (order.Order, bool)
	ForEachOrder(func(order.Order))
	TotalQuantity() int64
	Validate()
	Match(
		toCollect int64,
		reserveBidPrice int64,
	) MatcherResult
}

type MatcherResult interface {
	EventHead() event.TradeEvent
	EventTail() event.TradeEvent
	CollectedQuantity() int64
	RemovedOrders() []int64
}

// TODO Comparable<OrdersBucketNaive>, compareTo, hashCode, equals
type naiveOrderBucket struct {
	price         int64
	totalQuantity int64 // FIX because of this field, functions have side effects.
	orders        *linkedhashmap.Map
	_             struct{}
}

type matcherResult struct {
	head              event.TradeEvent
	tail              event.TradeEvent
	collectedQuantity int64
	removedOrders     []int64
	_                 struct{}
}

func (m *matcherResult) EventHead() event.TradeEvent {
	return m.head
}

func (m *matcherResult) EventTail() event.TradeEvent {
	return m.tail
}

func (m *matcherResult) CollectedQuantity() int64 {
	return m.collectedQuantity
}

func (m *matcherResult) RemovedOrders() []int64 {
	return m.removedOrders
}

func (n *naiveOrderBucket) Price() int64 {
	return n.price
}

func (n *naiveOrderBucket) Put(ord order.Order) {
	id := ord.Id()
	n.orders.Put(id, ord)
	n.totalQuantity += ord.Remained()
}

func (n *naiveOrderBucket) Remove(orderId int64) {
	val, ok := n.orders.Get(orderId)

	if !ok {
		return
	}

	ord := val.(order.Order)

	n.totalQuantity -= ord.Remained()
	n.orders.Remove(orderId)
}

func (n *naiveOrderBucket) NumOrders() int32 {
	return int32(n.orders.Size())
}

// TODO How to return `[]order.Order` without iterating the values? (performance cost of iteration)
// TODO side effects imposed by the caller
func (n *naiveOrderBucket) AllOrders() []interface{} {
	return n.orders.Values()
}

// TODO side effects imposed by the caller
func (n *naiveOrderBucket) FindOrder(orderId int64) (order.Order, bool) {
	val, ok := n.orders.Get(orderId)

	if !ok {
		return nil, false
	}

	return val.(order.Order), true
}

// TODO side effects imposed by the caller
func (n *naiveOrderBucket) ForEachOrder(f func(order.Order)) {
	for _, v := range n.orders.Values() {
		order := v.(order.Order)
		f(order)
	}
}

func (n *naiveOrderBucket) TotalQuantity() int64 {
	return n.totalQuantity
}

func (n *naiveOrderBucket) Validate() {
	sum := int64(0)

	for _, v := range n.orders.Values() {
		ord := v.(order.Order)
		sum += ord.Remained()
	}

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

func (n *naiveOrderBucket) Match(
	toCollect int64,
	reserveBidPrice int64, // only for bids
) MatcherResult {
	collected := int64(0)
	removedOrders := []int64{}
	var head, tail event.TradeEvent
	var bidderHoldPrice int64

	for _, orderId := range n.orders.Keys() {
		diff := toCollect - collected

		if diff == 0 {
			break
		}

		var tradedQuantity int64
		var fullMatch bool

		val, _ := n.orders.Get(orderId)
		ord := val.(order.Order)
		rem := ord.Remained()

		if rem <= diff {
			ord.Fill(rem)
			n.Remove(ord.Id())
			n.totalQuantity -= rem
			collected += rem
			tradedQuantity = rem
			fullMatch = true
			removedOrders = append(removedOrders, ord.Id())
		} else {
			ord.Fill(diff)
			n.totalQuantity -= diff
			collected += diff
			tradedQuantity = diff
			fullMatch = false
		}

		if ord.Action() == order.Ask {
			bidderHoldPrice = reserveBidPrice
		} else {
			bidderHoldPrice = ord.ReservedBidPrice()
		}

		tradeEvent := event.NewTradeEvent(
			ord.Id(),
			ord.UserId(),
			fullMatch,
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

	return &matcherResult{
		head:              head,
		tail:              tail,
		collectedQuantity: collected,
		removedOrders:     removedOrders,
	}
}

func (n *naiveOrderBucket) Less(than btree.Item) bool {
	return n.price < than.(NaiveOrderBucket).Price()
}

func (n *naiveOrderBucket) Marshal(out *bytes.Buffer) error {
	return MarshalNaiveOrderBucket(n, out)
}

func MarshalNaiveOrderBucket(
	in interface{},
	out *bytes.Buffer,
) error {
	n := in.(*naiveOrderBucket)

	if err := binary.Write(out, binary.LittleEndian, n.price); err != nil {
		return err
	}

	err := serialization.MarshalInt64InterfaceLinkedHashMap(
		n.orders,
		out,
		order.MarshalOrder,
	)

	if err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, n.totalQuantity); err != nil {
		return err
	}

	return nil
}

func UnmarshalNaiveOrderBucket(
	in *bytes.Buffer,
) (interface{}, error) {
	n := naiveOrderBucket{}

	if err := binary.Read(in, binary.LittleEndian, &(n.price)); err != nil {
		return nil, err
	}

	orders, err := serialization.UnmarshalInt64InterfaceLinkedHashMap(
		in,
		order.UnMarshalOrder,
	)

	if err != nil {
		return nil, err
	}

	n.orders = orders.(*linkedhashmap.Map)

	if err := binary.Read(in, binary.LittleEndian, &(n.totalQuantity)); err != nil {
		return nil, err
	}

	return &n, nil
}

func NewNaiveOrderBucket(price int64) NaiveOrderBucket {
	return &naiveOrderBucket{
		price:  price,
		orders: linkedhashmap.New(),
	}
}

func NewDumpNaiveOrderBucket(price int64) NaiveOrderBucket {
	return &naiveOrderBucket{
		price: price,
	}
}
