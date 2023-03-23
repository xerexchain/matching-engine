package api

import (
	"github.com/xerexchain/matching-engine/cfg"
	"github.com/xerexchain/matching-engine/disk"
)

type Exchange interface {
	ReplayJournalFull(cfg.InitialState, disk.Processor)
}

type exchange struct {
}

func (e *exchange) ReplayJournalFull(
	conf cfg.InitialState,
	proc disk.Processor,
) {
	// TODO implement from DiskSerializationProcessor.java
}
