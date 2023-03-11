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

// TODO implement MarshalFutureContractSymbol, UnmarshalFutureContractSymbol, MarshalOptionSymbol, UnmarshalOptionSymbol

func MarshalSymbols(in map[int32]Symbol, out *bytes.Buffer) error {
	return serialization.MarshalInt32Interface(in, out, MarshalSymbol)
}

func UnmarshalSymbols(in *bytes.Buffer) (interface{}, error) {
	return serialization.UnmarshalInt32Interface(
		in,
		UnmarshalSymbol,
	)
}
