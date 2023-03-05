package event

import (
	"encoding/json"

	"github.com/mitchellh/hashstructure/v2"
)

// TODO move activeOrderCompleted, eventType, section into the order?
// TODO REDUCE needs remaining size (can write into size), bidderHoldPrice - can write into price
// TODO REJECT needs remaining size (can not write into size),

type TradeEvent interface {
	Next() TradeEvent
	SetNext(next TradeEvent)
	FindTail() TradeEvent
	ChainSize() int
	HashCode() uint64
	String() string
}

type tradeEvent struct {
	Type    `hash:"ignore"`
	Section int

	// TODO join (requires 11+ bits)
	// false, except when activeOrder is completely filled, removed or rejected
	// it is always true for REJECT event
	// it is true for REDUCE event if reduce was triggered by COMMAND
	ActiveOrderCompleted bool

	// maker (for TRADE event type only)
	MatchedOrderID int64
	// 0 for rejection
	MatchedOrderUID int64
	// false, except when matchedOrder is completely filled
	MatchedOrderCompleted bool

	// actual Pice of the deal (from maker order), 0 for rejection (Pice can be taken from original order)
	Price int64

	// TRADE - trade Size
	// REDUCE - effective reduce Size of REDUCE command, or not filled Size for CANCEL command
	// REJECT - unmatched Size of rejected order
	Size int64

	// frozen price from BID order owner (depends on activeOrderAction)
	BbidderHoldPrice int64

	Nxt TradeEvent
}

func (t *tradeEvent) Next() TradeEvent {
	return t.Nxt
}

func (t *tradeEvent) SetNext(next TradeEvent) {
	t.Nxt = next
}

func (t *tradeEvent) FindTail() TradeEvent {
	var tail TradeEvent = t

	for tail.Next() != nil {
		tail = tail.Next()
	}

	return tail
}

func (t *tradeEvent) ChainSize() int {
	size := 1
	var tail TradeEvent = t

	for tail.Next() != nil {
		tail = tail.Next()
		size++
	}

	return size
}

func (t *tradeEvent) HashCode() uint64 {
	hash, err := hashstructure.Hash(t, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (t *tradeEvent) String() string {
	out, _ := json.Marshal(t)

	return string(out)
}

func CreateEventChian(chainSize int) TradeEvent {
	head := NewTradeEvent()
	prev := head

	for chainSize > 1 {
		next := NewTradeEvent()
		prev.SetNext(next)
		prev = next
		chainSize--
	}

	return head
}

func NewTradeEvent() TradeEvent {
	return &tradeEvent{}
}
