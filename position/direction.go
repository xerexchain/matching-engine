package position

import "github.com/xerexchain/matching-engine/order"

type Direction int8

const (
	_long  Direction = 1
	_short Direction = -1
	_empty Direction = 0
)

var _directions = map[int8]Direction{
	int8(_long):  _long,
	int8(_short): _short,
	int8(_empty): _empty,
}

func (d Direction) isOppositeTo(action order.Action) bool {
	return (d == _long && action == order.Ask) ||
		(d == _short && action == order.Bid)
}

func (d Direction) isSameAs(action order.Action) bool {
	return (d == _long && action == order.Bid) ||
		(d == _short && action == order.Ask)
}

func directionFromAction(action order.Action) Direction {
	if action == order.Bid {
		return _long
	} else {
		return _short
	}
}

func directionFromCode(code int8) (Direction, bool) {
	direction, ok := _directions[code]

	return direction, ok
}
