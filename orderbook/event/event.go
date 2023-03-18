package event

import (
	"github.com/xerexchain/matching-engine/order/action"
)

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
type Trade interface {
	Event
}

// TODO equals and hashCode overriden
// After reduce order - risk engine should unlock deposit accordingly
type Reduce interface {
	Event
}

// TODO equals and hashCode overriden
// Can happen only when MARKET order has to be rejected by Matcher Engine due lack of liquidity
// That basically means no ASK (or BID) orders left in the order book for any price.
// Before being rejected active order can be partially filled.
type Reject interface {
	Event
}

// TODO equals and hashCode overriden
// Custom binary data attached
type Binary interface {
	Event
}

// TODO redundant fields?
type trade struct {
	makerOrderId        int64
	makerUserId         int64
	makerOrderCompleted bool
	takerOrderCompleted bool
	price               int64 // actual price of the deal (from maker order)
	quantity            int64 // traded quantity, transfered from maker to taker
	bidderHoldPrice     int64 // frozen price from BID order owner (depends on activeOrderAction) // TODO doc
	next                Event
	_                   struct{}
}

// TODO redundant fields?
type reduce struct {
	makerOrderId        int64
	makerOrderCompleted bool
	price               int64
	quantity            int64 // reduced quantity
	action              action.Action
	next                Event
	_                   struct{}
}

// TODO redundant fields?
type reject struct {
	takerOrderId int64
	price        int64
	quantity     int64 // rejected quantity
	action       action.Action
	next         Event
	_            struct{}
}

// TODO complete impl
// TODO redundant fields?
type binary struct {
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

func (t *trade) Next() Event {
	return t.next
}

func (t *trade) SetNext(next Event) {
	t.next = next
}

func (t *trade) FindTail() Event {
	return findTail(t)
}

func (t *trade) ChainSize() int32 {
	return chainSize(t)
}

func (r *reject) Next() Event {
	return r.next
}

func (r *reject) SetNext(next Event) {
	r.next = next
}

func (r *reject) FindTail() Event {
	return findTail(r)
}

func (r *reject) ChainSize() int32 {
	return chainSize(r)
}

func (r *reduce) Next() Event {
	return r.next
}

func (r *reduce) SetNext(next Event) {
	r.next = next
}

func (r *reduce) FindTail() Event {
	return findTail(r)
}

func (r *reduce) ChainSize() int32 {
	return chainSize(r)
}

// TODO unused?
func CreateTradeChain(chainSize int32) Trade {
	head := &trade{}
	prev := head

	for chainSize > 1 {
		next := &trade{}
		prev.SetNext(next)
		prev = next
		chainSize--
	}

	return head
}

func PrependReject(
	to Event,
	orderId int64,
	price int64,
	quantity int64, // rejected quantity
	act action.Action,
) Reject {
	r := NewReject(
		orderId,
		price,
		quantity,
		act,
	)

	r.SetNext(to)

	return r
}

func NewTrade(
	makerOrderId int64,
	makerUserId int64,
	makerOrderCompleted bool,
	takerOrderCompleted bool,
	price int64,
	quantity int64, // traded quantity
	bidderHoldPrice int64,
) Trade {
	return &trade{
		makerOrderId:        makerOrderId,
		makerUserId:         makerUserId,
		makerOrderCompleted: makerOrderCompleted,
		takerOrderCompleted: takerOrderCompleted,
		price:               price,
		quantity:            quantity,
		bidderHoldPrice:     bidderHoldPrice,
	}
}

func NewReduce(
	makerOrderId int64,
	makerOrderCompleted bool,
	price int64,
	quantity int64, // reduced quantity
	act action.Action,
) Reduce {
	return &reduce{
		makerOrderId:        makerOrderId,
		makerOrderCompleted: makerOrderCompleted,
		price:               price,
		quantity:            quantity,
		action:              act,
	}
}

func NewReject(
	takerOrderId int64,
	price int64,
	quantity int64, // rejected quantity
	act action.Action,
) Reject {
	return &reject{
		takerOrderId: takerOrderId,
		price:        price,
		quantity:     quantity,
		action:       act,
	}
}
