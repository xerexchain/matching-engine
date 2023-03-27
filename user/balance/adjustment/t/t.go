package t

type T int8

const (
	Adjustment T = iota
	Suspend
)

var codeToT = map[int8]T{
	0: Adjustment,
	1: Suspend,
}

func From(b int8) (T, bool) {
	val, ok := codeToT[b]

	return val, ok
}
