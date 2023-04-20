package journaling

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/pierrec/lz4/v4"
	"github.com/xerexchain/matching-engine/cmd"
	"github.com/xerexchain/matching-engine/serialization"
)

// TODO byte or string?
type Category string

const (
	_riskEngine           Category = "RE"
	_matchingEngineRouter Category = "ME"
)

const (
	_maxOriginalSizeBytes   int32 = 1000000
	_maxCompressedSizeBytes int32 = 1000000
	_maxCommandSizeBytes    int32 = 256
)

var (
	errRecursiveCompressionBlock = errors.New("recursive compression block")
	errIncompressible            = errors.New("incompressible")
)

// TODO Comparable<SnapshotDescriptor>
// TODO compareTo
type Snapshot struct {
	// 0 means empty snapshot (clean start)
	id                 int64
	seq                int64
	timestampNS        int64
	numRiskEngines     int32
	numMatchingEngines int32

	prev *Snapshot

	// TODO can be a list
	next *Snapshot

	// all journals based on this snapshot
	// startingSeq -> Journal
	journals *treemap.Map

	_ struct{}
}

func newSnapshot(
	numRiskEngines int32,
	numMatchingEngines int32,
) *Snapshot {
	return &Snapshot{
		numRiskEngines:     numRiskEngines,
		numMatchingEngines: numMatchingEngines,
		journals:           treemap.NewWith(snapshotComparator),
	}
}

func (s *Snapshot) createNext(
	snapshotID int64,
	seq int64,
	timestampNS int64,
) *Snapshot {
	return &Snapshot{
		id:                 snapshotID,
		seq:                seq,
		timestampNS:        timestampNS,
		numRiskEngines:     s.numRiskEngines,
		numMatchingEngines: s.numMatchingEngines,
		prev:               s,
		journals:           treemap.NewWith(snapshotComparator),
	}
}

type Journal struct {
	timestampNS int64
	seqFirst    int64

	// -1 if not finished yet. // TODO make sure to init -1
	seqLast int64

	baseSnapshot *Snapshot
	prev         *Journal
	next         *Journal
	_            struct{}
}

type Processor struct {
	config *Config

	// TODO default -1
	enableJournalAfterSeq  int64
	journalBufFlushTrigger int32

	journalBuf *bytes.Buffer
	lz4Buf     *bytes.Buffer

	// TODO default value
	// TODO ConcurrentSkipListMap
	// snapshotIndex *treemap.Map

	lastSnapshot *Snapshot
	lastJournal  *Journal

	// TODO RandomAccessFile, FileChannel
	file *os.File

	fileCounter  int64
	writtenBytes int64
	mu           sync.RWMutex
	_            struct{}
}

func NewProcessor(
	config *Config,
	numRiskEngines int32,
	numMatchingEngines int32,
) *Processor {
	// TODO journalFileMaxSize
	var (
		journalBufFlushTrigger = config.journalBufSizeBytes - _maxCommandSizeBytes
		journalBuf             = bytes.NewBuffer(make([]byte, 0, config.journalBufSizeBytes))
		// TODO size
		lz4Buf = bytes.NewBuffer(make([]byte, 0, config.journalBufSizeBytes))
	)

	return &Processor{
		config:                 config,
		enableJournalAfterSeq:  -1,
		journalBufFlushTrigger: journalBufFlushTrigger,
		journalBuf:             journalBuf,
		lz4Buf:                 lz4Buf,
		// snapshotIndex: treemap.NewWith(),
		lastSnapshot: &Snapshot{
			numRiskEngines:     numRiskEngines,
			numMatchingEngines: numMatchingEngines,
		},
	}
}

func (p *Processor) Store(
	snapshotID int64,
	seq int64,
	timestampNS int64,
	category Category,
	instanceID int32,
	marshalable serialization.Marshalable,
) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO impl

	return true
}

func (p *Processor) Load(
	snapshotID int64,
	category Category,
	instanceID int32,
	f func(*bytes.Buffer) interface{},
) interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO impl

	return nil
}

/*
 * Returns error in case of writing issue
 * (will stop matching-engine from responding).
 */
