package user

import (
	"bytes"
	"time"

	"github.com/xerexchain/matching-engine/serialization"
)

type payload struct {
	id        int64
	timestamp time.Time
	_         struct{}
}

type balanceAdjustmentPayload struct {
	userId    int64
	currency  int32
	amount    int64
	txid      int64
	t         BalanceAdjustmentType
	timestamp time.Time
	_         struct{}
}

type userBatchPayload struct {
	rawUsers  *bytes.Buffer
	timestamp time.Time
	_         struct{}
}

func UnmarshalUsers(in *bytes.Buffer) (interface{}, error) {
	res, err := serialization.UnmarshalInt64Interface(
		in,
		serialization.UnmarshalInt32Int64,
	)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func MarshalUsers(in map[int64]map[int32]int64, out *bytes.Buffer) error {
	return serialization.MarshalInt64Interface(in, out, serialization.MarshalInt32Int64)
}
