package direction

import "github.com/xerexchain/matching-engine/order/action"

type Direction struct {
	Multiplier int8
	_          struct{}
}

var (
	Long = Direction{
		Multiplier: 1,
	}
	Short = Direction{
		Multiplier: -1,
	}
	Empty = Direction{
		Multiplier: 0,
	}
)

var codeToDirection = map[int8]Direction{
	1:  Long,
	-1: Short,
	0:  Empty,
}

func (d Direction) IsOppositeTo(act action.Action) bool {
	return (d == Long && act == action.Ask) ||
		(d == Short && act == action.Bid)
}

func (d Direction) IsSameAs(act action.Action) bool {
	return (d == Long && act == action.Bid) ||
		(d == Short && act == action.Ask)
}

func FromAction(act action.Action) Direction {
	if act == action.Bid {
		return Long
	} else {
		return Short
	}
}

func FromByte(b int8) Direction {
	return codeToDirection[b]
}
