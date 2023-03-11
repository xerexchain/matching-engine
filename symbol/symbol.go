package symbol

import (
	"bytes"
	"encoding/binary"

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
type FutureContractSymbol interface {
	Symbol
	MarginBuy() int64
	MarginSell() int64
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
// TODO Sym is *symbol, not symbol
// TODO equals overriden
// TODO complete implementation
type OptionSymbol interface {
	Symbol
}

type symbol struct {
	Id            int32
	BaseCurrency  int32
	QuoteCurrency int32
	BaseScaleK    int64 // lot size
	QuoteScaleK   int64 // step size

	// TODO fees per lot in quote? currency units
	TakerFee int64 // TODO check invariant: taker fee is not less than maker fee
	MakerFee int64
	_        struct{}
}

type futureContractSymbol struct {
	Sym *symbol

	MarginBuy_  int64 // quote currency
	MarginSell_ int64 // quote currency
	_           struct{}
}

type optionSymbol struct {
	_ struct{}
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

func (f *futureContractSymbol) MarginBuy() int64 {
	return f.MarginBuy_
}

func (f *futureContractSymbol) MarginSell() int64 {
	return f.MarginSell_
}

// TODO Sym is *symbol, not symbol
func (f *futureContractSymbol) Hash() uint64 {
	hash, err := hashstructure.Hash(*f, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (f *futureContractSymbol) Marshal(out *bytes.Buffer) error {
	return MarshalFutureContractSymbol(f, out)
}

// TODO This is incompatible with exchange-core: `bytes.writeByte(type.getCode());`
func MarshalSymbol(in interface{}, out *bytes.Buffer) error {
	s := in.(*symbol)

	if err := binary.Write(out, binary.LittleEndian, s.Id); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.BaseCurrency); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.QuoteCurrency); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.BaseScaleK); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.QuoteScaleK); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.TakerFee); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, s.MakerFee); err != nil {
		return err
	}

	return nil
}

func MarshalFutureContractSymbol(in interface{}, out *bytes.Buffer) error {
	f := in.(*futureContractSymbol)

	if err := f.Sym.Marshal(out); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, f.MarginBuy_); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, f.MarginSell_); err != nil {
		return err
	}

	return nil
}

// TODO This is incompatible with exchange-core: `SymbolType.of(bytes.readByte());`
func UnmarshalSymbol(in *bytes.Buffer) (interface{}, error) {
	s := symbol{}

	if err := binary.Read(in, binary.LittleEndian, &(s.Id)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.BaseCurrency)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.QuoteCurrency)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.BaseScaleK)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.QuoteScaleK)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.TakerFee)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(s.MakerFee)); err != nil {
		return nil, err
	}

	return &s, nil
}

func UnmarshalFutureContractSymbol(in *bytes.Buffer) (interface{}, error) {
	f := futureContractSymbol{}

	s, err := UnmarshalSymbol(in)

	if err != nil {
		return nil, err
	}

	f.Sym = s.(*symbol)

	if err := binary.Read(in, binary.LittleEndian, &(f.MarginBuy_)); err != nil {
		return nil, err
	}

	if err := binary.Read(in, binary.LittleEndian, &(f.MarginSell_)); err != nil {
		return nil, err
	}

	return &f, nil
}
