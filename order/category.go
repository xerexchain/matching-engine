package order

type _category int8

const (
	// GTC (Good till Cancel) - equivalent to regular limit order
	GTC _category = iota + 1

	// IOC (Immediate or Cancel) - equivalent to strict-risk market order
	IOC       // with price cap
	IOCBudget // with total amount cap

	// FOK (Fill or Kill) - execute immediately completely or not at all
	FOC       // with price cap
	FOCBudget // total amount cap
)

var _categories = map[int8]_category{
	int8(GTC):       GTC,
	int8(IOC):       IOC,
	int8(IOCBudget): IOCBudget,
	int8(FOC):       FOC,
	int8(FOCBudget): FOCBudget,
}

func categoryFrom(code int8) (_category, bool) {
	category, ok := _categories[code]

	return category, ok
}
