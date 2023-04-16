package symbol

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

type _Symbol interface {
	state.Hashable
	serialization.Marshalable
	serialization.Unmarshalable
	ID() int32
}

// TODO equals overriden
type Symbol struct {
	id            int32
	baseCurrency  int32
	quoteCurrency int32
	baseScaleK    int64 // lot size
	quoteScaleK   int64 // step size

	// TODO fees per lot in quote? currency units
	// TODO check invariant: taker fee is not less than maker fee
	takerFee int64
	makerFee int64
	_        struct{}
}

func (s *Symbol) ID() int32 {
	return s.id
}

// TODO unexported fields
// TODO remove panic?
func (s *Symbol) Hash() uint64 {
	hash, err := hashstructure.Hash(*s, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
func (s *Symbol) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt8(int8(_currencyExchangePair), out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(s.id, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(s.baseCurrency, out); err != nil {
		return err
	}

	if err := serialization.WriteInt32(s.quoteCurrency, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(s.baseScaleK, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(s.quoteScaleK, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(s.takerFee, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(s.makerFee, out); err != nil {
		return err
	}

	return nil
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
func (s *Symbol) Unmarshal(in *bytes.Buffer) error {
	code, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	if _, ok := categoryFrom(code); !ok {
		return fmt.Errorf("Symbol.Unmarshal: category: %v", code)
	}

	id, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	baseCurrency, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	quoteCurrency, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	baseScaleK, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	quoteScaleK, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	takerFee, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	makerFee, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	s.id = id
	s.baseCurrency = baseCurrency
	s.quoteCurrency = quoteCurrency
	s.baseScaleK = baseScaleK
	s.quoteScaleK = quoteScaleK
	s.takerFee = takerFee
	s.makerFee = makerFee

	return nil
}

// TODO equals overriden
type FutureContract struct {
	symbol     Symbol
	marginBuy  int64 // quote currency
	marginSell int64 // quote currency
	_          struct{}
}

func (s *FutureContract) ID() int32 {
	return s.symbol.ID()
}

func (f *FutureContract) MarginBuy() int64 {
	return f.marginBuy
}

func (f *FutureContract) MarginSell() int64 {
	return f.marginSell
}

// TODO unexported fields
// TODO remove panic?
func (f *FutureContract) Hash() uint64 {
	hash, err := hashstructure.Hash(*f, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (f *FutureContract) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt8(int8(_futureContract), out); err != nil {
		return err
	}

	if err := f.symbol.Marshal(out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(f.marginBuy, out); err != nil {
		return err
	}

	if err := serialization.WriteInt64(f.marginSell, out); err != nil {
		return err
	}

	return nil
}

func (f *FutureContract) Unmarshal(in *bytes.Buffer) error {
	code, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	if _, ok := categoryFrom(code); !ok {
		return fmt.Errorf("FutureContract.Unmarshal: category: %v", code)
	}

	symbol_ := &Symbol{}

	if err := symbol_.Unmarshal(in); err != nil {
		return err
	}

	marginBuy, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	marginSell, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	f.symbol = *symbol_
	f.marginBuy = marginBuy
	f.marginSell = marginSell

	return nil
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
// TODO equals overriden
// TODO complete implementation
type Option struct {
	symbol Symbol
	_      struct{}
}

func Unmarshal(in *bytes.Buffer) (_Symbol, error) {
	code, err := serialization.ReadInt8(in)

	if err != nil {
		return nil, err
	}

	if _, ok := categoryFrom(code); !ok {
		return nil, fmt.Errorf("Unmarshal: category: %v", code)
	}

	f := _factory[code]
	symbol_ := f()

	if err := symbol_.Unmarshal(in); err != nil {
		return nil, err
	}

	return symbol_, nil
}
