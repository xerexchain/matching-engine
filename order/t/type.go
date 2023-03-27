package t

type T int8

const (
	// Good till Cancel - equivalent to regular limit order
	GTC T = iota

	// Immediate or Cancel - equivalent to strict-risk market order
	IOC       // with price cap
	IOCBudget // with total amount cap

	// Fill or Kill - execute immediately completely or not at all
	FOC       // with price cap
	FOCBudget // total amount cap
)

var codeToT = map[int8]T{
	0: GTC,
	1: IOC,
	2: IOCBudget,
	3: FOC,
	4: FOCBudget,
}

func From(b int8) (T, bool) {
	val, ok := codeToT[b]

	return val, ok
}
