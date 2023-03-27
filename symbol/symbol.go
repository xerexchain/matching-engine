package symbol

import (
	"bytes"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

// TODO equals overriden
type Symbol interface {
	state.Hashable
	serialization.Marshalable
}

// TODO equals overriden
type FutureContract interface {
	Symbol
	MarginBuy() int64
	MarginSell() int64
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
// TODO equals overriden
// TODO complete implementation
type Option interface {
	Symbol
}

type symbol struct {
	Id_            int32
	BaseCurrency_  int32
	QuoteCurrency_ int32
	BaseScaleK_    int64 // lot size
	QuoteScaleK_   int64 // step size

	// TODO fees per lot in quote? currency units
	TakerFee_ int64 // TODO check invariant: taker fee is not less than maker fee
	MakerFee_ int64
	_         struct{}
}

type futureContract struct {
	Symbol_     symbol
	MarginBuy_  int64 // quote currency
	MarginSell_ int64 // quote currency
	_           struct{}
}

type option struct {
	Symbol_ symbol
	_       struct{}
}

func (s *symbol) Hash() uint64 {
	hash, err := hashstructure.Hash(*s, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (s *symbol) Marshal(out *bytes.Buffer) error {
	return MarshalSymbol(s, out)
}

func (f *futureContract) MarginBuy() int64 {
	return f.MarginBuy_
}

func (f *futureContract) MarginSell() int64 {
	return f.MarginSell_
}

func (f *futureContract) Hash() uint64 {
	hash, err := hashstructure.Hash(*f, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (f *futureContract) Marshal(out *bytes.Buffer) error {
	return MarshalFutureContract(f, out)
}

// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
func MarshalSymbol(in interface{}, out *bytes.Buffer) error {
	s := in.(*symbol)

	if err := serialization.MarshalInt32(s.Id_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(s.BaseCurrency_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt32(s.QuoteCurrency_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.BaseScaleK_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.QuoteScaleK_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.TakerFee_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(s.MakerFee_, out); err != nil {
		return err
	}

	return nil
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
func UnmarshalSymbol(b *bytes.Buffer) (interface{}, error) {
	s := symbol{}

	if val, err := serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	} else {
		s.Id_ = val.(int32)
	}

	if val, err := serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	} else {
		s.BaseCurrency_ = val.(int32)
	}

	if val, err := serialization.UnmarshalInt32(b); err != nil {
		return nil, err
	} else {
		s.QuoteCurrency_ = val.(int32)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		s.BaseScaleK_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		s.QuoteScaleK_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		s.TakerFee_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		s.MakerFee_ = val.(int64)
	}

	return &s, nil
}

func MarshalFutureContract(in interface{}, out *bytes.Buffer) error {
	f := in.(*futureContract)

	if err := MarshalSymbol(&(f.Symbol_), out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(f.MarginBuy_, out); err != nil {
		return err
	}

	if err := serialization.MarshalInt64(f.MarginSell_, out); err != nil {
		return err
	}

	return nil
}

func UnmarshalFutureContract(b *bytes.Buffer) (interface{}, error) {
	f := futureContract{}

	if val, err := UnmarshalSymbol(b); err != nil {
		return nil, err
	} else {
		f.Symbol_ = *(val.(*symbol)) // TODO performance
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		f.MarginBuy_ = val.(int64)
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		f.MarginSell_ = val.(int64)
	}

	return &f, nil
}
