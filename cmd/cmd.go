package cmd

import (
	"bytes"
	"fmt"

	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/symbol"
	"github.com/xerexchain/matching-engine/user"
)

var (
	Place_            int8 = (&order.Place{}).Code()
	Cancel_           int8 = (&order.Cancel{}).Code()
	Move_             int8 = (&order.Move{}).Code()
	Reduce_           int8 = (&order.Reduce{}).Code()
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
	Seq() int64
	SetSeq(int64)
	TimestampNS() int64
	Code() int8
}

// add order commands
var _codeToNew = map[int8]func() Command{
	Place_:       newPlace,
	Cancel_:      newCancel,
	Move_:        newMove,
	Reduce_:      newReduce,
	AddUser_:     newAddUser,
	BalanceAdj_:  newBalanceAdj,
	SuspendUser_: newSuspendUser,
	ResumeUser_:  newResumeUser,
	AddAccounts_: newAddAccounts,
	AddSymbols_:  newAddSymbols,
	Reset_:       newReset,
}

type Symbol interface {
	serialization.Marshalable
	serialization.Unmarshalable
	ID() int32
}

// TODO rename?
// rename everywhere used
type Metadata struct {
	Seq          int64
	ServiceFlags int32
	EventsGroup  int64
	TimestampNs  int64
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
	user.BalanceAdjCategory
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
	Symbols map[int32]Symbol
	Metadata
	_ struct{}
}

type Reset struct {
	Metadata
	_ struct{}
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

	cat, ok := user.BalanceAdjCategoryFrom(val.(int8))

	if !ok {
		return fmt.Errorf("failed to unmarshal balance adj type: %v", val)
	}

	c.UserId = userId.(int64)
	c.Currency = currency.(int32)
	c.TXID = txid.(int64)
	c.Amount = amount.(int64)
	c.BalanceAdjCategory = cat

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

	size, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	symbols := make(map[int32]Symbol, size)

	for ; size > 0; size-- {
		symbolID, err := serialization.ReadInt32(in)

		if err != nil {
			return err
		}

		symbol_, err := symbol.Unmarshal(in)

		if err != nil {
			return err
		}

		symbols[symbolID] = symbol_
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

func (m *Metadata) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(m.Seq, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.TimestampNs, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(m.ServiceFlags, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.EventsGroup, out); err != nil {
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

	if err := serialization.MarshalInt8(int8(c.BalanceAdjCategory), out); err != nil {
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

	size := int32(len(c.Symbols))

	if err := serialization.WriteInt32(size, out); err != nil {
		return err
	}

	for symbolID, symbol_ := range c.Symbols {
		if err := serialization.WriteInt32(symbolID, out); err != nil {
			return err
		}

		if err := symbol_.Marshal(out); err != nil {
			return err
		}
	}

	return nil
}

func (c *Reset) Marshal(out *bytes.Buffer) error {
	if err := c.Metadata.Marshal(out); err != nil {
		return err
	}

	return nil
}

func (c *AddUser) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *BalanceAdj) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *SuspendUser) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *ResumeUser) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *AddAccounts) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *AddSymbols) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *Reset) TimestampNS() int64 {
	return c.Metadata.TimestampNs
}

func (c *AddUser) Seq() int64 {
	return c.Metadata.Seq
}

func (c *BalanceAdj) Seq() int64 {
	return c.Metadata.Seq
}

func (c *SuspendUser) Seq() int64 {
	return c.Metadata.Seq
}

func (c *ResumeUser) Seq() int64 {
	return c.Metadata.Seq
}

func (c *AddAccounts) Seq() int64 {
	return c.Metadata.Seq
}

func (c *AddSymbols) Seq() int64 {
	return c.Metadata.Seq
}

func (c *Reset) Seq() int64 {
	return c.Metadata.Seq
}

func (c *AddUser) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *BalanceAdj) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *SuspendUser) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *ResumeUser) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *AddAccounts) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *AddSymbols) SetSeq(seq int64) {
	c.Metadata.Seq = seq
}

func (c *Reset) SetSeq(seq int64) {
	c.Metadata.Seq = seq
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

func newPlace() Command {
	return &order.Place{}
}

func newCancel() Command {
	return &order.Cancel{}
}

func newMove() Command {
	return &order.Move{}
}

func newReduce() Command {
	return &order.Reduce{}
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
	if f, ok := _codeToNew[code]; ok {
		return f(), true
	}

	return nil, false
}
