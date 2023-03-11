package state

// TODO EqualsAndHashCode
// TODO rename
type persistPayload struct {
	dumpId    int64
	seal      bool
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
// TODO rename
type resetPayload struct {
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
type nopPayload struct {
	timestamp int64
	_         struct{}
}
