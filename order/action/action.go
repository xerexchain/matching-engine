package action

type Action int8

const (
	Ask Action = iota
	Bid
)

func FromByte(b int8) Action {
	switch b {
	case int8(Ask):
		return Ask
	case int8(Bid):
		return Bid
	default:
		panic("Undefined action")
	}
}
