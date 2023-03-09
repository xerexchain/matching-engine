package symbol

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/serialization"
)

type symbol struct {
	Id int32
	Type
	Base       int32
	Quote      int32 // counter currency (OR futures contract currency)
	BaseScale  int64 // lot size in base currency units
	QuoteScale int64 // step size in quote currency units

	// fees per lot in quote? currency units
	TakerFee int64 // TODO check invariant: taker fee is not less than maker fee
	MakerFee int64

	// margin settings (for type=FUTURES_CONTRACT only)
	MarginBuy  int64
	MarginSell int64
	_          struct{}
}

type symbolBatchPayload struct {
	rawSymbols *bytes.Buffer
	timestamp  time.Time
	_          struct{}
}

func (s *symbol) Hash() uint64 {
	hash, err := hashstructure.Hash(s, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func Marshal(in interface{}, out *bytes.Buffer) error {
	s := in.(*symbol)

	if err := binary.Write(out, binary.LittleEndian, *s); err != nil {
		return err
	}

	return nil
}

func Unmarshal(in *bytes.Buffer) (interface{}, error) {
	s := &symbol{}

	if err := binary.Read(in, binary.LittleEndian, s); err != nil {
		return nil, err
	}

	return s, nil
}

func MarshalSymbols(in map[int32]*symbol, out *bytes.Buffer) error {
	return serialization.MarshalInt32Interface(in, out, Marshal)
}

func UnmarshalSymbols(in *bytes.Buffer) (interface{}, error) {
	res, err := serialization.UnmarshalInt32Interface(
		in,
		Unmarshal,
	)

	if err != nil {
		return nil, err
	}

	return res, nil
}
