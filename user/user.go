package user

type BalanceAdjType int8

const (
	Adjustment BalanceAdjType = iota
	Suspend
)

var codeToBalanceAdjType = map[int8]BalanceAdjType{
	0: Adjustment,
	1: Suspend,
}

// TODO rename
func FromCode(b int8) (BalanceAdjType, bool) {
	val, ok := codeToBalanceAdjType[b]

	return val, ok
}
