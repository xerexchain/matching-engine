package position

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/math"
	"github.com/xerexchain/matching-engine/order"
	riskengine "github.com/xerexchain/matching-engine/processor/risk_engine"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/symbol"
)

type Margin struct {
	userID       int64
	symbolID     int32
	currency     int32 // TODO currency vs symbolID?
	openQuantity int64 // TODO doc
	openPriceSum int64 // TODO doc // TODO break to openPrice and OpenSum
	profit       int64 // TODO doc
	direction    Direction

	// pending orders total quantity
	// increment before sending order to matching engine
	// decrement after receiving trade confirmation from matching engine
	pendingSellQuantity int64
	pendingBuyQuantity  int64
	_                   struct{}
}

func NewMargin(
	userID int64,
	symbolID int32,
	currency int32,
) *Margin {
	return &Margin{
		userID:    userID,
		symbolID:  symbolID,
		currency:  currency,
		direction: _empty,
	}
}

func (m *Margin) SetUserID(id int64) {
	m.userID = id
}

// Check if position is empty (no pending orders, no open trades) - can remove it from hashmap
func (m *Margin) IsEmpty() bool {
	return m.direction == _empty &&
		m.pendingSellQuantity == 0 &&
		m.pendingBuyQuantity == 0
}

func (m *Margin) PendingHold(action order.Action, quantity int64) {
	if action == order.Ask {
		m.pendingSellQuantity += quantity
	} else {
		m.pendingBuyQuantity += quantity
	}

	// TODO handle overflow
}

func (m *Margin) PendingRelease(action order.Action, quantity int64) {
	if action == order.Ask {
		m.pendingSellQuantity -= quantity
	} else {
		m.pendingBuyQuantity -= quantity
	}

	// TODO check for negative values
}

// TODO relation of `symbol_` and `symbolID`
func (m *Margin) EstimateProfit(
	symbol_ symbol.FutureContract,
	rec riskengine.LastPriceCacheRecord, // TODO rename
) int64 {
	switch m.direction {
	case _empty:
		return m.profit
	case _long:
		{
			profit := m.profit

			if rec != nil && rec.BidPrice() != 0 {
				profit += m.openQuantity*rec.BidPrice() - m.openPriceSum
			} else {
				// unknown price - no liquidity - require extra margin
				profit += symbol_.MarginBuy() * m.openQuantity
			}

			return profit
		}
	case _short:
		{
			profit := m.profit

			if rec != nil && rec.AskPrice() != math.MaxInt64 {
				profit += m.openPriceSum - m.openQuantity*rec.AskPrice()
			} else {
				// unknown price - no liquidity - require extra margin
				profit += symbol_.MarginSell() * m.openQuantity
			}

			return profit
		}
	default:
		// not possible state
		// TODO panic?
		return 0
	}
}

// TODO rename
func (m *Margin) marginBuymarginSell(
	symbol_ symbol.FutureContract,
) (int64, int64) {
	signedPosition := m.openQuantity * int64(m.direction)
	currRiskBuyQuantity := m.pendingBuyQuantity + signedPosition
	currRiskSellQuantity := m.pendingSellQuantity - signedPosition

	marginBuy := currRiskBuyQuantity * symbol_.MarginBuy()
	MarginSell := currRiskSellQuantity * symbol_.MarginSell()

	return marginBuy, MarginSell
}

func (m *Margin) CalculateRequiredMarginForFutures(
	symbol_ symbol.FutureContract,
) int64 {
	marginBuy, MarginSell := m.marginBuymarginSell(symbol_)

	return math.Max(marginBuy, MarginSell)
}

// considering extra quantity added to current position (or outstanding orders)
// return -1 if order will reduce current exposure (no additional margin required),
// otherwise full margin for symbol position if order placed/executed
func (m *Margin) CalculateRequiredMarginForOrder(
	symbol_ symbol.FutureContract,
	action order.Action,
	quantity int64,
) int64 {
	marginBuy, marginSell := m.marginBuymarginSell(symbol_)
	currMargin := math.Max(marginBuy, marginSell)

	if action == order.Bid {
		marginBuy += symbol_.MarginBuy() * quantity
	} else {
		marginSell += symbol_.MarginSell() * quantity
	}

	newMargin := math.Max(marginBuy, marginSell)

	if newMargin <= currMargin {
		return -1
	} else {
		return newMargin
	}
}

// Update position for one user
// return opened quantity
func (m *Margin) UpdateForMarginTrade(
	action order.Action,
	quantity int64,
	price int64,
) (int64, error) {
	// 1. Un-hold pending quantity
	m.PendingRelease(action, quantity)

	// 2. Reduce opposite position accordingly (if exists)
	quantityToOpen, err := m.CloseCurrentPositionFutures(action, quantity, price)

	if err != nil {
		return 0, err
	}

	// 3. Increase forward position accordingly (if quantity left in the trading event)
	if quantityToOpen > 0 {
		if err := m.OpenPositionMargin(action, quantityToOpen, price); err != nil {
			return 0, err
		}
	}

	return quantityToOpen, nil
}

