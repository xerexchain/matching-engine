package order

import (
	"bytes"
	"encoding/json"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/order/action"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

// TODO type (GTC,...), Symbol
// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type Order interface {
	state.Hashable
	serialization.Marshalable
	Id() int64
	UserId() int64
	Price() int64
	Filled() int64
	Remained() int64
	Fill(int64)
	Reduce(int64)
	Action() action.Action
	ReservedBidPrice() int64
	Timestamp() int64
	String() string
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
// No external references allowed to such object - order objects only live inside OrderBook.
type order struct {
	Id_               int64
	UserId_           int64
	Price_            int64
	Quantity_         int64
	Filled_           int64
	ReservedBidPrice_ int64 // new orders - reserved price for fast moves of GTC bid orders in exchange mode // TODO logic
	Timestamp_        int64
	Action_           action.Action
}

func (o *order) Id() int64 {
	return o.Id_
}

func (o *order) UserId() int64 {
	return o.UserId_
}

func (o *order) Price() int64 {
	return o.Price_
}

func (o *order) Filled() int64 {
	return o.Filled_
}

func (o *order) Remained() int64 {
	return o.Quantity_ - o.Filled_
}

func (o *order) Fill(quantity int64) {
	o.Filled_ += quantity

	if o.Quantity_ < 0 {
		panic("Fill: reduced to less than zero")
	}

	if o.Filled_ > o.Quantity_ {
		panic("Fill: filled more than quantity")
	}
}

func (o *order) Reduce(quantity int64) {
	o.Quantity_ -= quantity

	if o.Quantity_ < 0 {
		panic("Reduce: reduced to less than zero")
	}

	if o.Filled_ > o.Quantity_ {
		panic("Reduce: filled more than quantity")
	}
}

func (o *order) Action() action.Action {
	return o.Action_
}

func (o *order) ReservedBidPrice() int64 {
	return o.ReservedBidPrice_
}

func (o *order) Timestamp() int64 {
	return o.Timestamp_
}

func (o *order) String() string {
	b, _ := json.Marshal(o)

	return string(b)
}

func (o *order) Hash() uint64 {
	hash, err := hashstructure.Hash(*o, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (o *order) Marshal(out *bytes.Buffer) error {
	return Marshal(o, out)
}

func Marshal(in interface{}, out *bytes.Buffer) error {
	o := in.(*order)

	if err := serialization.MarshalInt64(o.Id_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.Price_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.Quantity_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.Filled_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.ReservedBidPrice_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt8(int8(o.Action_), out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.UserId_, out); err != nil {
		return err
	}
	if err := serialization.MarshalInt64(o.Timestamp_, out); err != nil {
		return err
	}

	return nil
}

func UnMarshal(b *bytes.Buffer) (interface{}, error) {
	o := order{}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.Id_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.Price_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.Quantity_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.Filled_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.ReservedBidPrice_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt8(b); err != nil {
		return nil, err
	} else {
		o.Action_ = action.FromByte(val.(int8))
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.UserId_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		o.Timestamp_ = val.(int64)
	}

	return &o, nil
}

func New(
	id int64,
	userId int64,
	price int64,
	quantity int64,
	filled int64,
	reservedBidPrice int64,
	timestamp int64,
	action action.Action,
) Order {
	return &order{
		Id_:               id,
		UserId_:           userId,
		Price_:            price,
		Quantity_:         quantity,
		Filled_:           filled,
		ReservedBidPrice_: reservedBidPrice,
		Timestamp_:        timestamp,
		Action_:           action,
	}
}
