package disk

// TODO rename

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/pierrec/lz4/v4"
	"github.com/xerexchain/matching-engine/cfg"
	"github.com/xerexchain/matching-engine/cmd"
	"github.com/xerexchain/matching-engine/serialization"
)

type module string // TODO rename? byte or string?

const (
	riskEngine           module = "RE"
	matchingEngineRouter module = "ME"
	maxOriginalSize      int32  = 1000000
	maxCompressedSize    int32  = 1000000
	maxCommandSizeBytes  int32  = 256
)

// TODO Comparable<SnapshotDescriptor>
// TODO compareTo
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
	_        struct{}
}

type journal struct {
	timestampNs  int64
	seqFirst     int64
	seqLast      int64 // -1 if not finished yet // TODO make sure to init -1
	baseSnapshot *snapshot
	prev         *journal
	next         *journal
	_            struct{}
}

// TODO rename?
type Processor interface {
	Store(int64, int64, int64, module, int32, serialization.Marshalable) bool
	Load(int64, module, int32, func(*bytes.Buffer) interface{}) interface{}

	// error in case of writing issue (will stop matching-engine from responding)
	WriteToJournal(cmd.Command, int64, bool) error

	// enable only after specified sequence, for lower sequences no writes to journal
	EnableJournalingAfter(int64)

	// sequential map of snapshots, int64 -> *Snapshot
	// Snapshots() *treemap.Map
	SnapshotExists(int64, module, int32) bool
}

