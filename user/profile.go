package user

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/order/position"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

type Profile interface {
	state.Hashable
	serialization.Marshalable
	MarginPositionOf(symbolId int32) (position.MarginPosition, error)
}

type profile struct {
	UserId             int64
	AdjustmentsCounter int64 // protects from double adjustment
	Status
	Balance         map[int32]int64       // currency -> balance
	MarginPositions map[int32]interface{} // symbolId -> margin position
	_               struct{}
}

func (p *profile) MarginPositionOf(
	symbolId int32,
) (position.MarginPosition, error) {
	val, ok := p.MarginPositions[symbolId]

	if !ok {
		return nil, fmt.Errorf("not found position for symbol %v", symbolId)
	}

	return val.(position.MarginPosition), nil
}

// TODO This not equals to java stateHash
func (p *profile) Hash() uint64 {
	hash, err := hashstructure.Hash(*p, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

func (p *profile) Marshal(out *bytes.Buffer) error {
	return MarshalProfile(p, out)
}

func NewProfile(userId int64, status Status) Profile {
	return &profile{
		UserId:          userId,
		Status:          status,
		Balance:         make(map[int32]int64),
		MarginPositions: make(map[int32]interface{}),
	}
}

// TODO incompatible with exchange-core
func MarshalProfile(in interface{}, out *bytes.Buffer) error {
	p := in.(*profile)

	if err := binary.Write(out, binary.LittleEndian, p.UserId); err != nil {
		return err
	}

	if err := serialization.MarshalInt32Interface(
		p.MarginPositions,
		out,
		position.MarshalMarginPosition,
	); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, p.AdjustmentsCounter); err != nil {
		return err
	}

	if err := serialization.MarshalInt32Int64(p.Balance, out); err != nil {
		return err
	}

	if err := binary.Write(out, binary.LittleEndian, p.Status); err != nil {
		return err
	}

	return nil
}

// TODO incompatible with exchange-core
func UnmarshalProfile(in *bytes.Buffer) (interface{}, error) {
	p := profile{}

	if err := binary.Read(in, binary.LittleEndian, &(p.UserId)); err != nil {
		return nil, err
	}

	positions, err := serialization.UnmarshalInt32Interface(
		in,
		position.UnmarshalMarginPosition,
	)

	if err != nil {
		return nil, err
	}

	p.MarginPositions = positions.(map[int32]interface{})

	if err := binary.Read(in, binary.LittleEndian, &(p.AdjustmentsCounter)); err != nil {
		return nil, err
	}

	balance, err := serialization.UnmarshalInt32Int64(in)

	if err != nil {
		return nil, err
	}

	p.Balance = balance.(map[int32]int64)

	if err := binary.Read(in, binary.LittleEndian, &(p.Status)); err != nil {
		return nil, err
	}

	return &p, nil
}
