package cmd

import (
	"bytes"
	"fmt"

	"github.com/xerexchain/matching-engine/order/action"
	orderType "github.com/xerexchain/matching-engine/order/t"
	"github.com/xerexchain/matching-engine/serialization"
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

	BinaryDataQuery_   int8 = 90
	BinaryDataCommand_ int8 = 91

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
	Metadata() *Metadata
	Code() int8
}

var codeToNew = map[int8]func() Command{
	PlaceOrder_:        newPlaceOrder,
	CancelOrder_:       newCancelOrder,
	MoveOrder_:         newMoveOrder,
	ReduceOrder_:       newReduceOrder,
	AddUser_:           newAddUser,
	BalanceAdj_:        newBalanceAdj,
	SuspendUser_:       newSuspendUser,
	ResumeUser_:        newResumeUser,
	BinaryDataCommand_: newBinaryDataCommand,
	Reset_:             newReset,
}

// TODO rename?
type Metadata struct {
	Seq          int64
	ServiceFlags int32
	EventsGroup  int64
	TimestampNs  int64
}

type PlaceOrder struct {
	orderId       int64 // TODO Is it redundant?
	userId        int64
	price         int64
	quantity      int64
	reservedPrice int64
	symbolId      int32
	userCookie    int32
	action        action.Action
	t             orderType.T
	metadata      *Metadata
	_             struct{}
}

type CancelOrder struct {
	orderId  int64
	userId   int64
	symbolId int32
	metadata *Metadata
	_        struct{}
}

type MoveOrder struct {
	orderId  int64
	userId   int64
	symbolId int32
	toPrice  int64
	metadata *Metadata
	_        struct{}
}

type ReduceOrder struct {
	orderId   int64
	userId    int64
	symbolId  int32
	quanitity int64
	metadata  *Metadata
	_         struct{}
}

type AddUser struct {
	userId   int64
	metadata *Metadata
	_        struct{}
}

type BalanceAdj struct {
	userId   int64
	currency int32
	amount   int64
	txid     int64
	t        balanceAdjType.T
	metadata *Metadata
	_        struct{}
}

type SuspendUser struct {
	userId   int64
	metadata *Metadata
	_        struct{}
}

type ResumeUser struct {
	userId   int64
	metadata *Metadata
	_        struct{}
}

type BinaryDataCommand struct {
	lastFlag int8
	word0    int64
	word1    int64
	word2    int64
	word3    int64
	word4    int64
	metadata *Metadata
	_        struct{}
}

type Reset struct {
	metadata *Metadata
	_        struct{}
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
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

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
	act, ok := action.From(code)

	if !ok {
		return fmt.Errorf("failed to unmarshal action: %v", code)
	}

	code = (actAndType >> 1) & 0b1111
	t, ok := orderType.From(code)

	if !ok {
		return fmt.Errorf("failed to unmarshal order type: %v", code)
	}

	c.orderId = orderId
	c.userId = userId
	c.price = price.(int64)
	c.quantity = quanitity.(int64)
	c.reservedPrice = reservedBidPrice.(int64)
	c.symbolId = symbolId
	c.userCookie = userCookie.(int32)
	c.action = act
	c.t = t
	c.metadata = metadata

	return nil
}

func (c *CancelOrder) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, symbolId, orderId, err := unmarshalUserIdSymbolIdOrderId(in)

	if err != nil {
		return err
	}

	c.orderId = orderId
	c.userId = userId
	c.symbolId = symbolId
	c.metadata = metadata

	return nil
}

func (c *MoveOrder) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

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

	c.orderId = orderId
	c.userId = userId
	c.symbolId = symbolId
	c.toPrice = toPrice.(int64)
	c.metadata = metadata

	return nil
}

func (c *ReduceOrder) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

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

	c.orderId = orderId
	c.userId = userId
	c.symbolId = symbolId
	c.quanitity = quanitity.(int64)
	c.metadata = metadata

	return nil
}

func (c *AddUser) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.metadata = metadata

	return nil
}

func (c *BalanceAdj) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

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

	c.userId = userId.(int64)
	c.currency = currency.(int32)
	c.txid = txid.(int64)
	c.amount = amount.(int64)
	c.t = t
	c.metadata = metadata

	return nil
}

func (c *SuspendUser) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.metadata = metadata

	return nil
}

func (c *ResumeUser) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.metadata = metadata

	return nil
}

func (c *BinaryDataCommand) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	lastFlag, err := serialization.UnmarshalInt8(in)

	if err != nil {
		return err
	}

	word0, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	word1, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	word2, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	word3, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	word4, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.lastFlag = lastFlag.(int8)
	c.word0 = word0.(int64)
	c.word1 = word1.(int64)
	c.word2 = word2.(int64)
	c.word3 = word3.(int64)
	c.word4 = word4.(int64)
	c.metadata = metadata

	return nil
}

func (c *Reset) Unmarshal(in *bytes.Buffer) error {
	metadata := &Metadata{}

	err := metadata.Unmarshal(in)

	if err != nil {
		return err
	}

	c.metadata = metadata

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
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.userId,
		c.symbolId,
		c.orderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.price, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.reservedPrice, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.quantity, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(c.userCookie, out); err != nil {
		return err
	}

	actionAndType := (int8(c.t) << 1) | int8(c.action)

	if err := serialization.MarshalInt8(actionAndType, out); err != nil {
		return err
	}

	return nil
}

func (c *CancelOrder) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.userId,
		c.symbolId,
		c.orderId,
		out,
	); err != nil {
		return err
	}

	return nil
}

func (c *MoveOrder) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.userId,
		c.symbolId,
		c.orderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.toPrice, out); err != nil {
		return err
	}

	return nil
}

func (c *ReduceOrder) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalUserIdSymbolIdOrderId(
		c.userId,
		c.symbolId,
		c.orderId,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.quanitity, out); err != nil {
		return err
	}

	return nil
}

func (c *AddUser) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *BalanceAdj) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(c.currency, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.txid, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.amount, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt8(int8(c.t), out); err != nil {
		return err
	}

	return nil
}

func (c *SuspendUser) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *ResumeUser) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *BinaryDataCommand) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := serialization.MarshalInt8(c.lastFlag, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.word0, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.word1, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.word2, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.word3, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.word4, out); err != nil {
		return err
	}

	return nil
}

func (c *Reset) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	return nil
}

func (c *PlaceOrder) Metadata() *Metadata {
	return c.metadata
}

func (c *CancelOrder) Metadata() *Metadata {
	return c.metadata
}

func (c *MoveOrder) Metadata() *Metadata {
	return c.metadata
}

func (c *ReduceOrder) Metadata() *Metadata {
	return c.metadata
}

func (c *AddUser) Metadata() *Metadata {
	return c.metadata
}

func (c *BalanceAdj) Metadata() *Metadata {
	return c.metadata
}

func (c *SuspendUser) Metadata() *Metadata {
	return c.metadata
}

func (c *ResumeUser) Metadata() *Metadata {
	return c.metadata
}

func (c *BinaryDataCommand) Metadata() *Metadata {
	return c.metadata
}

func (c *Reset) Metadata() *Metadata {
	return c.metadata
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

func (c *BinaryDataCommand) Code() int8 {
	return BinaryDataCommand_
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

func newBinaryDataCommand() Command {
	return &BinaryDataCommand{}
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
