package event

type Type int8

const (
	// Trade event
	// Can be triggered by place ORDER or for MOVE order command.
	Trade Type = iota

	// Reject event
	// Can happen only when MARKET order has to be rejected by Matcher Engine due lack of liquidity
	// That basically means no ASK (or BID) orders left in the order book for any price.
	// Before being rejected active order can be partially filled.
	Reject

	// After cancel/reduce order - risk engine should unlock deposit accordingly
	Reduce

	// Custom binary data attached
	BinaryEvent
)
