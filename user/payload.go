package user

import (
	"bytes"

	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/user/balance/adjustment/t"
)

// TODO EqualsAndHashCode
type payload struct {
	userId    int64
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
type balanceAdjustmentPayload struct {
	userId    int64
	currency  int32
	amount    int64
	txid      int64
	t         t.T
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
type batchPayload struct {
	transferId int64
	raw        *bytes.Buffer
	timestamp  int64
	_          struct{}
}

func (b *batchPayload) Code() int32 {
	return 1002
}

func MarshalUsers(in interface{}, out *bytes.Buffer) error {
	users := in.(map[int64]map[int32]int64)

	return serialization.MarshalMap(
		users,
		out,
		serialization.MarshalInt64,
		func(in interface{}, b *bytes.Buffer) error {
			return serialization.MarshalMap(
				in,
				out,
				serialization.MarshalInt32,
				serialization.MarshalInt64,
			)
		},
	)
}

func UnmarshalUser(b *bytes.Buffer) (interface{}, error) {
	return serialization.UnmarshalInt32Int64(b)
}

func UnmarshalUsers(b *bytes.Buffer) (interface{}, error) {
	var val interface{}
	var err error

	if val, err = serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	users := make(map[int64]map[int32]int64, size)

	for size > 0 {
		if k, err := serialization.UnmarshalInt64(b); err != nil {
			return nil, err
		} else {
			if v, err := UnmarshalUser(b); err != nil {
				return nil, err
			} else {
				users[k.(int64)] = v.(map[int32]int64)
			}
		}

		size--
	}

	return users, nil
}
