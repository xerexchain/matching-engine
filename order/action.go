package order

type Action int8

const (
	// To sell
	Ask Action = iota + 1

	// To buy
	Bid
)

var _actions = map[int8]Action{
	int8(Ask): Ask,
	int8(Bid): Bid,
}

func ActionFrom(code int8) (Action, bool) {
	action, ok := _actions[code]

	return action, ok
}
