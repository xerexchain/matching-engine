package state

import "time"

type payload struct {
	dumpId    int64
	seal      bool
	timestamp time.Time
	_         struct{}
}

type resetPayload struct {
	timestamp time.Time
	_         struct{}
}

type nopPayload struct {
	timestamp time.Time
	_         struct{}
}
