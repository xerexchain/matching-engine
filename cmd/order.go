package cmd

import (
	"github.com/xerexchain/matching-engine/cmd/result_code"
	"github.com/xerexchain/matching-engine/market"
	"github.com/xerexchain/matching-engine/matcher/event"
	"github.com/xerexchain/matching-engine/order"
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
	orderAction order.Action

	symbol int
	price  int64
	size   int64

	// new orders INPUT - reserved price for fast moves of GTC bid orders in exchange mode
	reserveBidPrice int64

	timestamp  int64
	userCookie int

	// filled by grouping processor:
	eventsGroup  int64
	serviceFlags int

	// can also be used for saving intermediate state
	resultCode resultcode.ResultCode

	tradeEvent event.TradeEvent

	marketData market.L2MarketData
}

// No removing/revoking
func (o *orderCommand) ProcessMatcherEvents(ch chan<- event.TradeEvent) {
	eve := o.tradeEvent

	for eve != nil {
		ch <- eve
		eve = eve.Next()
	}
}

func NewOrderCommand() OrderCommand {
	return &orderCommand{}
}
