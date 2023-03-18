package position

import (
	"bytes"
	"log"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/math"
	"github.com/xerexchain/matching-engine/order/action"
	riskengine "github.com/xerexchain/matching-engine/process/risk_engine"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
	"github.com/xerexchain/matching-engine/symbol"
)

var (
	Long = Direction{
		Multiplier: 1,
	}
	Short = Direction{
		Multiplier: -1,
	}
	Empty = Direction{
		Multiplier: 0,
	}
)

type Direction struct {
	Multiplier int8
	_          struct{}
}

type Margin interface {
	state.Hashable
	serialization.Marshalable
	SetUserId(int64)
	IsEmpty() bool
	PendingHold(action.Action, int64)
	PendingRelease(action.Action, int64)
	EstimateProfit(
		symbol.FutureContractSymbol,
		riskengine.LastPriceCacheRecord,
	) int64
	CalculateRequiredMarginForFutures(
		symbol.FutureContractSymbol,
	) int64
	CalculateRequiredMarginForOrder(
		symbol.FutureContractSymbol,
		action.Action,
		int64,
	) int64
	UpdateForMarginTrade(
		action.Action,
		int64,
		int64,
	) int64
	Reset()
	ValidateInternalState()
}

type margin struct {
	UserId       int64
	SymbolId     int32
	Currency     int32 // TODO currency vs symbolId?
	OpenQuantity int64 // TODO doc
	OpenPriceSum int64 // TODO doc // TODO break to openPrice and OpenSum
	Profit       int64 // TODO doc
	Direction

	// pending orders total quantity
	// increment before sending order to matching engine
	// decrement after receiving trade confirmation from matching engine
	PendingSellQuantity int64
	PendingBuyQuantity  int64
	_                   struct{}
}

func (d Direction) IsOppositeTo(act action.Action) bool {
	return (d == Long && act == action.Ask) || (d == Short && act == action.Bid)
}

func (d Direction) IsSameAs(act action.Action) bool {
	return (d == Long && act == action.Bid) || (d == Short && act == action.Ask)
}

func DirectionFromAction(act action.Action) Direction {
	if act == action.Bid {
		return Long
	} else {
		return Short
	}
}

func DirectionFromByte(b int8) Direction {
	switch b {
	case Long.Multiplier:
		return Long
	case Short.Multiplier:
		return Short
	case Empty.Multiplier:
		return Empty
	default:
		panic("Undefined direction")
	}
}

func (m *margin) SetUserId(id int64) {
	m.UserId = id
}

// Check if position is empty (no pending orders, no open trades) - can remove it from hashmap
func (m *margin) IsEmpty() bool {
	return m.Direction == Empty && m.PendingSellQuantity == 0 && m.PendingBuyQuantity == 0
}

func (m *margin) PendingHold(act action.Action, quantity int64) {
	if act == action.Ask {
		m.PendingSellQuantity += quantity
	} else {
		m.PendingBuyQuantity += quantity
	}

	// TODO handle overflow
}

func (m *margin) PendingRelease(act action.Action, quantity int64) {
	if act == action.Ask {
		m.PendingSellQuantity -= quantity
	} else {
		m.PendingBuyQuantity -= quantity
	}

	// TODO check for negative values
}

// TODO relation of sym and symbolId
func (m *margin) EstimateProfit(
	sym symbol.FutureContractSymbol,
	rec riskengine.LastPriceCacheRecord,
) int64 {
	switch m.Direction {
	case Empty:
		return m.Profit
	case Long:
		{
			p := m.Profit

			if rec != nil && rec.BidPrice() != 0 {
				p += m.OpenQuantity*rec.BidPrice() - m.OpenPriceSum
			} else {
				// unknown price - no liquidity - require extra margin
				p += sym.MarginBuy() * m.OpenQuantity
			}

			return p
		}
	case Short:
		{
			p := m.Profit

			if rec != nil && rec.AskPrice() != math.MaxInt64 {
				p += m.OpenPriceSum - m.OpenQuantity*rec.AskPrice()
			} else {
				// unknown price - no liquidity - require extra margin
				p += sym.MarginSell() * m.OpenQuantity
			}

			return p
		}
	default:
		panic("Unknown position")
	}
}

func (m *margin) M(
	sym symbol.FutureContractSymbol,
) (int64, int64) {
	signedPosition := m.OpenQuantity * int64(m.Direction.Multiplier)
	currRiskBuyQuantity := m.PendingBuyQuantity + signedPosition
	currRiskSellQuantity := m.PendingSellQuantity - signedPosition

	marginBuy := currRiskBuyQuantity * sym.MarginBuy()
	MarginSell := currRiskSellQuantity * sym.MarginSell()

	return marginBuy, MarginSell
}

func (m *margin) CalculateRequiredMarginForFutures(
	sym symbol.FutureContractSymbol,
) int64 {
	marginBuy, MarginSell := m.M(sym)

	return math.Max(marginBuy, MarginSell)
}

