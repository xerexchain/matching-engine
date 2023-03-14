package order

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
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
	Quantity() int64
	Filled() int64
	Remained() int64
	Fill(int64)
	Action() Action
	ReservedBidPrice() int64 // new orders - reserved price for fast moves of GTC bid orders in exchange mode // TODO doc
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type Move interface {
	OrderId() int64
	UserId() int64
	NewPrice() int64 // TODO rename
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type Reduce interface {
	OrderId() int64
	UserId() int64
	ReduceQuantity() int64 // TODO rename
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type Cancel interface {
	OrderId() int64
	UserId() int64
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
// No external references allowed to such object - order objects only live inside OrderBook.
type order struct {
	Id_              int64
	UserId_          int64
	Price_           int64
	Quantity_        int64
	Filled_          int64
	ReserveBidPrice_ int64
	Timestamp_       int64
	Action_          Action
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type move struct {
	OrderId_  int64
	UserId_   int64
	NewPrice_ int64
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type reduce struct {
	OrderId_        int64
	UserId_         int64
	ReduceQuantity_ int64
}

// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type cancel struct {
	OrderId_ int64
	UserId_  int64
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
func (o *order) Quantity() int64 {
	return o.Quantity_
}
func (o *order) Filled() int64 {
	return o.Filled_
}
func (o *order) Remained() int64 {
	return o.Quantity_ - o.Filled_
}
func (o *order) Fill(quantity int64) {
	o.Filled_ += quantity

	if o.Filled_ > o.Quantity_ {
		panic("filled more than quantity")
	}
}
func (o *order) Action() Action {
	return o.Action_
}
func (o *order) ReserveBidPrice() int64 {
	return o.ReserveBidPrice_
}

func (o *order) Hash() uint64 {
	hash, err := hashstructure.Hash(*o, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (o *order) Marshal(out *bytes.Buffer) error {
	return MarshalOrder(o, out)
}

func MarshalOrder(in interface{}, out *bytes.Buffer) error {
	o := in.(*order)

	if err := binary.Write(out, binary.LittleEndian, o.Id_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.Price_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.Quantity_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.Filled_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.ReserveBidPrice_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.Action_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.UserId_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, o.Timestamp_); err != nil {
		return err
	}

	return nil
}

func UnMarshalOrder(in *bytes.Buffer) (interface{}, error) {
	o := order{}

	if err := binary.Read(in, binary.LittleEndian, &(o.Id_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.Price_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.Quantity_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.Filled_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.ReserveBidPrice_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.Action_)); err != nil {
		return nil, err
	}

	if o.Action_ != Ask && o.Action_ != Bid {
		return nil, fmt.Errorf("invalid action: %v", o.Action_)
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.UserId_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(o.Timestamp_)); err != nil {
		return nil, err
	}

	return &o, nil
}
