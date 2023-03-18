package cmd

import (
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/order/action"
	"github.com/xerexchain/matching-engine/orderbook"
	"github.com/xerexchain/matching-engine/orderbook/event"
	resultcode "github.com/xerexchain/matching-engine/result_code"
)

type OrderCommand interface {
}

type orderCommand struct {
	t         Type
	uid       int64
	orderId   int64
	orderType order.Type

	// required for PLACE_ORDER only;
	// for CANCEL/MOVE contains original order action (filled by orderbook)
	orderAction action.Action

	symbol int32
	price  int64
	size   int64

	// new orders INPUT - reserved price for fast moves of GTC bid orders in exchange mode
	reservedBidPrice int64

	timestamp  int64
	userCookie int32

	// filled by grouping processor:
	eventsGroup  int64
	serviceFlags int32

	// can also be used for saving intermediate state
	resultCode resultcode.ResultCode

	tradeEvent event.Trade

	marketData orderbook.L2MarketData
	_          struct{}
}

// No removing/revoking
func (o *orderCommand) ProcessMatcherEvents(ch chan<- event.Trade) {
	eve := o.tradeEvent

	for eve != nil {
		ch <- eve
		eve = eve.Next()
	}
}

func NewOrderCommand() OrderCommand {
	return &orderCommand{}
}
