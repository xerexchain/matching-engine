package riskengine

import (
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
)

type LastPriceCacheRecord interface {
	state.Hashable
	serialization.Marshalable
	AskPrice() int64
	BidPrice() int64
}