func (m *Margin) CloseCurrentPositionFutures(
	action order.Action,
	tradeQuantity int64,
	price int64,
) (int64, error) {
	if m.direction == _empty || m.direction == directionFromAction(action) {
		// nothing to close
		return tradeQuantity, nil
	}

	if m.openQuantity > tradeQuantity {
		// current position is bigger than trade quantity - just reduce position accordingly, don't fix profit
		m.openQuantity -= tradeQuantity
		m.openPriceSum -= tradeQuantity * price
		return 0, nil
	}

	// current position smaller than trade quantity, can close completely and calculate profit
	m.profit += (m.openQuantity*price - m.openPriceSum) * int64(m.direction)
	m.openPriceSum = 0
	m.direction = _empty
	quantityToOpen := tradeQuantity - m.openQuantity
	m.openQuantity = 0

	// TODO comment or not?
	if err := m.ValidateInternalState(); err != nil {
		return 0, err
	}

	return quantityToOpen, nil
}

func (m *Margin) OpenPositionMargin(
	action order.Action,
	quantityToOpen int64,
	price int64,
) error {
	m.openQuantity += quantityToOpen
	m.openPriceSum += quantityToOpen * price
	m.direction = directionFromAction(action)

	// TODO comment or not?
	if err := m.ValidateInternalState(); err != nil {
		return err
	}

	return nil
}

func (m *Margin) Reset() {
	m.pendingBuyQuantity = 0
	m.pendingSellQuantity = 0
	m.openQuantity = 0
	m.openPriceSum = 0
	m.direction = _empty
}

func (m *Margin) ValidateInternalState() error {
	if m.direction == _empty && (m.openQuantity != 0 || m.openPriceSum != 0) {
		const msg = "margin: userId %v, position %v, totalQuantity %v, openPriceSum %v"

		return fmt.Errorf(
			msg,
			m.userID,
			m.direction,
			m.openQuantity,
			m.openPriceSum,
		)
	}

	if m.direction == _empty && (m.openQuantity <= 0 || m.openPriceSum <= 0) {
		const msg = "margin: userId %v, position %v, totalQuantity %v, openPriceSum %v"

		return fmt.Errorf(
			msg,
			m.userID,
			m.direction,
			m.openQuantity,
			m.openPriceSum,
		)
	}

	if m.pendingSellQuantity < 0 || m.pendingBuyQuantity < 0 {
		const msg = "margin: userId %v, pendingSellQuantity %v, pendingBuyQuantity %v"

		return fmt.Errorf(
			msg,
			m.userID,
			m.pendingSellQuantity,
			m.pendingBuyQuantity,
		)
	}

	return nil
}

// TODO fields are unexported
// TODO remove panic?
func (m *Margin) Hash() uint64 {
	hash, err := hashstructure.Hash(*m, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

// TODO incompatible with exchange-core
func (m *Margin) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(m.userID, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(m.symbolID, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(m.currency, out); err != nil {
		return err
	}

	if err := serialization.WriteInt8(int8(m.direction), out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.openQuantity, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.openPriceSum, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.profit, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.pendingSellQuantity, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.pendingBuyQuantity, out); err != nil {
		return err
	}

	return nil
}

// TODO incompatible with exchange-core
func (m *Margin) Unmarshal(in *bytes.Buffer) error {
	userID, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	symbolID, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	currency, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	code, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	direction, ok := directionFromCode(code)

	if !ok {
		return fmt.Errorf("Margin unmarshal: direction: %v", code)
	}

	openQuantity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	openPriceSum, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	profit, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	pendingSellQuantity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	pendingBuyQuantity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	m.userID = userID
	m.symbolID = symbolID
	m.currency = currency
	m.direction = direction
	m.openQuantity = openQuantity
	m.openPriceSum = openPriceSum
	m.profit = profit
	m.pendingSellQuantity = pendingSellQuantity
	m.pendingBuyQuantity = pendingBuyQuantity

	return nil
}

func UnmarshalMargins(in *bytes.Buffer) (map[int32]*Margin, error) {
	size, err := serialization.ReadInt32(in)

	if err != nil {
		return nil, err
	}

	positions := make(map[int32]*Margin, size)

	for ; size > 0; size-- {
		symbolID, err := serialization.ReadInt32(in)

		if err != nil {
			return nil, err
		}

		margin := &Margin{}

		if err := margin.Unmarshal(in); err != nil {
			return nil, err
		}

		positions[symbolID] = margin
	}

	return positions, nil
}
