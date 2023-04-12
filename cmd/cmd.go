package cmd

import (
	"bytes"
	"fmt"

	"github.com/xerexchain/matching-engine/order"
	orderType "github.com/xerexchain/matching-engine/order/t"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/symbol"
	balanceAdjType "github.com/xerexchain/matching-engine/user/balance/adjustment/t"
)

const (
	PlaceOrder_  int8 = 1
	CancelOrder_ int8 = 2
	MoveOrder_   int8 = 3
	ReduceOrder_ int8 = 4

	OrderBookRequest_ int8 = 6

	AddUser_     int8 = 10
	BalanceAdj_  int8 = 11
	SuspendUser_ int8 = 12
	ResumeUser_  int8 = 13
	AddAccounts_ int8 = 14 // TODO vs ADD_ACCOUNTS(1002),

	AddSymbols_ int8 = 40 // TODO vs ADD_SYMBOLS(1003);

	PersistStateMatching_ int8 = 110
	PersistStateRisk_     int8 = 111

	GroupingControl_ int8 = 118
	NOP_             int8 = 120
	Reset_           int8 = 124
	ShutdownSignal_  int8 = 127

	ReservedCompressed_ int8 = -1
)

type Command interface {
	serialization.Marshalable
	serialization.Unmarshalable
	Metadata_() *Metadata
	Code() int8
}

var codeToNew = map[int8]func() Command{
	PlaceOrder_:  newPlaceOrder,
	CancelOrder_: newCancelOrder,
	MoveOrder_:   newMoveOrder,
	ReduceOrder_: newReduceOrder,
	AddUser_:     newAddUser,
	BalanceAdj_:  newBalanceAdj,
	SuspendUser_: newSuspendUser,
	ResumeUser_:  newResumeUser,
	AddAccounts_: newAddAccounts,
	AddSymbols_:  newAddSymbols,
	Reset_:       newReset,
}

// TODO rename?
type Metadata struct {
	Seq          int64
	ServiceFlags int32
	EventsGroup  int64
	TimestampNs  int64
}

type PlaceOrder struct {
	/*
	 * When the order is not fully filled,
	 * OrderId is used to insert new order into orderbook
	 */
	OrderId int64

	UserId        int64
	Price         int64
	Quantity      int64
	ReservedPrice int64
	SymbolId      int32
	UserCookie    int32
	Timestamp     int64 // TODO make sure filled everywhere
	order.Action
	orderType.T
	Metadata
	_ struct{}
}

type CancelOrder struct {
	OrderId  int64
	UserId   int64
	SymbolId int32
	Metadata
	_ struct{}
}

type MoveOrder struct {
	OrderId  int64
	UserId   int64
	SymbolId int32
	ToPrice  int64
	Metadata
	_ struct{}
}

type ReduceOrder struct {
	OrderId   int64
	UserId    int64
	SymbolId  int32
	Quanitity int64
	Metadata
	_ struct{}
}

type AddUser struct {
	UserId int64
	Metadata
	_ struct{}
}

type BalanceAdj struct {
	UserId   int64
	Currency int32
	Amount   int64
	TXID     int64
	balanceAdjType.T
	Metadata
	_ struct{}
}

type SuspendUser struct {
	UserId int64
	Metadata
	_ struct{}
}

type ResumeUser struct {
	UserId int64
	Metadata
	_ struct{}
}

// TODO EqualsAndHashCode overriden
type AddAccounts struct {
	Users map[interface{}]interface{} // map[int64]map[int32]int64
	Metadata
	_ struct{}
}

// TODO EqualsAndHashCode overriden
type AddSymbols struct {
	Symbols map[interface{}]interface{} // map[int32]symbol.Symbol
	Metadata
	_ struct{}
}

type Reset struct {
	Metadata
	_ struct{}
}

func unmarshalUserIdSymbolIdOrderId(in *bytes.Buffer) (int64, int32, int64, error) {

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return 0, 0, 0, err
	}

	symbolId, err := serialization.UnmarshalInt32(in)

	if err != nil {
		return 0, 0, 0, err
	}

	orderId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return 0, 0, 0, err
	}

	return userId.(int64), symbolId.(int32), orderId.(int64), nil
}

func (m *Metadata) Unmarshal(in *bytes.Buffer) error {
	seq, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	timestampNs, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	serviceFlags, err := serialization.UnmarshalInt32(in)

	if err != nil {
		return err
	}

	eventsGroup, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	m.Seq = seq.(int64)
	m.TimestampNs = timestampNs.(int64)
	m.ServiceFlags = serviceFlags.(int32)
	m.EventsGroup = eventsGroup.(int64)

	return nil
}