// considering extra quantity added to current position (or outstanding orders)
// return -1 if order will reduce current exposure (no additional margin required),
// otherwise full margin for symbol position if order placed/executed
func (m *margin) CalculateRequiredMarginForOrder(
	sym symbol.FutureContractSymbol,
	act action.Action,
	quantity int64,
) int64 {
	marginBuy, marginSell := m.M(sym)
	currMargin := math.Max(marginBuy, marginSell)

	if act == action.Bid {
		marginBuy += sym.MarginBuy() * quantity
	} else {
		marginSell += sym.MarginSell() * quantity
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
func (m *margin) UpdateForMarginTrade(
	act action.Action,
	quantity int64,
	price int64,
) int64 {
	// 1. Un-hold pending quantity
	m.PendingRelease(act, quantity)

	// 2. Reduce opposite position accordingly (if exists)
	quantityToOpen := m.CloseCurrPositionFutures(act, quantity, price)

	// 3. Increase forward position accordingly (if quantity left in the trading event)
	if quantityToOpen > 0 {
		m.OpenPositionMargin(act, quantityToOpen, price)
	}

	return quantityToOpen
}

func (m *margin) CloseCurrPositionFutures(
	act action.Action,
	tradeQuantity int64,
	price int64,
) int64 {
	if m.Direction == Empty || m.Direction == DirectionFromAction(act) {
		// nothing to close
		return tradeQuantity
	}

	if m.OpenQuantity > tradeQuantity {
		// current position is bigger than trade quantity - just reduce position accordingly, don't fix profit
		m.OpenQuantity -= tradeQuantity
		m.OpenPriceSum -= tradeQuantity * price
		return 0
	}

	// current position smaller than trade quantity, can close completely and calculate profit
	m.Profit += (m.OpenQuantity*price - m.OpenPriceSum) * int64(m.Direction.Multiplier)
	m.OpenPriceSum = 0
	m.Direction = Empty
	quantityToOpen := tradeQuantity - m.OpenQuantity
	m.OpenQuantity = 0

	m.ValidateInternalState() // TODO comment or not?

	return quantityToOpen
}

func (m *margin) OpenPositionMargin(
	act action.Action,
	quantityToOpen int64,
	price int64,
) {
	m.OpenQuantity += quantityToOpen
	m.OpenPriceSum += quantityToOpen * price
	m.Direction = DirectionFromAction(act)

	m.ValidateInternalState() // TODO comment or not?
}

func (m *margin) Reset() {
	m.PendingBuyQuantity = 0
	m.PendingSellQuantity = 0
	m.OpenQuantity = 0
	m.OpenPriceSum = 0
	m.Direction = Empty
}

func (m *margin) ValidateInternalState() {
	if m.Direction == Empty && (m.OpenQuantity != 0 || m.OpenPriceSum != 0) {
		log.Panicf(
			"Error: userId %v : position:%v totalQuantity:%v openPriceSum:%v",
			m.UserId,
			m.Direction,
			m.OpenQuantity,
			m.OpenPriceSum,
		)
	}

	if m.Direction == Empty && (m.OpenQuantity <= 0 || m.OpenPriceSum <= 0) {
		log.Panicf(
			"Error: userId %v : position:%v totalQuantity:%v openPriceSum:%v",
			m.UserId,
			m.Direction,
			m.OpenQuantity,
			m.OpenPriceSum,
		)
	}

	if m.PendingSellQuantity < 0 || m.PendingBuyQuantity < 0 {
		log.Panicf("Error: userId %v : pendingSellQuantity:%v pendingBuyQuantity:%v",
			m.UserId,
			m.PendingSellQuantity,
			m.PendingBuyQuantity,
		)
	}
}

func (m *margin) Hash() uint64 {
	hash, err := hashstructure.Hash(*m, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (m *margin) Marshal(out *bytes.Buffer) error {
	return MarshalMargin(m, out)
}

// TODO incompatible with exchange-core
func MarshalMargin(in interface{}, out *bytes.Buffer) error {
	s := in.(*margin)

	if err := serialization.MarshalInt64(s.UserId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(s.SymbolId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(s.Currency, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt8(s.Direction.Multiplier, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.OpenQuantity, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.OpenPriceSum, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.Profit, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.PendingSellQuantity, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.PendingBuyQuantity, out); err != nil {
		return err
	}

	return nil
}

// TODO incompatible with exchange-core
func UnmarshalMargin(b *bytes.Buffer) (interface{}, error) {
	m := margin{}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.UserId = val.(int64)
	}

	if val, err := serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	} else {
		m.SymbolId = val.(int32)
	}

	if val, err := serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	} else {
		m.Currency = val.(int32)
	}

	if val, err := serialization.UnmarshalInt8(b); err != nil {
		return nil, err
	} else {
		m.Direction = DirectionFromByte(val.(int8))
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.OpenQuantity = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.OpenPriceSum = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.Profit = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.PendingSellQuantity = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		m.PendingBuyQuantity = val.(int64)
	}

	return &m, nil
}

func UnmarshalMargins(b *bytes.Buffer) (interface{}, error) {
	var val interface{}
	var err error

	if val, err = serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	positions := make(map[int32]Margin, size)

	for size > 0 {
		if k, v, err := serialization.UnmarshalKeyVal(
			b,
			serialization.UnmarshalInt32,
			UnmarshalMargin,
		); err != nil {
			return nil, err
		} else {
			positions[k.(int32)] = v.(Margin)
		}

		size--
	}

	return positions, nil
}

func NewMargin(
	userId int64,
	symbolId int32,
	currency int32,
) Margin {
	return &margin{
		UserId:    userId,
		SymbolId:  symbolId,
		Currency:  currency,
		Direction: Empty,
	}
}
