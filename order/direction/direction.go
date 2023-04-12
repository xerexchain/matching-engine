package direction

import "github.com/xerexchain/matching-engine/order"

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

func (d Direction) IsOppositeTo(action order.Action) bool {
	return (d == Long && action == order.Ask) ||
		(d == Short && action == order.Bid)
}

func (d Direction) IsSameAs(action order.Action) bool {
	return (d == Long && action == order.Bid) ||
		(d == Short && action == order.Ask)
}

func FromAction(action order.Action) Direction {
	if action == order.Bid {
		return Long
	} else {
		return Short
	}
}

func FromByte(b int8) Direction {
	return codeToDirection[b]
}
