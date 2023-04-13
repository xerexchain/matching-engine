package order

import (
	"bytes"
	"fmt"

	"github.com/xerexchain/matching-engine/serialization"
)

// TODO rename Place, Cancel, Move, Reduce
// prepend or append Command?

type _metadata struct {
	seq          int64
	serviceFlags int32
	eventsGroup  int64
	timestampNs  int64
	_            struct{}
}

func (m *_metadata) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(m.seq, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.timestampNs, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(m.serviceFlags, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.eventsGroup, out); err != nil {
		return err
	}

	return nil
}

func (m *_metadata) Unmarshal(in *bytes.Buffer) error {
	seq, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	timestampNs, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	serviceFlags, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	eventsGroup, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	m.seq = seq
	m.timestampNs = timestampNs
	m.serviceFlags = serviceFlags
	m.eventsGroup = eventsGroup

	return nil
}

/*
 * `Place` and `Order` present the same data.
 * `Order` is used inside `Orderbook` while
 * `Place` is the payload received from user.
 */
type Place struct {
	/*
	 * When the `Order` is not fully filled,
	 * `orderID` is used to insert new `Order` into `Orderbook`.
	 */
	orderID int64

	userID        int64
	price         int64
	quantity      int64
	reservedPrice int64
	symbolID      int32
	userCookie    int32 // TODO expose this field? security
	timestamp     int64 // TODO make sure filled everywhere
	action        Action
	category      _category
	metadata      _metadata
	_             struct{}
}

func NewPlace(
	orderID int64,
	userID int64,
	price int64,
	quantity int64,
	reservedPrice int64,
	symbolID int32,
	timestamp int64,
	action Action,
	category _category,
) *Place {
	return &Place{
		orderID:       orderID,
		userID:        userID,
		price:         price,
		quantity:      quantity,
		reservedPrice: reservedPrice,
		symbolID:      symbolID,
		timestamp:     timestamp,
		action:        action,
		category:      category,
	}
}

func (p *Place) Code() int8 {
	return 1
}

func (p *Place) OrderID() int64 {
	return p.orderID
}

func (p *Place) UserID() int64 {
	return p.userID
}

func (p *Place) Price() int64 {
	return p.price
}

func (p *Place) Quantity() int64 {
	return p.quantity
}

func (p *Place) Reduce(quantity int64) {
	p.quantity -= quantity
}

func (p *Place) ReservedPrice() int64 {
	return p.reservedPrice
}

func (p *Place) SymbolID() int32 {
	return p.symbolID
}

func (p *Place) UserCookie() int32 {
	return p.userCookie
}

func (p *Place) Timestamp() int64 {
	return p.timestamp
}

func (p *Place) Action() Action {
	return p.action
}

func (p *Place) Category() _category {
	return p.category
}

