package cmd

import (
	"bytes"
	"fmt"

	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/order/action"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/user"
)

const (
	placeOrder  int8 = 1
	cancelOrder int8 = 2
	moveOrder   int8 = 3
	reduceOrder int8 = 4

	orderBookRequest int8 = 6

	addUser     int8 = 10
	balanceAdj  int8 = 11
	suspendUser int8 = 12
	resumeUser  int8 = 13

	binaryDataQuery   int8 = 90
	binaryDataCommand int8 = 91

	persistStateMatching int8 = 110
	PersistStateRisk_    int8 = 111

	groupingControl int8 = 118
	nop             int8 = 120
	Reset_           int8 = 124
	ShutdownSignal  int8 = 127

	ReservedCompressed int8 = -1
)

type Command interface {
	serialization.Marshalable
	serialization.Unmarshalable
	Seq() int64
	SetSeq(int64)
	TimestampNs() int64
	Code() int8 // TODO remove?
}

var codeToNew = map[int8]func() Command{
	placeOrder:        newPlaceOrder,
	cancelOrder:       newCancelOrder,
	moveOrder:         newMoveOrder,
	reduceOrder:       newReduceOrder,
	addUser:           newAddUser,
	balanceAdj:        newBalanceAdj,
	suspendUser:       newSuspendUser,
	resumeUser:        newResumeUser,
	binaryDataCommand: newBinaryDataCommand,
	PersistStateRisk_: newPersistStateRisk,
	Reset_:             newReset,
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
	t             order.Type
	seq           int64
	serviceFlag   int32
	eventsGroup   int64
	timestampNs   int64
	_             struct{}
}

type CancelOrder struct {
	orderId     int64
	userId      int64
	symbolId    int32
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type MoveOrder struct {
	orderId     int64
	userId      int64
	symbolId    int32
	toPrice     int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type ReduceOrder struct {
	orderId     int64
	userId      int64
	symbolId    int32
	quanitity   int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type AddUser struct {
	userId      int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type BalanceAdj struct {
	userId      int64
	currency    int32
	amount      int64
	txid        int64
	t           user.BalanceAdjType
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type SuspendUser struct {
	userId      int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type ResumeUser struct {
	userId      int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type BinaryDataCommand struct {
	lastFlag    int8
	word0       int64
	word1       int64
	word2       int64
	word3       int64
	word4       int64
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type PersistStateRisk struct {
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

type Reset struct {
	seq         int64
	serviceFlag int32
	eventsGroup int64
	timestampNs int64
	_           struct{}
}

func unmarshalSeqSerFlagEveGroupTimestamp(in *bytes.Buffer) (int64, int32, int64, int64, error) {
	var seq int64

	if val, err := serialization.UnmarshalInt64(in); err != nil {
		return 0, 0, 0, 0, err
	} else {
		seq = val.(int64)
	}

	var timestampNs int64

	if val, err := serialization.UnmarshalInt64(in); err != nil {
		return 0, 0, 0, 0, err
	} else {
		timestampNs = val.(int64)
	}

	var serviceFlags int32

	if val, err := serialization.UnmarshalInt32(in); err != nil {
		return 0, 0, 0, 0, err
	} else {
		serviceFlags = val.(int32)
	}

	var eventsGroup int64

	if val, err := serialization.UnmarshalInt64(in); err != nil {
		return 0, 0, 0, 0, err
	} else {
		eventsGroup = val.(int64)
	}

	return seq, serviceFlags, eventsGroup, timestampNs, nil
}

func unmarshalUserIdSymbolIdOrderId(in *bytes.Buffer) (int64, int32, int64, error) {
	var userId int64

	if val, err := serialization.UnmarshalInt64(in); err != nil {
		return 0, 0, 0, err
	} else {
		userId = val.(int64)
	}

	var symbolId int32

	if val, err := serialization.UnmarshalInt32(in); err != nil {
		return 0, 0, 0, err
	} else {
		symbolId = val.(int32)
	}

	var orderId int64

	if val, err := serialization.UnmarshalInt64(in); err != nil {
		return 0, 0, 0, err
	} else {
		orderId = val.(int64)
	}

	return userId, symbolId, orderId, nil
}

func (c *PlaceOrder) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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
	act, ok := action.FromCode(code)

	if !ok {
		return fmt.Errorf("failed to unmarshal action: %v", code)
	}

	code = (actAndType >> 1) & 0b1111
	t, ok := order.FromCode(code)

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
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *CancelOrder) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *MoveOrder) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *ReduceOrder) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *AddUser) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *BalanceAdj) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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

	t, ok := user.FromCode(val.(int8))

	if !ok {
		return fmt.Errorf("failed to unmarshal balance adj type: %v", val)
	}

	c.userId = userId.(int64)
	c.currency = currency.(int32)
	c.txid = txid.(int64)
	c.amount = amount.(int64)
	c.t = t
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *SuspendUser) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *ResumeUser) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

	if err != nil {
		return err
	}

	userId, err := serialization.UnmarshalInt64(in)

	if err != nil {
		return err
	}

	c.userId = userId.(int64)
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *BinaryDataCommand) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

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
	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *PersistStateRisk) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

	if err != nil {
		return err
	}

	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

	return nil
}

