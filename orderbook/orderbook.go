package orderbook

import "time"

type requestPayload struct {
	symbolId  int32
	size      int32
	timestamp time.Time
	_         struct{}
}
