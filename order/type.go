package order

type Type int8

const (
	// Good till Cancel - equivalent to regular limit order
	GTC Type = iota

	// Immediate or Cancel - equivalent to strict-risk market order
	IOC       // with price cap
	IOCBudget // with total amount cap

	// Fill or Kill - execute immediately completely or not at all
	FOC       // with price cap
	FOCBudget // total amount cap
)

var codeToType = map[int8]Type{
	0: GTC,
	1: IOC,
	2: IOCBudget,
	3: FOC,
	4: FOCBudget,
}

// TODO rename
func FromCode(b int8) (Type, bool) {
	val, ok := codeToType[b]

	return val, ok
}
