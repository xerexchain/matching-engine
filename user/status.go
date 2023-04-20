package user

type Status int8

const (
	_active Status = iota + 1
	_suspended
)

var _statuses = map[int8]Status{
	int8(_active):    _active,
	int8(_suspended): _suspended,
}

func statusFrom(code int8) (Status, bool) {
	status, ok := _statuses[code]

	return status, ok
}
