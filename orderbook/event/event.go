package event

import "github.com/xerexchain/matching-engine/order"

// TODO move activeOrderCompleted, section into the order?
// TODO REDUCE needs remaining size (can write into size), bidderHoldPrice - can write into price
// TODO REJECT needs remaining size (can not write into size),

// TODO equals and hashCode overriden
type Event interface {
	Next() Event
	SetNext(next Event)
	FindTail() Event
	ChainSize() int32
}

// TODO equals and hashCode overriden
// Can be triggered by place ORDER or for MOVE order command.
type TradeEvent interface {
	Event
}

// TODO equals and hashCode overriden
// After reduce order - risk engine should unlock deposit accordingly
type ReduceEvent interface {
	Event
}

// TODO equals and hashCode overriden
// Can happen only when MARKET order has to be rejected by Matcher Engine due lack of liquidity
// That basically means no ASK (or BID) orders left in the order book for any price.
// Before being rejected active order can be partially filled.
type RejectEvent interface {
	Event
}

// TODO equals and hashCode overriden
// Custom binary data attached
type BinaryEvent interface {
	Event
}

// TODO redundant fields?
type tradeEvent struct {
	makerOrderId        int64
	makerUserId         int64
	makerOrderCompleted bool
	takerOrderCompleted bool
	price               int64 // actual price of the deal (from maker order)
	tradedQuantity      int64 // traded quantity, transfered from maker to taker
	bidderHoldPrice     int64 // frozen price from BID order owner (depends on activeOrderAction) // TODO doc
	next                Event
	_                   struct{}
}

// TODO redundant fields?
type reduceEvent struct {
	makerOrderId        int64
	makerOrderCompleted bool
	price               int64
	reduceQuantity     int64
	action              order.Action
	next                Event
	_                   struct{}
}

// TODO redundant fields?
type rejectEvent struct {
	takerOrderId     int64
	price            int64
	rejectedQuantity int64
	action           order.Action
	next             Event
	_                struct{}
}

// TODO complete impl
// TODO redundant fields?
type binaryEvent struct {
	_ struct{}
}

func findTail(e Event) Event {
	for e.Next() != nil {
		e = e.Next()
	}

	return e
}

func chainSize(e Event) int32 {
	var size int32 = 1

	for e.Next() != nil {
		e = e.Next()
		size++
	}

	return size
}

func (t *tradeEvent) Next() Event {
	return t.next
}

func (t *tradeEvent) SetNext(next Event) {
	t.next = next
}

func (t *tradeEvent) FindTail() Event {
	return findTail(t)
}

func (t *tradeEvent) ChainSize() int32 {
	return chainSize(t)
}

func (r *rejectEvent) Next() Event {
	return r.next
}

func (r *rejectEvent) SetNext(next Event) {
	r.next = next
}

func (r *rejectEvent) FindTail() Event {
	return findTail(r)
}

func (r *rejectEvent) ChainSize() int32 {
	return chainSize(r)
}

func (r *reduceEvent) Next() Event {
	return r.next
}

func (r *reduceEvent) SetNext(next Event) {
	r.next = next
}

func (r *reduceEvent) FindTail() Event {
	return findTail(r)
}

func (r *reduceEvent) ChainSize() int32 {
	return chainSize(r)
}

// TODO unused?
func CreateTradeEventChain(chainSize int32) TradeEvent {
	head := &tradeEvent{}
	prev := head

	for chainSize > 1 {
		next := &tradeEvent{}
		prev.SetNext(next)
		prev = next
		chainSize--
	}

	return head
}

func PrependRejectEvent(
	to Event,
	orderId int64,
	price int64,
	rejectedQuantity int64,
	action order.Action,
) RejectEvent {
	rejectEvent := NewRejectEvent(
		orderId,
		price,
		rejectedQuantity,
		action,
	)

	rejectEvent.SetNext(to)

	return rejectEvent
}

func NewTradeEvent(
	makerOrderId int64,
	makerUserId int64,
	makerOrderCompleted bool,
	takerOrderCompleted bool,
	price int64,
	tradedQuantity int64,
	bidderHoldPrice int64,
) TradeEvent {
	return &tradeEvent{
		makerOrderId:        makerOrderId,
		makerUserId:         makerUserId,
		makerOrderCompleted: makerOrderCompleted,
		takerOrderCompleted: takerOrderCompleted,
		price:               price,
		tradedQuantity:      tradedQuantity,
		bidderHoldPrice:     bidderHoldPrice,
	}
}

func NewReduceEvent(
	makerOrderId int64,
	makerOrderCompleted bool,
	price int64,
	reduceQuantity int64,
	action order.Action,
) ReduceEvent {
	return &reduceEvent{
		makerOrderId:        makerOrderId,
		makerOrderCompleted: makerOrderCompleted,
		price:               price,
		reduceQuantity:     reduceQuantity,
		action:              action,
	}
}

func NewRejectEvent(
	takerOrderId int64,
	price int64,
	rejectedQuantity int64,
	action order.Action,
) RejectEvent {
	return &rejectEvent{
		takerOrderId:     takerOrderId,
		price:            price,
		rejectedQuantity: rejectedQuantity,
		action:           action,
	}
}