// TODO thread safe?
func (p *Processor) WriteToJournal(
	command cmd.Command,
	seq int64, // distruptor sequence
	endOFBatch bool,
) error {
	if p.enableJournalAfterSeq == -1 || seq+p.config.baseSnapshotID <= p.enableJournalAfterSeq {
		return nil
	}

	timestamp := command.TimestampNS() // TODO vs place.timestamp
	code := command.Code()

	if code == cmd.ShutdownSignal_ {
		if err := p.flush(false, timestamp); err != nil {
			// TODO
		}

		return nil
	}

	// TODO
	// if (!cmdType.isMutate()) {
	// 	// skip queries
	// 	return;
	// }

	if p.file == nil {
		if err := p.newFile(timestamp); err != nil { // TODO vs place.timestamp
			// TODO
		}
	}

	if err := serialization.WriteInt8(command.Code(), p.journalBuf); err != nil {
		return err
	}

	command.SetSeq(seq + p.config.baseSnapshotSeq)

	if err := command.Marshal(p.journalBuf); err != nil {
		// TODO
	}

	if code == cmd.PersistStateRisk_ {
		// p.registerNextSnapshot() // TODO
		// basesnapshotID = TODO
		p.fileCounter = 0
		if err := p.flush(true, timestamp); err != nil {
			// TODO
		}
	} else if code == cmd.Reset_ {
		if err := p.flush(true, timestamp); err != nil {
			// TODO
		}
	} else if endOFBatch || p.journalBufFlushTrigger <= int32(p.journalBuf.Len()) {
		if err := p.flush(false, timestamp); err != nil {
			// TODO
		}
	}

	return nil
}

/*
 * Enable only after specified sequence,
 * For lower sequences no writes to journal.
 */
func (p *Processor) EnableJournalingAfter(seq int64) {
	p.enableJournalAfterSeq = seq
}

// func (p *processor) Snapshots() *treemap.Map {
// 	return p.snapshotIndex
// }

func (p *Processor) SnapshotExists(
	snapshotID int64,
	category Category,
	instanceID int32,
) bool {
	path := p.snapshotPath(snapshotID, category, instanceID)
	_, err := os.Stat(path)

	return err != nil
}

// TODO incompatible with exchange-core
// TODO types of uint8 vs byte (-128 to 127), action, order type, balance adj, ...
// TODO handle panic(s)
func (p *Processor) readCommands(
	buf *bytes.Buffer,
	lastSeq *int64,
	insideCompressedBlock bool,
) ([]cmd.Command, error) {
	var res []cmd.Command

	for code, err := serialization.ReadInt8(buf); err != nil; {
		emptyCommand, ok := cmd.From(code)

		if !ok {
			const msg = "Processor: readCommands: command: %v"

			return nil, fmt.Errorf(msg, code)
		}

		if emptyCommand.Code() == cmd.ReservedCompressed_ {
			if insideCompressedBlock {
				return nil, errRecursiveCompressionBlock
			}

			compressedSize, err := serialization.ReadInt32(buf)

			if err != nil {
				return nil, err
			}

			if compressedSize > _maxCompressedSizeBytes {
				const msg = "Processor: readCommands: bad compressed block size = %v (data corrupted)"

				return nil, fmt.Errorf(msg, compressedSize)
			}

			originalSize, err := serialization.ReadInt32(buf)

			if err != nil {
				return nil, err
			}

			if originalSize > _maxOriginalSizeBytes {
				const msg = "Processor: readCommands: bad original block size = %v (data corrupted)"

				return nil, fmt.Errorf(msg, originalSize)
			}

			originalData := make([]byte, originalSize)

			if _, err := lz4.UncompressBlock(
				buf.Bytes()[:compressedSize],
				originalData,
			); err != nil {
				return nil, err
			} else {
				buf.Next(int(compressedSize))
			}

			if partialResult, err := p.readCommands(
				bytes.NewBuffer(originalData),
				lastSeq,
				true, // insideCompressedBlock
			); err != nil {
				return nil, err
			} else {
				res = append(res, partialResult...)
			}
		} else {
			if err := emptyCommand.Unmarshal(buf); err != nil {
				// TODO
			}

			command := emptyCommand
			seq := command.Seq()

			if seq != *lastSeq+1 {
				log.Printf("warn: Sequence gap %v->%v (%v)", lastSeq, seq, seq-*lastSeq)
			}

			*lastSeq = seq

			res = append(res, command)
		}
	}

	return res, nil
}

