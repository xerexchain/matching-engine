package disk

// TODO rename

import (
	"bytes"
	"fmt"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/xerexchain/matching-engine/cmd"
	"github.com/xerexchain/matching-engine/config"
	"github.com/xerexchain/matching-engine/serialization"
)

type module string // TODO rename? byte?

const (
	riskEngine           module = "RE"
	matchingEngineRouter module = "ME"
)

// TODO Comparable<SnapshotDescriptor>
type snapshot struct {
	id                 int64 // 0 means empty snapshot (clean start)
	seq                int64
	timestampNs        int64
	numRiskEngines     int32
	numMatchingEngines int32
	prev               *snapshot
	next               *snapshot // TODO can be a list

	// all journals based on this snapshot
	// mapping: startingSeq -> Journal
	journals *treemap.Map
}

type journal struct {
	timestampNs  int64
	seqFirst     int64
	seqLast      int64 // -1 if not finished yet
	baseSnapshot *snapshot
	prev         *journal
	next         *journal
}

// TODO rename?
type Processor interface {
	Store(int64, int64, int64, module, int32, serialization.Marshalable) bool
	Load(int64, module, int32, func(*bytes.Buffer) interface{}) interface{}

	// error in case of writing issue (will stop matching-engine from responding)
	WriteToJournal(cmd.OrderCommand, int64, bool) error

	// enable only after specified sequence, for lower sequences no writes to journal
	EnableJournalingAfter(int64)

	// int64 -> *Snapshot
	Snapshots() *treemap.Map
	SnapshotExists(int64, module, int32) bool
}

func snapshotComparator(a, b interface{}) int {
	aAsserted := a.(int64)
	bAsserted := b.(int64)

	switch {
	case aAsserted > bAsserted:
		return 1
	case aAsserted < bAsserted:
		return -1
	default:
		return 0
	}
}
func canLoadFromSnapshot(
	p Processor,
	cfg config.InitialState,
	shardId int32,
	mod module,
) bool {
	if cfg.FromSnapshot() {
		if p.SnapshotExists(
			cfg.SnapshotId(),
			mod,
			shardId,
		) {
			return true
		}

		if cfg.PanicIfSnapshotNotFound() {
			panic(
				fmt.Sprintf(
					"Snapshot %v sharedId %v not found for %v", cfg.SnapshotId(), shardId, mod,
				),
			)
		}
	}

	return false
}

func (s *snapshot) createNext(snapshotId, seq, timestampNs int64) *snapshot {
	return &snapshot{
		id:                 snapshotId,
		seq:                seq,
		timestampNs:        timestampNs,
		numRiskEngines:     s.numRiskEngines,
		numMatchingEngines: s.numMatchingEngines,
		prev:               s,
		journals:           treemap.NewWith(snapshotComparator),
	}
}

func newSnapshot(
	numRiskEngines int32,
	numMatchingEngines int32,
) *snapshot {
	return &snapshot{
		numRiskEngines:     numRiskEngines,
		numMatchingEngines: numMatchingEngines,
		journals:           treemap.NewWith(snapshotComparator),
	}
}
