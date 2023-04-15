package event

import (
	"github.com/xerexchain/matching-engine/order"
)

// TODO move activeOrderCompleted, section into the order?
// TODO REDUCE needs remaining size (can write into size), bidderHoldPrice - can write into price
// TODO REJECT needs remaining size (can not write into size),

// TODO equals and hashCode overriden
type Event interface {
	Next() Event
	SetNext(Event)
	FindTail() Event
	ChainSize() int32
}

// TODO equals and hashCode overriden
// TODO redundant fields?
// TODO rename
// Can be triggered by place ORDER or for MOVE order command.
type Trade struct {
	makerOrderID        int64
	makerUserID         int64
	makerOrderCompleted bool
	takerOrderCompleted bool

	// actual price of the deal (from maker order)
	price int64

	// traded quantity, transfered from maker to taker
	quantity int64

	// frozen price from BID order owner (depends on activeOrderAction) // TODO logic
	bidderHoldPrice int64
	next            Event
	_               struct{}
}

func NewTrade(
	makerOrderID int64,
	makerUserID int64,
	makerOrderCompleted bool,
	takerOrderCompleted bool,
	price int64,
	quantity int64, // traded quantity
	bidderHoldPrice int64,
) *Trade {
	return &Trade{
		makerOrderID:        makerOrderID,
		makerUserID:         makerUserID,
		makerOrderCompleted: makerOrderCompleted,
		takerOrderCompleted: takerOrderCompleted,
		price:               price,
		quantity:            quantity,
		bidderHoldPrice:     bidderHoldPrice,
	}
}

func (t *Trade) Next() Event {
	return t.next
}

func (t *Trade) SetNext(next Event) {
	t.next = next
}

func (t *Trade) FindTail() Event {
	return findTail(t)
}

func (t *Trade) ChainSize() int32 {
	return chainSize(t)
}

// TODO equals and hashCode overriden
// TODO redundant fields?
// TODO rename
// After reduce order - risk engine should unlock deposit accordingly.
type Reduce struct {
	makerOrderID        int64
	makerOrderCompleted bool
	price               int64

	// reduced quantity
	quantity int64
	action   order.Action
	next     Event
	_        struct{}
}

func NewReduce(
	makerOrderID int64,
	makerOrderCompleted bool,
	price int64,
	quantity int64, // reduced quantity
	action order.Action,
) *Reduce {
	return &Reduce{
		makerOrderID:        makerOrderID,
		makerOrderCompleted: makerOrderCompleted,
		price:               price,
		quantity:            quantity,
		action:              action,
	}
}

func (r *Reduce) Next() Event {
	return r.next
}

func (r *Reduce) SetNext(next Event) {
	r.next = next
}

func (r *Reduce) FindTail() Event {
	return findTail(r)
}

func (r *Reduce) ChainSize() int32 {
	return chainSize(r)
}

// TODO equals and hashCode overriden
// TODO redundant fields?
// TODO rename
// Can happen only when MARKET order has to be rejected by Matcher Engine due lack of liquidity.
// That basically means no ASK (or BID) orders left in the order book for any price.
// Before being rejected active order can be partially filled.
type Reject struct {
	takerOrderID int64
	price        int64
	quantity     int64 // rejected quantity
	action       order.Action
	next         Event
	_            struct{}
}

func NewReject(
	takerOrderID int64,
	price int64,
	quantity int64, // rejected quantity
	action order.Action,
) *Reject {
	return &Reject{
		takerOrderID: takerOrderID,
		price:        price,
		quantity:     quantity,
		action:       action,
	}
}

func (r *Reject) Next() Event {
	return r.next
}

func (r *Reject) SetNext(next Event) {
	r.next = next
}

func (r *Reject) FindTail() Event {
	return findTail(r)
}

func (r *Reject) ChainSize() int32 {
	return chainSize(r)
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
