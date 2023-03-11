package position

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/order"
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

type MarginPosition interface {
	state.Hashable
	serialization.Marshalable
	SetUserId(int64)
	IsEmpty() bool
	PendingHold(action order.Action, quantity int64)
	PendingRelease(action order.Action, quantity int64)
	EstimateProfit(
		symbol.FutureContractSymbol,
		riskengine.LastPriceCacheRecord,
	) int64
	CalculateRequiredMarginForFutures(
		symbol.FutureContractSymbol,
	) int64
	CalculateRequiredMarginForOrder(
		sym symbol.FutureContractSymbol,
		action order.Action,
		quantity int64,
	) int64
	UpdatePositionForMarginTrade(
		action order.Action,
		quantity int64,
		price int64,
	) int64
	Reset()
	ValidateInternalState()
}

type marginPosition struct {
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

func (p Direction) IsOppositeToAction(action order.Action) bool {
	return (p == Long && action == order.Ask) || (p == Short && action == order.Bid)
}

func (p Direction) IsSameAsAction(action order.Action) bool {
	return (p == Long && action == order.Bid) || (p == Short && action == order.Ask)
}

func DirectionFromAction(act order.Action) Direction {
	if act == order.Bid {
		return Long
	} else {
		return Short
	}
}

func (mp *marginPosition) SetUserId(id int64) {
	mp.UserId = id
}

// Check if position is empty (no pending orders, no open trades) - can remove it from hashmap
func (mp *marginPosition) IsEmpty() bool {
	return mp.Direction == Empty && mp.PendingSellQuantity == 0 && mp.PendingBuyQuantity == 0
}

func (mp *marginPosition) PendingHold(action order.Action, quantity int64) {
	if action == order.Ask {
		mp.PendingSellQuantity += quantity
	} else {
		mp.PendingBuyQuantity += quantity
	}

	// TODO handle overflow
}

func (mp *marginPosition) PendingRelease(action order.Action, quantity int64) {
	if action == order.Ask {
		mp.PendingSellQuantity -= quantity
	} else {
		mp.PendingBuyQuantity -= quantity
	}

	// TODO check for negative values
}

// TODO relation of sym and symbolId
func (mp *marginPosition) EstimateProfit(
	sym symbol.FutureContractSymbol,
	rec riskengine.LastPriceCacheRecord,
) int64 {
	switch mp.Direction {
	case Empty:
		return mp.Profit
	case Long:
		{
			p := mp.Profit

			if rec != nil && rec.BidPrice() != 0 {
				p += mp.OpenQuantity*rec.BidPrice() - mp.OpenPriceSum
			} else {
				// unknown price - no liquidity - require extra margin
				p += sym.MarginBuy() * mp.OpenQuantity
			}

			return p
		}
	case Short:
		{
			p := mp.Profit

			if rec != nil && rec.AskPrice() != math.MaxInt64 {
				p += mp.OpenPriceSum - mp.OpenQuantity*rec.AskPrice()
			} else {
				// unknown price - no liquidity - require extra margin
				p += sym.MarginSell() * mp.OpenQuantity
			}

			return p
		}
	default:
		panic("Unknown position")
	}
}

func (mp *marginPosition) M(
	sym symbol.FutureContractSymbol,
) (int64, int64) {
	signedPosition := mp.OpenQuantity * int64(mp.Direction.Multiplier)
	currRiskBuyQuantity := mp.PendingBuyQuantity + signedPosition
	currRiskSellQuantity := mp.PendingSellQuantity - signedPosition

	marginBuy := currRiskBuyQuantity * sym.MarginBuy()
	MarginSell := currRiskSellQuantity * sym.MarginSell()

	return marginBuy, MarginSell
}

func (mp *marginPosition) CalculateRequiredMarginForFutures(
	sym symbol.FutureContractSymbol,
) int64 {
	marginBuy, MarginSell := mp.M(sym)

	if marginBuy > MarginSell {
		return marginBuy
	} else {
		return MarginSell
	}
}

// considering extra quantity added to current position (or outstanding orders)
// return -1 if order will reduce current exposure (no additional margin required),
// otherwise full margin for symbol position if order placed/executed
func (mp *marginPosition) CalculateRequiredMarginForOrder(
	sym symbol.FutureContractSymbol,
	action order.Action,
	quantity int64,
) int64 {
	marginBuy, MarginSell := mp.M(sym)
	var currMargin int64

	if marginBuy > MarginSell {
		currMargin = marginBuy
	} else {
		currMargin = MarginSell
	}

	if action == order.Bid {
		marginBuy += sym.MarginBuy() * quantity
	} else {
		MarginSell += sym.MarginSell() * quantity
	}

	var newMargin int64

	if marginBuy > MarginSell {
		newMargin = marginBuy
	} else {
		newMargin = MarginSell
	}

	if newMargin <= currMargin {
		return -1
	} else {
		return newMargin
	}
}

// Update position for one user
// return opened quantity
func (mp *marginPosition) UpdatePositionForMarginTrade(
	action order.Action,
	quantity int64,
	price int64,
) int64 {
	// 1. Un-hold pending quantity
	mp.PendingRelease(action, quantity)

	// 2. Reduce opposite position accordingly (if exists)
	quantityToOpen := mp.CloseCurrPositionFutures(action, quantity, price)

	// 3. Increase forward position accordingly (if quantity left in the trading event)
	if quantityToOpen > 0 {
		mp.OpenPositionMargin(action, quantityToOpen, price)
	}

	return quantityToOpen
}

func (mp *marginPosition) CloseCurrPositionFutures(
	action order.Action,
	tradeQuantity int64,
	tradePrice int64,
) int64 {
	if mp.Direction == Empty || mp.Direction == DirectionFromAction(action) {
		// nothing to close
		return tradeQuantity
	}

	if mp.OpenQuantity > tradeQuantity {
		// current position is bigger than trade quantity - just reduce position accordingly, don't fix profit
		mp.OpenQuantity -= tradeQuantity
		mp.OpenPriceSum -= tradeQuantity * tradePrice
		return 0
	}

	// current position smaller than trade quantity, can close completely and calculate profit
	mp.Profit += (mp.OpenQuantity*tradePrice - mp.OpenPriceSum) * int64(mp.Direction.Multiplier)
	mp.OpenPriceSum = 0
	mp.Direction = Empty
	quantityToOpen := tradeQuantity - mp.OpenQuantity
	mp.OpenQuantity = 0

	mp.ValidateInternalState() // TODO comment or not?

	return quantityToOpen
}

func (mp *marginPosition) OpenPositionMargin(
	action order.Action,
	quantityToOpen int64,
	tradePrice int64,
) {
	mp.OpenQuantity += quantityToOpen
	mp.OpenPriceSum += quantityToOpen * tradePrice
	mp.Direction = DirectionFromAction(action)

	mp.ValidateInternalState() // TODO comment or not?
}

func (mp *marginPosition) Reset() {
	mp.PendingBuyQuantity = 0
	mp.PendingSellQuantity = 0
	mp.OpenQuantity = 0
	mp.OpenPriceSum = 0
	mp.Direction = Empty
}

func (mp *marginPosition) ValidateInternalState() {
	if mp.Direction == Empty && (mp.OpenQuantity != 0 || mp.OpenPriceSum != 0) {
		log.Panicf(
			"Error: userId %v : position:%v totalQuantity:%v openPriceSum:%v",
			mp.UserId,
			mp.Direction,
			mp.OpenQuantity,
			mp.OpenPriceSum,
		)
		panic("invalid margin position state 1")
	}

	if mp.Direction == Empty && (mp.OpenQuantity <= 0 || mp.OpenPriceSum <= 0) {
		log.Panicf(
			"Error: userId %v : position:%v totalQuantity:%v openPriceSum:%v",
			mp.UserId,
			mp.Direction,
			mp.OpenQuantity,
			mp.OpenPriceSum,
		)

		panic("invalid margin position state 2")
	}

	if mp.PendingSellQuantity < 0 || mp.PendingBuyQuantity < 0 {
		log.Panicf("Error: userId %v : pendingSellSize:%v pendingBuySize:%v",
			mp.UserId,
			mp.PendingSellQuantity,
			mp.PendingBuyQuantity,
		)

		panic("invalid margin position state 3")
	}
}

func (mp *marginPosition) Hash() uint64 {
	hash, err := hashstructure.Hash(*mp, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (mp *marginPosition) Marshal(out *bytes.Buffer) error {
	return MarshalMarginPosition(mp, out)
}

// TODO incompatible with exchange-core
func MarshalMarginPosition(in interface{}, out *bytes.Buffer) error {
	s := in.(*marginPosition)

	if err := binary.Write(out, binary.LittleEndian, s.UserId); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.SymbolId); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.Currency); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.Direction.Multiplier); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.OpenQuantity); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.OpenPriceSum); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.Profit); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.PendingSellQuantity); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.PendingBuyQuantity); err != nil {
		return err
	}

	return s.Marshal(out)
}

// TODO incompatible with exchange-core
func UnmarshalMarginPosition(in *bytes.Buffer) (interface{}, error) {
	m := marginPosition{}

	if err := binary.Read(in, binary.LittleEndian, &(m.UserId)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.SymbolId)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.Currency)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.Direction.Multiplier)); err != nil {
		return nil, err
	}

	if m.Direction != Long && m.Direction != Short && m.Direction != Empty {
		return nil, fmt.Errorf("invalid position direction: %v", m.Direction.Multiplier)
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.OpenQuantity)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.OpenPriceSum)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.Profit)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.PendingSellQuantity)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(m.PendingBuyQuantity)); err != nil {
		return nil, err
	}

	return &m, nil
}

func NewMarginPosition(
	userId int64,
	symbolId int32,
	currency int32,
) MarginPosition {
	return &marginPosition{
		UserId:    userId,
		SymbolId:  symbolId,
		Currency:  currency,
		Direction: Empty,
	}
}
