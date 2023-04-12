package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xerexchain/matching-engine/serialization"
)

type QuantityError struct {
	OrderID int64 `json:"orderId"`
	Before  int64 `json:"before"`
	After   int64 `json:"after"`
	_       struct{}
}

func (e *QuantityError) Error() string {
	b, _ := json.Marshal(e)

	return string(b)
}

/*
 * `Category` is determined by `Command` (like `GTC`, etc).
 * `Symbol` is determined by `Orderbook`.
 * No external references allowed to such object.
 * `Order`(s) only live inside `OrderBook`.
 */
// TODO equals and hashCode overriden, timestamp ignored in equals, statehash impl
type Order struct {
	id       int64
	userID   int64
	price    int64
	quantity int64
	filled   int64

	// new orders - reserved price for fast moves of `GTC` bid orders in exchange mode
	// TODO logic
	reservedBidPrice int64  `json:"reservedBidPrice"`
	timestamp        int64  `json:"timestamp"`
	action           Action `json:"action"`
	_                struct{}
}

func New(
	id int64,
	userID int64,
	price int64,
	quantity int64,
	filled int64,
	reservedBidPrice int64,
	timestamp int64,
	action Action,
) *Order {
	return &Order{
		id:               id,
		userID:           userID,
		price:            price,
		quantity:         quantity,
		filled:           filled,
		reservedBidPrice: reservedBidPrice,
		timestamp:        timestamp,
		action:           action,
	}
}

func (o *Order) ID() int64 {
	return o.id
}

func (o *Order) UserID() int64 {
	return o.userID
}

func (o *Order) Price() int64 {
	return o.price
}

func (o *Order) ReservedBidPrice() int64 {
	return o.reservedBidPrice
}

func (o *Order) Timestamp() int64 {
	return o.timestamp
}

func (o *Order) Action() Action {
	return o.action
}

func (o *Order) Remained() int64 {
	return o.quantity - o.filled
}

func (o *Order) Fill(quantity int64) error {
	after := o.filled + quantity

	if after < 0 || after > o.quantity {
		return &QuantityError{
			OrderID: o.id,
			Before:  o.quantity,
			After:   after,
		}
	}

	o.filled += quantity

	return nil
}

func (o *Order) Reduce(quantity int64) error {
	return o.Fill(-quantity)
}

// TODO Order fields are not exported.
// func (o *Order) Hash() uint64 {
// 	hash, err := hashstructure.Hash(*o, hashstructure.FormatV2, nil)

// 	if err != nil {
// 		panic(err)
// 	}

// 	return hash
// }

func (o *Order) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(o.id, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.price, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.quantity, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.filled, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.reservedBidPrice, out); err != nil {
		return err
	}

	if err := serialization.WriteInt8(int8(o.action), out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.userID, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(o.timestamp, out); err != nil {
		return err
	}

	return nil
}

func (o *Order) Unmarshal(in *bytes.Buffer) error {
	id, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	price, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	quantity, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	filled, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	reservedBidPrice, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	code, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	action, ok := ActionFrom(code)

	if !ok {
		return fmt.Errorf("unmarshal: invalid action: %v", code)
	}

	userID, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	timestamp, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	o.id = id
	o.price = price
	o.quantity = quantity
	o.filled = filled
	o.reservedBidPrice = reservedBidPrice
	o.action = action
	o.userID = userID
	o.timestamp = timestamp

	return nil
}