func (c *PlaceOrder) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, symbolId, orderId, err := unmarshalUserIdSymbolIdOrderId(in)

	if err != nil {
		return err
	}

	price, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	reservedBidPrice, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	quanitity, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	userCookie, err := serialization.UnmarshalInt32(in)

	if err != nil {
		return err
	}

	val, err := serialization.UnmarshalInt8(in)

	if err != nil {
		return err
	}

	actAndType := val.(int8)
	code := actAndType & 0b1
	action, ok := order.ActionFrom(code)

	if !ok {
		return fmt.Errorf("failed to unmarshal action: %v", code)
	}

	code = (actAndType >> 1) & 0b1111
	t, ok := orderType.From(code)

	if !ok {
		return fmt.Errorf("failed to unmarshal order type: %v", code)
	}

	c.OrderId = orderId
	c.UserId = userId
	c.Price = price.(int64)
	c.Quantity = quanitity.(int64)
	c.ReservedPrice = reservedBidPrice.(int64)
	c.SymbolId = symbolId
	c.UserCookie = userCookie.(int32)
	c.Action = action
	c.T = t

	return nil
}

func (c *CancelOrder) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, symbolId, orderId, err := unmarshalUserIdSymbolIdOrderId(in)

	if err != nil {
		return err
	}

	c.OrderId = orderId
	c.UserId = userId
	c.SymbolId = symbolId

	return nil
}

func (c *MoveOrder) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, symbolId, orderId, err := unmarshalUserIdSymbolIdOrderId(in)

	if err != nil {
		return err
	}

	toPrice, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.OrderId = orderId
	c.UserId = userId
	c.SymbolId = symbolId
	c.ToPrice = toPrice.(int64)

	return nil
}

func (c *ReduceOrder) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, symbolId, orderId, err := unmarshalUserIdSymbolIdOrderId(in)

	if err != nil {
		return err
	}

	quanitity, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.OrderId = orderId
	c.UserId = userId
	c.SymbolId = symbolId
	c.Quanitity = quanitity.(int64)

	return nil
}

func (c *AddUser) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.UserId = userId.(int64)

	return nil
}

func (c *BalanceAdj) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	currency, err := serialization.UnmarshalInt32(in)

	if err != nil {
		return err
	}

	txid, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	amount, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	val, err := serialization.UnmarshalInt8(in)

	if err != nil {
		return err
	}

	t, ok := balanceAdjType.From(val.(int8))

	if !ok {
		return fmt.Errorf("failed to unmarshal balance adj type: %v", val)
	}

	c.UserId = userId.(int64)
	c.Currency = currency.(int32)
	c.TXID = txid.(int64)
	c.Amount = amount.(int64)
	c.T = t

	return nil
}

func (c *SuspendUser) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.UserId = userId.(int64)

	return nil
}

func (c *ResumeUser) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.UserId = userId.(int64)

	return nil
}

func (c *AddAccounts) Unmarshal(in *bytes.Buffer) error {
	// err := c.Metadata.Unmarshal(in)

	// if err != nil {
	// 	return err
	// }

	users, err := serialization.UnmarshalMap(
		in,
		serialization.UnmarshalInt64,
		func(b *bytes.Buffer) (interface{}, error) {
			return serialization.UnmarshalMap(
				b,
				serialization.UnmarshalInt32,
				serialization.UnmarshalInt64,
			)
		},
	)

	if err != nil {
		return err
	}

	c.Users = users

	return nil
}

func (c *AddSymbols) Unmarshal(in *bytes.Buffer) error {
	// err := c.Metadata.Unmarshal(in)

	// if err != nil {
	// 	return err
	// }

	symbols, err := serialization.UnmarshalMap(
		in,
		serialization.UnmarshalInt32,
		symbol.UnmarshalSymbol,
	)

	if err != nil {
		return err
	}

	c.Symbols = symbols

	return nil
}

func (c *Reset) Unmarshal(in *bytes.Buffer) error {
	err := c.Metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	return nil
}

