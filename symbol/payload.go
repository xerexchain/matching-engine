package symbol

import (
	"bytes"

	"github.com/xerexchain/matching-engine/serialization"
)

// TODO EqualsAndHashCode
type batchPayload struct {
	transferId int64
	raw        *bytes.Buffer
	timestamp  int64
	_          struct{}
}

func (b *batchPayload) Code() int32 {
	return 1003
}

// TODO implement MarshalFutureContractSymbols, UnmarshalFutureContractSymbols, MarshalOptionSymbols, UnmarshalOptionSymbols

func MarshalSymbols(in interface{}, out *bytes.Buffer) error {
	symbols := in.(map[int32]Symbol)

	return serialization.MarshalMap(
		symbols,
		out,
		serialization.MarshalInt32,
		MarshalSymbol,
	)
}

func UnmarshalSymbols(b *bytes.Buffer) (interface{}, error) {
	var val interface{}
	var err error

	if val, err = serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	symbols := make(map[int32]Symbol, size)

	for size > 0 {
		if k, v, err := serialization.UnmarshalKeyVal(
			b,
			serialization.UnmarshalInt32,
			UnmarshalSymbol,
		); err != nil {
			return nil, err
		} else {
			symbols[k.(int32)] = v.(Symbol)
		}

		size--
	}

	return symbols, nil
}