// TODO rename?
type processor struct {
	diskCfg                *cfg.DiskProc
	initialStateCfg        *cfg.InitialState
	enableJournalAfterSeq  int64 // TODO default -1
	journalBufFlushTrigger int32
	journalBuf             *bytes.Buffer
	lz4Buf                 *bytes.Buffer
	// snapshotIndex          *treemap.Map // TODO ConcurrentSkipListMap // TODO default val
	lastSnapshot *snapshot
	lastJournal  *journal
	raf          *os.File // TODO RandomAccessFile, FileChannel // TODO rename
	fileCounter  int64
	writtenBytes int64
	mu           sync.RWMutex
	_            struct{}
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
	cfg cfg.InitialState,
	shardId int32,
	mod module,
) bool {
	if cfg.BaseSnapshotId != 0 {
		if p.SnapshotExists(
			cfg.BaseSnapshotId,
			mod,
			shardId,
		) {
			return true
		}

		if cfg.PanicIfSnapshotNotFound {
			panic(
				fmt.Sprintf(
					"Snapshot %v sharedId %v not found for %v", cfg.BaseSnapshotId, shardId, mod,
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

func (p *processor) Store(
	snapshotId int64,
	seq int64,
	timestampNs int64,
	mod module,
	instanceId int32,
	marshalable serialization.Marshalable,
) bool {
	p.mu.Lock()

	path := p.snapshotPath(snapshotId, mod, instanceId)
	f, err := os.Create(path)

	if err != nil {
		log.Printf("error: can not write snapshot file: %v", err)
		p.mu.Unlock()

		return false
	}

	defer f.Close()

	// TODO impl

	p.mu.Unlock()

	return true
}

func (p *processor) Load(
	snapshotId int64,
	mod module,
	instanceId int32,
	init func(*bytes.Buffer) interface{},
) interface{} {
	p.mu.Lock()

	path := p.snapshotPath(snapshotId, mod, instanceId)
	f, err := os.Open(path)

	if err != nil {
		p.mu.Unlock()
		log.Panicf("can not read snapshot file: %v", err)
	}

	defer f.Close()

	// TODO impl

	p.mu.Unlock()

	return nil
}

// TODO thread safe?
func (p *processor) WriteToJournal(
	command cmd.Command,
	dSeq int64, // distruptor sequence // TODO rename to seq?
	eob bool, // TODO rename
) error {
	if p.enableJournalAfterSeq == -1 || dSeq+p.initialStateCfg.BaseSnapshotSeq <= p.enableJournalAfterSeq {
		return nil
	}

	timestamp := command.Metadata().TimestampNs // TODO vs cmd.timestamp
	code := command.Code()

	if code == cmd.ShutdownSignal_ {
		p.flush(false, timestamp)

		return nil
	}

	// TODO
	// if (!cmdType.isMutate()) {
	// 	// skip queries
	// 	return;
	// }

	if p.raf == nil {
		p.newFile(timestamp) // TODO vs cmd.timestamp
	}

	if err := serialization.MarshalInt8(command.Code(), p.journalBuf); err != nil {
		return err
	}

	command.Metadata().Seq = dSeq + p.initialStateCfg.BaseSnapshotSeq
	command.Marshal(p.journalBuf)

	if code == cmd.PersistStateRisk_ {
		// p.registerNextSnapshot() // TODO
		// baseSnapshotId = TODO
		p.fileCounter = 0
		p.flush(true, timestamp)
	} else if code == cmd.Reset_ {
		p.flush(true, timestamp)
	} else if eob || p.journalBufFlushTrigger <= int32(p.journalBuf.Len()) {
		p.flush(false, timestamp)
	}

	return nil
}

func (p *processor) EnableJournalingAfter(seq int64) {
	p.enableJournalAfterSeq = seq
}

// func (p *processor) Snapshots() *treemap.Map {
// 	return p.snapshotIndex
// }

func (p *processor) SnapshotExists(
	snapshotId int64,
	mod module,
	instanceId int32,
) bool {
	path := p.snapshotPath(snapshotId, mod, instanceId)
	_, err := os.Stat(path)

	return err != nil
}

// TODO incompatible with exchange-core
// TODO types of uint8 vs byte (-128 to 127), action, order type, balance adj, ...
// TODO handle panic(s)
func (p *processor) readCommands(
	buf *bytes.Buffer,
	lastSeq *int64,
	insideCompressedBlock bool,
) ([]cmd.Command, error) {
	res := []cmd.Command{}

	for val, err := serialization.UnmarshalInt8(buf); err != nil; {
		emptyCommand, ok := cmd.From(val.(int8))

		if !ok {
			return nil, fmt.Errorf("unexpected command: %v", val)
		}

		if emptyCommand.Code() == cmd.ReservedCompressed_ {
			if insideCompressedBlock {
				return nil, errors.New("recursive compression block (data corrupted)")
			}

			var compSize int32

			if val, err := serialization.UnmarshalInt32(buf); err != nil {
				return nil, err
			} else {
				compSize = val.(int32)
			}

			if compSize > maxCompressedSize {
				return nil, fmt.Errorf("bad compressed block size = %v (data corrupted)", compSize)
			}

			var origSize int32

			if val, err := serialization.UnmarshalInt32(buf); err != nil {
				return nil, err
			} else {
				origSize = val.(int32)
			}

			if origSize > maxOriginalSize {
				return nil, fmt.Errorf("bad original block size = %v (data corrupted)", origSize)
			}

			origData := make([]byte, origSize)

			if _, err := lz4.UncompressBlock(
				buf.Bytes()[:compSize],
				origData,
			); err != nil {
				return nil, err
			} else {
				buf.Next(int(compSize))
			}

			if partialRes, err := p.readCommands(
				bytes.NewBuffer(origData),
				lastSeq,
				true,
			); err != nil {
				return nil, err
			} else {
				res = append(res, partialRes...)
			}
		} else {
			emptyCommand.Unmarshal(buf)
			command := emptyCommand
			seq := command.Metadata().Seq

			if seq != *lastSeq+1 {
				log.Printf("warn: Sequence gap %v->%v (%v)", lastSeq, seq, seq-*lastSeq)
			}

			*lastSeq = seq

			res = append(res, command)
		}
	}

	return res, nil
}

func (p *processor) flush(
	forceStartNextFile bool,
	timestampNs int64,
) error {
	length := int32(p.journalBuf.Len())

	if length < p.diskCfg.JournalBatchCompressThresholdBytes {
		if _, err := p.raf.Write(p.journalBuf.Bytes()); err != nil {
			// TODO reset journalBuf?
			return err
		}

		p.writtenBytes += int64(length)
	} else {
		prefixLen := 1 + 4 + 4
		p.lz4Buf.Reset()

		// indicates compressed block
		if err := serialization.MarshalInt8(cmd.ReservedCompressed_, p.lz4Buf); err != nil {
			return err
		}

		// reserve 4 bytes space for compressed length
		if err := serialization.MarshalInt32(0, p.lz4Buf); err != nil {
			return err
		}

		// uncompressed length
		if err := serialization.MarshalInt32(length, p.lz4Buf); err != nil {
			return err
		}

		n, err := p.diskCfg.JournalCompressor.Compress(
			p.journalBuf.Bytes(),
			p.lz4Buf.Bytes()[prefixLen:],
		)

		if err != nil {
			// TODO reset journalBuf?
			return err
		}

		if n == 0 {
			// TODO reset journalBuf?
			return errors.New("incompressible")
		}

		view := bytes.NewBuffer(p.lz4Buf.Bytes()[1:1])

		if err := serialization.MarshalInt32(int32(n), view); err != nil {
			return err
		}

		totalWritten := prefixLen + n
		if _, err = p.raf.Write(p.lz4Buf.Bytes()[:totalWritten]); err != nil {
			// TODO reset journalBuf?
			return err
		}

		p.lz4Buf.Reset()
		p.writtenBytes += int64(totalWritten)
	}

	p.journalBuf.Reset()

	if forceStartNextFile || p.diskCfg.JournalFileMaxSizeBytes <= p.writtenBytes {
		// TODO start preparing new file asynchronously, but ONLY ONCE
		p.newFile(timestampNs)
		p.writtenBytes = 0
	}

	return nil
}

func (p *processor) newFile(timestampNs int64) error {
	p.fileCounter++

	if p.raf != nil {
		p.raf.Close()
	}

	path := p.journalPath(p.fileCounter, p.initialStateCfg.BaseSnapshotId)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if f, err := os.OpenFile(
			path,
			os.O_CREATE|os.O_RDWR,
			0644, // TODO equals to rwd?
		); err == nil {
			p.raf = f
			p.registerNextJournal(p.initialStateCfg.BaseSnapshotId, timestampNs) // TODO fix time

			return nil
		}
	}

	return fmt.Errorf("can't create new file: %s", path)
}

// call only from journal thread
func (p *processor) registerNextJournal(
	seq int64,
	timestampNs int64,
) {
	p.lastJournal = &journal{
		timestampNs:  timestampNs,
		seqFirst:     seq,
		seqLast:      -1,
		baseSnapshot: p.lastSnapshot,
		prev:         p.lastJournal,
	}
}

// call only from journal thread
func (p *processor) registerNextSnapshot(
	snapshotId int64,
	seq int64,
	timestampNs int64,
) {
	p.lastSnapshot = p.lastSnapshot.createNext(
		snapshotId,
		seq,
		timestampNs,
	)
}

func (p *processor) mainLogPath() string {
	return fmt.Sprintf(
		"%s/%s.eca",
		p.diskCfg.StorageFolder,
		p.initialStateCfg.ExchangeId,
	)
}

func (p *processor) snapshotPath(
	snapshotId int64,
	mod module,
	instanceId int32,
) string {
	return fmt.Sprintf(
		"%s/%s_snapshot_%d_%s%d.ecs",
		p.diskCfg.StorageFolder,
		p.initialStateCfg.ExchangeId,
		snapshotId,
		mod,
		instanceId,
	)
}

func (p *processor) journalPath(
	partitionId, snapshotId int64,
) string {
	return fmt.Sprintf(
		"%s/%s_journal_%d_%04X.ecj",
		p.diskCfg.StorageFolder,
		p.initialStateCfg.ExchangeId,
		snapshotId,
		partitionId,
	)
}

func NewProcessor(
	diskCfg *cfg.DiskProc,
	initialStateCfg *cfg.InitialState,
	performanceCfg *cfg.Performance,

) Processor {
	// TODO journalFileMaxSize

	return &processor{
		diskCfg:                diskCfg,
		initialStateCfg:        initialStateCfg,
		enableJournalAfterSeq:  -1,
		journalBufFlushTrigger: diskCfg.JournalBufSizeBytes - maxCommandSizeBytes,
		journalBuf:             bytes.NewBuffer(make([]byte, 0, diskCfg.JournalBufSizeBytes)),
		lz4Buf:                 bytes.NewBuffer(make([]byte, 0, diskCfg.JournalBufSizeBytes)), // TODO size
		// snapshotIndex: treemap.NewWith(),
		lastSnapshot: &snapshot{
			numRiskEngines:     performanceCfg.NumRiskEngines,
			numMatchingEngines: performanceCfg.NumMatchingEngines,
		},
	}
}