func marshalUserIdSymbolIdOrderId(
	userId int64,
	symbolId int32,
	orderId int64,
	out *bytes.Buffer,
) error {
	if err := serialization.MarshalInt64(userId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(symbolId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(orderId, out); err != nil {
		return err
	}

	return nil
}

func (m *Metadata) Marshal(out *bytes.Buffer) error {
	if err := serialization.MarshalInt64(m.Seq, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(m.TimestampNs, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(m.ServiceFlags, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(m.EventsGroup, out); err != nil {
		return err
	}

	return nil
}

func (c *PlaceOrder) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.UserId,
		c.SymbolId,
		c.OrderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.Price, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.ReservedPrice, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.Quantity, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(c.UserCookie, out); err != nil {
		return err
	}

	actionAndType := (int8(c.T) << 1) | int8(c.Action)

	if err := serialization.MarshalInt8(actionAndType, out); err != nil {
		return err
	}

	return nil
}

func (c *CancelOrder) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.UserId,
		c.SymbolId,
		c.OrderId,
		out,
	); err != nil {
		return err
	}

	return nil
}

func (c *MoveOrder) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.UserId,
		c.SymbolId,
		c.OrderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.ToPrice, out); err != nil {
		return err
	}

	return nil
}

func (c *ReduceOrder) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.UserId,
		c.SymbolId,
		c.OrderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.Quanitity, out); err != nil {
		return err
	}

	return nil
}

func (c *AddUser) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.UserId, out); err != nil {
		return err
	}

	return nil
}

func (c *BalanceAdj) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.UserId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(c.Currency, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.TXID, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.Amount, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt8(int8(c.T), out); err != nil {
		return err
	}

	return nil
}

func (c *SuspendUser) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.UserId, out); err != nil {
		return err
	}

	return nil
}

func (c *ResumeUser) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.UserId, out); err != nil {
		return err
	}

	return nil
}

func (c *AddAccounts) Marshal(out *bytes.Buffer) error {
	// if err := c.Metadata.Marshal(out); err != nil {
	// 	return err
	// }

	return serialization.MarshalMap(
		c.Users,
		out,
		serialization.MarshalInt64,
		func(in interface{}, b *bytes.Buffer) error {
			return serialization.MarshalMap(
				in,
				b,
				serialization.MarshalInt32,
				serialization.MarshalInt64,
			)
		},
	)
}

func (c *AddSymbols) Marshal(out *bytes.Buffer) error {
	// if err := c.Metadata.Marshal(out); err != nil {
	// 	return err
	// }

	return serialization.MarshalMap(
		c.Symbols,
		out,
		serialization.MarshalInt32,
		symbol.MarshalSymbol,
	)
}

func (c *Reset) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	return nil
}

func (c *PlaceOrder) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *CancelOrder) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *MoveOrder) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *ReduceOrder) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *AddUser) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *BalanceAdj) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *SuspendUser) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *ResumeUser) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *AddAccounts) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *AddSymbols) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *Reset) Metadata_() *Metadata {
	return &(c.Metadata)
}

func (c *PlaceOrder) Code() int8 {
	return PlaceOrder_
}

func (c *CancelOrder) Code() int8 {
	return CancelOrder_
}

func (c *MoveOrder) Code() int8 {
	return MoveOrder_
}

func (c *ReduceOrder) Code() int8 {
	return ReduceOrder_
}

func (c *AddUser) Code() int8 {
	return AddUser_
}

func (c *BalanceAdj) Code() int8 {
	return BalanceAdj_
}

func (c *SuspendUser) Code() int8 {
	return SuspendUser_
}

func (c *ResumeUser) Code() int8 {
	return ResumeUser_
}

func (c *AddAccounts) Code() int8 {
	return AddAccounts_
}

func (c *AddSymbols) Code() int8 {
	return AddSymbols_
}

func (c *Reset) Code() int8 {
	return Reset_
}

func newPlaceOrder() Command {
	return &PlaceOrder{}
}

func newCancelOrder() Command {
	return &CancelOrder{}
}

func newMoveOrder() Command {
	return &MoveOrder{}
}

func newReduceOrder() Command {
	return &ReduceOrder{}
}

func newAddUser() Command {
	return &AddUser{}
}

func newBalanceAdj() Command {
	return &BalanceAdj{}
}

func newSuspendUser() Command {
	return &SuspendUser{}
}

func newResumeUser() Command {
	return &ResumeUser{}
}

func newAddAccounts() Command {
	return &AddAccounts{}
}

func newAddSymbols() Command {
	return &AddSymbols{}
}

func newReset() Command {
	return &Reset{}
}

func From(code int8) (Command, bool) {
	if f, ok := codeToNew[code]; ok {
		return f(), true
	}

	return nil, false
}
