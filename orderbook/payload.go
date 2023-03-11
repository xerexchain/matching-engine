package orderbook

// TODO EqualsAndHashCode
// TODO rename
type payload struct {
	symbolId  int32
	quantity  int32 // TODO doc
	timestamp int64
	_         struct{}
}
