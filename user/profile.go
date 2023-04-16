package user

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/position"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

type Profile interface {
	state.Hashable
	serialization.Marshalable
	MarginPositionOf(symbolId int32) (*position.Margin, error)
}

type profile struct {
	UserId             int64
	AdjustmentsCounter int64 // protects from double adjustment
	Status
	Balance         map[int32]int64            // currency -> balance
	MarginPositions map[int32]*position.Margin // symbolId -> margin position
	_               struct{}
}

func (p *profile) MarginPositionOf(
	symbolId int32,
) (*position.Margin, error) {
	if pos, ok := p.MarginPositions[symbolId]; !ok {
		return nil, fmt.Errorf("not found position for symbol %v", symbolId)
	} else {
		return pos, nil
	}
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
		MarginPositions: make(map[int32]*position.Margin),
	}
}

// TODO incompatible with exchange-core
func MarshalProfile(in interface{}, out *bytes.Buffer) error {
	p := in.(*profile)

	if err := serialization.MarshalInt64(p.UserId, out); err != nil {
		return err
	}

	size := int32(len(p.MarginPositions))

	if err := serialization.WriteInt32(size, out); err != nil {
		return err
	}

	for symbolID, margin := range p.MarginPositions {

		if err := serialization.WriteInt32(symbolID, out); err != nil {
			return err
		}

		if err := margin.Marshal(out); err != nil {
			return err
		}
	}

	if err := serialization.MarshalInt64(p.AdjustmentsCounter, out); err != nil {
		return err
	}

	if err := serialization.MarshalMap(
		p.Balance,
		out,
		serialization.MarshalInt32,
		serialization.MarshalInt64,
	); err != nil {
		return err
	}

	if err := serialization.MarshalInt8(int8(p.Status), out); err != nil {
		return err
	}

	return nil
}

func UnmarshalBalance(b *bytes.Buffer) (interface{}, error) {
	return serialization.UnmarshalInt32Int64(b)
}

// TODO incompatible with exchange-core
func UnmarshalProfile(b *bytes.Buffer) (interface{}, error) {
	p := profile{}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		p.UserId = val.(int64)
	}

	if positions, err := position.UnmarshalMargins(b); err != nil {
		return nil, err
	} else {
		p.MarginPositions = positions
	}

	if val, err := serialization.UnmarshalInt64(b); err != nil {
		return nil, err
	} else {
		p.AdjustmentsCounter = val.(int64)
	}

	if balance, err := UnmarshalBalance(b); err != nil {
		return nil, err
	} else {
		p.Balance = balance.(map[int32]int64)
	}

	if val, err := serialization.UnmarshalInt8(b); err != nil {
		return nil, err
	} else {
		p.Status = StatusFromByte(val.(int8))
	}

	return &p, nil
}
