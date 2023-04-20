package user

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/xerexchain/matching-engine/position"
	"github.com/xerexchain/matching-engine/serialization"
)

type Profile struct {
	userID int64

	// protects from double adjustment
	adjustmentsCounter int64
	status             Status

	// key: symbolID
	marginPositions map[int32]*position.Margin

	// currency -> balance
	balances map[int32]int64
	_        struct{}
}

func NewProfile(userID int64, status Status) *Profile {
	return &Profile{
		userID:          userID,
		status:          status,
		balances:        make(map[int32]int64),
		marginPositions: make(map[int32]*position.Margin),
	}
}

func (p *Profile) MarginPositionOf(
	symbolID int32,
) (*position.Margin, bool) {
	position_, ok := p.marginPositions[symbolID]

	return position_, ok
}

// TODO This is not equal to java stateHash.
// TODO unexported fields
// TODO panic?
func (p *Profile) Hash() uint64 {
	hash, err := hashstructure.Hash(*p, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	return hash
}

// TODO incompatible with exchange-core
func (p *Profile) Marshal(out *bytes.Buffer) error {
	if err := serialization.WriteInt64(p.userID, out); err != nil {
		return err
	}

	positionsSize := int32(len(p.marginPositions))

	if err := serialization.WriteInt32(positionsSize, out); err != nil {
		return err
	}

	for symbolID, margin := range p.marginPositions {
		if err := serialization.WriteInt32(symbolID, out); err != nil {
			return err
		}

		if err := margin.Marshal(out); err != nil {
			return err
		}
	}

	if err := serialization.WriteInt64(p.adjustmentsCounter, out); err != nil {
		return err
	}

	balancesSize := int32(len(p.balances))

	if err := serialization.WriteInt32(balancesSize, out); err != nil {
		return err
	}

	for currency, balance := range p.balances {
		if err := serialization.WriteInt32(currency, out); err != nil {
			return err
		}
		if err := serialization.WriteInt64(balance, out); err != nil {
			return err
		}
	}

	if err := serialization.WriteInt8(int8(p.status), out); err != nil {
		return err
	}

	return nil
}

// TODO incompatible with exchange-core
func (p *Profile) Unmarshal(in *bytes.Buffer) error {

	userID, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	positionsSize, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	marginPositions := make(map[int32]*position.Margin, positionsSize)

	for ; positionsSize > 0; positionsSize-- {
		symbolID, err := serialization.ReadInt32(in)

		if err != nil {
			return err
		}

		margin := &position.Margin{}

		if err := margin.Unmarshal(in); err != nil {
			return err
		}

		marginPositions[symbolID] = margin
	}

	adjustmentsCounter, err := serialization.ReadInt64(in)

	if err != nil {
		return err
	}

	balancesSize, err := serialization.ReadInt32(in)

	if err != nil {
		return err
	}

	balances := make(map[int32]int64, balancesSize)

	for ; balancesSize > 0; balancesSize-- {
		currency, err := serialization.ReadInt32(in)

		if err != nil {
			return err
		}

		balance, err := serialization.ReadInt64(in)

		if err != nil {
			return err
		}

		balances[currency] = balance
	}

	code, err := serialization.ReadInt8(in)

	if err != nil {
		return err
	}

	status, ok := statusFrom(code)

	if !ok {
		return fmt.Errorf("Profile.Unmarshal: status: %v", code)
	}

	p.userID = userID
	p.marginPositions = marginPositions
	p.adjustmentsCounter = adjustmentsCounter
	p.balances = balances
	p.status = status

	return nil
}
