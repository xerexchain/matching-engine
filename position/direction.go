package position

import "github.com/xerexchain/matching-engine/order"

type direction int8

const (
	_long  direction = 1
	_short direction = -1
	_empty direction = 0
)

var _int8ToDirection = map[int8]direction{
	int8(_long):  _long,
	int8(_short): _short,
	int8(_empty): _empty,
}

func (d direction) isOppositeTo(action order.Action) bool {
	return (d == _long && action == order.Ask) ||
		(d == _short && action == order.Bid)
}

func (d direction) isSameAs(action order.Action) bool {
	return (d == _long && action == order.Bid) ||
		(d == _short && action == order.Ask)
}

func directionFromAction(action order.Action) direction {
	if action == order.Bid {
		return _long
	} else {
		return _short
	}
}

func directionFromInt8(code int8) direction {
	return _int8ToDirection[code]
}
