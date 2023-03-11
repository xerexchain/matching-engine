package orderbook

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
// After cancel order - risk engine should unlock deposit accordingly
type CancelEvent interface {
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

// TODO equals and hashCode overriden
type tradeEvent struct {
	makerOrderId        int64
	makerUserId         int64 // TODO is it redundant?
	makerOrderCompleted bool  // TODO is it redundant?
	takerOrderCompleted bool
	tradedPrice         int64 // actual price of the deal (from maker order)
	tradedQuantity      int64 // traded quantity, transfered from maker to taker
	bidderHoldPrice     int64 // frozen price from BID order owner (depends on activeOrderAction) // TODO doc
	next                TradeEvent
	_                   struct{}
}

// TODO equals and hashCode overriden
// TODO complete impl
type reduceEvent struct {
	_ struct{}
}

// TODO equals and hashCode overriden
// TODO complete impl
type cancelEvent struct {
	_ struct{}
}

// TODO equals and hashCode overriden
// TODO complete impl
type rejectEvent struct {
	_ struct{}
}

// TODO equals and hashCode overriden
// TODO complete impl
type binaryEvent struct {
	_ struct{}
}

func (t *tradeEvent) Next() Event {
	return t.next
}

func (t *tradeEvent) SetNext(next Event) {
	t.next = next
}

func (t *tradeEvent) FindTail() Event {
	var tail TradeEvent = t

	for tail.Next() != nil {
		tail = tail.Next()
	}

	return tail
}

func (t *tradeEvent) ChainSize() int32 {
	var size int32 = 1
	var tail TradeEvent = t

	for tail.Next() != nil {
		tail = tail.Next()
		size++
	}

	return size
}

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

func NewTradeEvent(
	makerOrderId int64,
	makerUserId int64,
	makerOrderCompleted bool,
	takerOrderCompleted bool,
	tradedPrice int64,
	tradedQuantity int64,
	bidderHoldPrice int64,
) TradeEvent {
	return &tradeEvent{
		makerOrderId:        makerOrderId,
		makerUserId:         makerUserId,
		tradedPrice:         tradedPrice,
		makerOrderCompleted: makerOrderCompleted,
		takerOrderCompleted: takerOrderCompleted,
		tradedQuantity:      tradedQuantity,
		bidderHoldPrice:     bidderHoldPrice,
	}
}
