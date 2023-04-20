package user

type BalanceAdjCategory int8

const (
	_adjustment BalanceAdjCategory = iota + 1
	_suspend
)

var _int8ToBalanceAdjCategory = map[int8]BalanceAdjCategory{
	int8(_adjustment): _adjustment,
	int8(_suspend):    _suspend,
}

func BalanceAdjCategoryFrom(code int8) (BalanceAdjCategory, bool) {
	val, ok := _int8ToBalanceAdjCategory[code]

	return val, ok
}