func (c *Reset) Unmarshal(in *bytes.Buffer) error {
	seq, serviceFlags, eventsGroup, timestampNs, err := unmarshalSeqSerFlagEveGroupTimestamp(in)

	if err != nil {
		return err
	}

	c.seq = seq
	c.serviceFlag = serviceFlags
	c.eventsGroup = eventsGroup
	c.timestampNs = timestampNs

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

func marshalSeqSerFlagEveGroupTimestamp(
	seq int64,
	serviceFlags int32,
	eventsGroup int64,
	timestampNs int64,
	out *bytes.Buffer,
) error {
	if err := serialization.MarshalInt64(seq, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(timestampNs, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(serviceFlags, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(eventsGroup, out); err != nil {
		return err
	}

	return nil
}

func (c *PlaceOrder) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *BalanceAdj) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *ResumeUser) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(c.userId, out); err != nil {
		return err
	}

	return nil
}

func (c *BinaryDataCommand) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
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

func (c *PersistStateRisk) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
		return err
	}

	return nil
}

func (c *Reset) Marshal(out *bytes.Buffer) error {
	if err := marshalSeqSerFlagEveGroupTimestamp(
		c.seq,
		c.serviceFlag,
		c.eventsGroup,
		c.timestampNs,
		out,
	); err != nil {
		return err
	}

	return nil
}

func (c *PlaceOrder) Seq() int64 {
	return c.seq
}

func (c *CancelOrder) Seq() int64 {
	return c.seq
}

func (c *MoveOrder) Seq() int64 {
	return c.seq
}

func (c *ReduceOrder) Seq() int64 {
	return c.seq
}

func (c *AddUser) Seq() int64 {
	return c.seq
}

func (c *BalanceAdj) Seq() int64 {
	return c.seq
}

func (c *SuspendUser) Seq() int64 {
	return c.seq
}

func (c *ResumeUser) Seq() int64 {
	return c.seq
}

func (c *BinaryDataCommand) Seq() int64 {
	return c.seq
}

func (c *PersistStateRisk) Seq() int64 {
	return c.seq
}

func (c *Reset) Seq() int64 {
	return c.seq
}

func (c *PlaceOrder) SetSeq(seq int64) {
	c.seq = seq
}

func (c *CancelOrder) SetSeq(seq int64) {
	c.seq = seq
}

func (c *MoveOrder) SetSeq(seq int64) {
	c.seq = seq
}

func (c *ReduceOrder) SetSeq(seq int64) {
	c.seq = seq
}

func (c *AddUser) SetSeq(seq int64) {
	c.seq = seq
}

func (c *BalanceAdj) SetSeq(seq int64) {
	c.seq = seq
}

func (c *SuspendUser) SetSeq(seq int64) {
	c.seq = seq
}

func (c *ResumeUser) SetSeq(seq int64) {
	c.seq = seq
}

func (c *BinaryDataCommand) SetSeq(seq int64) {
	c.seq = seq
}

func (c *PersistStateRisk) SetSeq(seq int64) {
	c.seq = seq
}

func (c *Reset) SetSeq(seq int64) {
	c.seq = seq
}

func (c *PlaceOrder) TimestampNs() int64 {
	return c.timestampNs
}

func (c *CancelOrder) TimestampNs() int64 {
	return c.timestampNs
}

func (c *MoveOrder) TimestampNs() int64 {
	return c.timestampNs
}

func (c *ReduceOrder) TimestampNs() int64 {
	return c.timestampNs
}

func (c *AddUser) TimestampNs() int64 {
	return c.timestampNs
}

func (c *BalanceAdj) TimestampNs() int64 {
	return c.timestampNs
}

func (c *SuspendUser) TimestampNs() int64 {
	return c.timestampNs
}

func (c *ResumeUser) TimestampNs() int64 {
	return c.timestampNs
}

func (c *BinaryDataCommand) TimestampNs() int64 {
	return c.timestampNs
}

func (c *PersistStateRisk) TimestampNs() int64 {
	return c.timestampNs
}

func (c *Reset) TimestampNs() int64 {
	return c.timestampNs
}

func (c *PlaceOrder) Code() int8 {
	return placeOrder
}

func (c *CancelOrder) Code() int8 {
	return cancelOrder
}

func (c *MoveOrder) Code() int8 {
	return moveOrder
}

func (c *ReduceOrder) Code() int8 {
	return reduceOrder
}

func (c *AddUser) Code() int8 {
	return addUser
}

func (c *BalanceAdj) Code() int8 {
	return balanceAdj
}

func (c *SuspendUser) Code() int8 {
	return suspendUser
}

func (c *ResumeUser) Code() int8 {
	return resumeUser
}

func (c *BinaryDataCommand) Code() int8 {
	return binaryDataCommand
}

func (c *PersistStateRisk) Code() int8 {
	return Reset_
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

func newPersistStateRisk() Command {
	return &PersistStateRisk{}
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
