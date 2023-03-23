package action

type Action int8

const (
	Ask Action = iota
	Bid
)

var codeToAction = map[int8]Action{
	0: Ask,
	1: Bid,
}

func FromCode(b int8) (Action, bool) {
	val, ok := codeToAction[b]
	return val, ok
}
