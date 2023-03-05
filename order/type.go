package order

type Type int

const (
	// Good till Cancel - equivalent to regular limit order
	GTC Type = iota

	// Immediate or Cancel - equivalent to strict-risk market order
	IOC        // with price cap
	IOC_BUDGET // with total amount cap

	// Fill or Kill - execute immediately completely or not at all
	FOC        // with price cap
	FOC_BUDGET // total amount cap
)
