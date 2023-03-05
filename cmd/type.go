package cmd

type Type struct {
	code   int8
	mutate bool
}

var (
	PlaceOrder  = &Type{1, true}
	CancelOrder = &Type{2, true}
	MoveOrder   = &Type{3, true}
	ReduceOrder = &Type{4, true}

	OrderBookRequest = &Type{6, false}

	AddUser           = &Type{10, true}
	BalanceAdjustment = &Type{11, true}
	SuspendUser       = &Type{12, true}
	ResumeUser        = &Type{13, true}

	BinaryDataQuery   = &Type{90, false}
	BinaryDataCommand = &Type{91, true}

	PersistStateMatching = &Type{110, true}
	PersistStateRisk     = &Type{111, true}

	GroupingControl = &Type{118, false}
	NOP             = &Type{120, false}
	Reset           = &Type{124, true}
	ShutdownSignal  = &Type{127, false}

	ReservedCompressed = &Type{-1, false}
)
