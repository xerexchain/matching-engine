package user

import (
	"bytes"
	"github.com/xerexchain/matching-engine/serialization"
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
	t         BalanceAdjustmentType
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

func MarshalUsers(in map[int64]map[int32]int64, out *bytes.Buffer) error {
	return serialization.MarshalInt64Interface(
		in,
		out,
		serialization.MarshalInt32Int64,
	)
}

func UnmarshalUsers(in *bytes.Buffer) (interface{}, error) {
	return serialization.UnmarshalInt64Interface(
		in,
		serialization.UnmarshalInt32Int64,
	)
}
