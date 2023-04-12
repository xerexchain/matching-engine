package user

type Status int8

const (
	Active Status = iota + 1
	Suspended
)

func StatusFromByte(b int8) Status {
	switch b {
	case int8(Active):
		return Active
	case int8(Suspended):
		return Suspended
	default:
		panic("Undefined user status")
	}
}