func (p *Place) Marshal(out *bytes.Buffer) error {
	if err := p.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalCommon(
		p.userID,
		p.symbolID,
		p.orderID,
		out,
	); err != nil {
		return err
	}

	if err := serialization.WriteInt64(p.price, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(p.reservedPrice, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(p.quantity, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(p.userCookie, out); err != nil {
		return err
	}

	actionAndCategory := (int8(p.category) << 1) | int8(p.action)

	if err := serialization.WriteInt8(actionAndCategory, out); err != nil {
		return err
	}

	return nil
}

func (p *Place) Unmarshal(in *bytes.Buffer) error {
	if err := p.metadata.Unmarshal(in); err != nil {
		return err
	}

	userID, symbolID, orderID, err := unmarshalCommon(in)

	if err != nil {
		return err
	}

	price, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	reservedPrice, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	quanitity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	userCookie, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	actionAndCategory, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	code := actionAndCategory & 0b1
	action, ok := ActionFrom(code)

	if !ok {
		return fmt.Errorf("unmarshal: action: %v", code)
	}

	code = (actionAndCategory >> 1) & 0b1111
	category, ok := categoryFrom(code)

	if !ok {
		return fmt.Errorf("unmarshal: category : %v", code)
	}

	p.orderID = orderID
	p.userID = userID
	p.price = price
	p.quantity = quanitity
	p.reservedPrice = reservedPrice
	p.symbolID = symbolID
	p.userCookie = userCookie
	p.action = action
	p.category = category

	return nil
}

type Cancel struct {
	orderID  int64
	userID   int64
	symbolID int32
	metadata _metadata
	_        struct{}
}

func NewCancel() *Cancel {
	return &Cancel{}
}

func (c *Cancel) Code() int8 {
	return 2
}

func (c *Cancel) OrderID() int64 {
	return c.orderID
}

func (c *Cancel) UserID() int64 {
	return c.userID
}

func (c *Cancel) SymbolID() int32 {
	return c.symbolID
}

func (c *Cancel) Marshal(out *bytes.Buffer) error {
	if err := c.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalCommon(
		c.userID,
		c.symbolID,
		c.orderID,
		out,
	); err != nil {
		return err
	}

	return nil
}

func (c *Cancel) Unmarshal(in *bytes.Buffer) error {
	if err := c.metadata.Unmarshal(in); err != nil {
		return err
	}

	userID, symbolID, orderID, err := unmarshalCommon(in)

	if err != nil {
		return err
	}

	c.orderID = orderID
	c.userID = userID
	c.symbolID = symbolID

	return nil
}

type Move struct {
	orderID  int64
	userID   int64
	symbolID int32
	toPrice  int64
	metadata _metadata
	_        struct{}
}

func NewMove() *Move {
	return &Move{}
}

func (c *Move) Code() int8 {
	return 3
}

func (m *Move) OrderID() int64 {
	return m.orderID
}

func (m *Move) UserID() int64 {
	return m.userID
}

func (m *Move) SymbolID() int32 {
	return m.symbolID
}

func (m *Move) ToPrice() int64 {
	return m.toPrice
}

func (m *Move) Marshal(out *bytes.Buffer) error {
	if err := m.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalCommon(
		m.userID,
		m.symbolID,
		m.orderID,
		out,
	); err != nil {
		return err
	}

	if err := serialization.WriteInt64(m.toPrice, out); err != nil {
		return err
	}

	return nil
}

func (m *Move) Unmarshal(in *bytes.Buffer) error {
	if err := m.metadata.Unmarshal(in); err != nil {
		return err
	}

	userID, symbolID, orderID, err := unmarshalCommon(in)

	if err != nil {
		return err
	}

	toPrice, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	m.orderID = orderID
	m.userID = userID
	m.symbolID = symbolID
	m.toPrice = toPrice

	return nil
}

type Reduce struct {
	orderID  int64
	userID   int64
	symbolID int32

	// quantity to be reduced, not to quantity
	quantity int64
	metadata _metadata
	_        struct{}
}

func NewReduce(
	orderID int64,
	symbolID int32,
	quantity int64,
) *Reduce {
	return &Reduce{
		orderID:  orderID,
		symbolID: symbolID,
		quantity: quantity,
	}
}

func (c *Reduce) Code() int8 {
	return 4
}

func (r *Reduce) OrderID() int64 {
	return r.orderID
}

func (r *Reduce) UserID() int64 {
	return r.userID
}

func (r *Reduce) SymbolID() int32 {
	return r.symbolID
}

func (r *Reduce) Quantity() int64 {
	return r.quantity
}

func (r *Reduce) Marshal(out *bytes.Buffer) error {
	if err := r.metadata.Marshal(out); err != nil {
		return err
	}

	if err := marshalCommon(
		r.userID,
		r.symbolID,
		r.orderID,
		out,
	); err != nil {
		return err
	}

	if err := serialization.WriteInt64(r.quantity, out); err != nil {
		return err
	}

	return nil
}

func (r *Reduce) Unmarshal(in *bytes.Buffer) error {
	if err := r.metadata.Unmarshal(in); err != nil {
		return err
	}

	userID, symbolID, orderID, err := unmarshalCommon(in)

	if err != nil {
		return err
	}

	quanitity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	r.orderID = orderID
	r.userID = userID
	r.symbolID = symbolID
	r.quantity = quanitity

	return nil
}

// TODO rename
func marshalCommon(
	userID int64,
	symbolID int32,
	orderID int64,
	out *bytes.Buffer,
) error {
	if err := serialization.WriteInt64(userID, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(symbolID, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(orderID, out); err != nil {
		return err
	}

	return nil
}

// TODO rename
func unmarshalCommon(in *bytes.Buffer) (int64, int32, int64, error) {
	userID, err := serialization.ReadInt64(in)

	if err != nil {
		return 0, 0, 0, err
	}

	symbolID, err := serialization.ReadInt32(in)

	if err != nil {
		return 0, 0, 0, err
	}

	orderID, err := serialization.ReadInt64(in)

	if err != nil {
		return 0, 0, 0, err
	}

	return userID, symbolID, orderID, nil
}