func (p *Processor) flush(
	forceStartNextFile bool,
	timestampNS int64,
) error {
	length := int32(p.journalBuf.Len())

	if length < p.config.journalBatchCompressThresholdBytes {
		if _, err := p.file.Write(p.journalBuf.Bytes()); err != nil {
			// TODO reset journalBuf?
			return err
		}

		p.writtenBytes += int64(length)
	} else {
		prefixLen := 1 + 4 + 4
		p.lz4Buf.Reset()

		// indicates compressed block
		if err := serialization.WriteInt8(cmd.ReservedCompressed_, p.lz4Buf); err != nil {
			return err
		}

		// reserve 4 bytes space for compressed length
		if err := serialization.WriteInt32(0, p.lz4Buf); err != nil {
			return err
		}

		// uncompressed length
		if err := serialization.WriteInt32(length, p.lz4Buf); err != nil {
			return err
		}

		n, err := p.config.journalCompressor.CompressBlock(
			p.journalBuf.Bytes(),
			p.lz4Buf.Bytes()[prefixLen:],
		)

		if err != nil {
			// TODO reset journalBuf?
			return err
		}

		if n == 0 {
			// TODO reset journalBuf?
			return errIncompressible
		}

		view := bytes.NewBuffer(p.lz4Buf.Bytes()[1:1])

		if err := serialization.WriteInt32(int32(n), view); err != nil {
			return err
		}

		totalWritten := prefixLen + n
		if _, err = p.file.Write(p.lz4Buf.Bytes()[:totalWritten]); err != nil {
			// TODO reset journalBuf?
			return err
		}

		p.lz4Buf.Reset()
		p.writtenBytes += int64(totalWritten)
	}

	p.journalBuf.Reset()

	if forceStartNextFile || p.config.journalFileMaxSizeBytes <= p.writtenBytes {
		// TODO start preparing new file asynchronously, but ONLY ONCE
		if err := p.newFile(timestampNS); err != nil {
			// TODO
		}
		p.writtenBytes = 0
	}

	return nil
}

func (p *Processor) newFile(timestampNS int64) error {
	p.fileCounter++

	if p.file != nil {
		if err := p.file.Close(); err != nil {
			// TODO
		}
	}

	path := p.journalPath(p.fileCounter, p.config.baseSnapshotID)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if f, err := os.OpenFile(
			path,
			os.O_CREATE|os.O_RDWR,
			0644, // TODO equals to rwd?
		); err == nil {
			p.file = f
			p.registerNextJournal(p.config.baseSnapshotID, timestampNS) // TODO fix time

			return nil
		}
	}

	return fmt.Errorf("Processor: newFile: can't create new: %s", path)
}

// call only from journal thread
func (p *Processor) registerNextJournal(
	seq int64,
	timestampNS int64,
) {
	p.lastJournal = &Journal{
		timestampNS:  timestampNS,
		seqFirst:     seq,
		seqLast:      -1,
		baseSnapshot: p.lastSnapshot,
		prev:         p.lastJournal,
	}
}

// call only from journal thread
func (p *Processor) registerNextSnapshot(
	snapshotID int64,
	seq int64,
	timestampNS int64,
) {
	p.lastSnapshot = p.lastSnapshot.createNext(
		snapshotID,
		seq,
		timestampNS,
	)
}

func (p *Processor) mainLogPath() string {
	return fmt.Sprintf(
		"%s/%s.eca",
		p.config.storageFolder,
		p.config.exchangeID,
	)
}

func (p *Processor) snapshotPath(
	snapshotID int64,
	category Category,
	instanceID int32,
) string {
	return fmt.Sprintf(
		"%s/%s_snapshot_%d_%s%d.ecs",
		p.config.storageFolder,
		p.config.exchangeID,
		snapshotID,
		category,
		instanceID,
	)
}

func (p *Processor) journalPath(
	partitionId, snapshotID int64,
) string {
	return fmt.Sprintf(
		"%s/%s_journal_%d_%04X.ecj",
		p.config.storageFolder,
		p.config.exchangeID,
		snapshotID,
		partitionId,
	)
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

func CanLoadFromSnapshot(
	processor *Processor,
	config *Config,
	shardId int32,
	category Category,
) bool {
	if config.baseSnapshotID != 0 {
		if processor.SnapshotExists(
			config.baseSnapshotID,
			category,
			shardId,
		) {
			return true
		}

		// TODO panic?
	}

	return false
}
