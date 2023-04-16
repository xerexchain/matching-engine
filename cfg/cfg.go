package cfg

import (
	"math"

	"github.com/pierrec/lz4/v4"
	"github.com/xerexchain/matching-engine/orderbook"
	"github.com/xerexchain/matching-engine/symbol"
)

// TODO change to factory because of thread safty
// TODO move to init?
var (
	lz4Fast = &fastCompressor{}
	lz4High = &highCompressor{}
)

// note: not thread safe
type Compressor interface {
	Compress(src []byte, dst []byte) (int, error)
}

type highCompressor struct {
	c *lz4.CompressorHC
}

type fastCompressor struct {
	c *lz4.Compressor
}

func (h *highCompressor) Compress(src []byte, dst []byte) (int, error) {
	return h.c.CompressBlock(src, dst)
}

func (f *fastCompressor) Compress(src []byte, dst []byte) (int, error) {
	return f.c.CompressBlock(src, dst)
}

// note: using LZ4 HIGH will require about twice more time
// note: using LZ4 HIGH is not recommended because of very high impact on throughput
type DiskProc struct {
	StorageFolder           string
	JournalBufSizeBytes     int32
	JournalFileMaxSizeBytes int64
	SnapshotCompressor      Compressor
	JournalCompressor       Compressor

	// use LZ4 compression if batch size exceeds this value for batches threshold
	// average batch size depends on traffic and disk write delay and can reach up to 20-100 kilobytes (3M TPS and 0.15ms disk write delay)
	// under moderate load for single messages compression is never used
	JournalBatchCompressThresholdBytes int32
	_                                  struct{}
}

func DefaultDiskProc() *DiskProc {
	return &DiskProc{
		StorageFolder:                      "./dumps",
		JournalBufSizeBytes:                256 * 1024,         // 256 KB - TODO calculate based on ringBufferSize
		JournalFileMaxSizeBytes:            4000 * 1024 * 1024, // 4 GB
		JournalBatchCompressThresholdBytes: 2048,               // 2048 B
		SnapshotCompressor:                 lz4Fast,
		JournalCompressor:                  lz4Fast,
	}
}

type InitialState struct {
	// TODO validate // TODO int or uuid
	// Should not have special characters because it is used for file names.
	ExchangeId      string
	BaseSnapshotId  int64
	BaseSnapshotSeq int64

	// When loading from journal, it will stop replaying commands as soon as this timestamp reached.
	// Set to 0 to ignore the journal, or int64 MAX_VALUE to read full available journal (or until reading error).
	JournalTimestampNs      int64
	PanicIfSnapshotNotFound bool
	_                       struct{}
}

func CleanStartInitialState(exchangeId string) *InitialState {
	return &InitialState{
		ExchangeId: exchangeId,
	}
}

func CleanStartJournalingInitialState(exchangeId string) *InitialState {
	return &InitialState{
		ExchangeId:              exchangeId,
		PanicIfSnapshotNotFound: true,
	}
}

// Configuration that loads from snapshot, without journal replay with journaling off.
func FromSnapshotOnlyInitialState(
	exchangeId string,
	snapshotId int64,
	snapshotBaseSeq int64,
) *InitialState {
	return &InitialState{
		ExchangeId:              exchangeId,
		BaseSnapshotId:          snapshotId,
		BaseSnapshotSeq:         snapshotBaseSeq,
		PanicIfSnapshotNotFound: true,
	}
}

// Configuration that load exchange from last known state including journal replay till last known start. Journal is enabled.
// TODO how to recreate from the next journal section recorded after the first recovery?
func LastKnownStateFromJournalInitialState(
	exchangeId string,
	snapshotId int64,
	snapshotBaseSeq int64,
) *InitialState {
	return &InitialState{
		ExchangeId:              exchangeId,
		BaseSnapshotId:          snapshotId,
		BaseSnapshotSeq:         snapshotBaseSeq,
		PanicIfSnapshotNotFound: true,
		JournalTimestampNs:      math.MaxInt64,
	}
}

// TODO toString overriden
// TODO add expected number of users and symbols
type Performance struct {
	// number of commands. Must be power of 2
	RingBufSize int32

	// Each instance requires extra CPU core.
	NumRiskEngines int32

	// Each instance requires extra CPU core.
	NumMatchingEngines int32

	/*
	 * max number of messages not processed by R2 stage. Must be less than quarter of ringBufferSize.
	 * Lower values, like 100, provide better mean latency.
	 * Higher values, like 2000 provide better throughput and tail latency.
	 */
	MSGsInGroupLimit int32

	/*
	 * max interval when messages not processed by R2 stage.
	 * Interfere with msgsInGroupLimit parameter.
	 * Lower values, like 1000 (1us), provide better mean latency.
	 * Higher values, like 2000 provide better throughput and tail latency.
	 */
	MaxGroupDurationNS int32

	/*
	 * send L2 for every successfully executed command
	 *
	 * Regular L2 updates is important for Risk Processor, to evaluate PnL for margin trading.
	 * By default (false), Matching Engine sends L2 only when requested by Grouping Processor (every 10ms).
	 * When true - L2 data will be sent for every successfully executed command.
	 * Enabling this will impact the performance.
	 *
	 */
	SendL2ForEveryCMD bool

	/*
	 * Depth of Regular L2 updates.
	 * Default is 8 (sufficient for Risk Processor because it does not check order book depth)
	 * If set Integer.MAX_VALUE - full order book will be sent.
	 */
	L2RefreshDepth int32

	// private final ThreadFactory threadFactory; // TODO
	// private final CoreWaitStrategy waitStrategy; // TODO

	OrderBookFactory func(symbol.Symbol) *orderbook.Naive

	CompressorFactory func() Compressor

	_ struct{}
}

func newHighCompressor() Compressor {
	return &highCompressor{}
}

func DefaultPerformance() *Performance {
	return &Performance{
		RingBufSize:        16 * 1024,
		NumRiskEngines:     1,
		NumMatchingEngines: 1,
		MSGsInGroupLimit:   256,
		MaxGroupDurationNS: 10000,
		SendL2ForEveryCMD:  false,
		L2RefreshDepth:     8,
		OrderBookFactory:   orderbook.NewNaive,
		CompressorFactory:  newHighCompressor,
	}
	// TODO
	// .threadFactory(Thread::new)
	// .waitStrategy(CoreWaitStrategy.BLOCKING)
}

func LatencyPerformance() *Performance {
	return &Performance{
		RingBufSize:        2 * 1024,
		NumRiskEngines:     1,
		NumMatchingEngines: 1,
		MSGsInGroupLimit:   256,
		MaxGroupDurationNS: 10000,
		SendL2ForEveryCMD:  false,
		L2RefreshDepth:     8,
		CompressorFactory:  newHighCompressor,
	}
	// TODO
	// .threadFactory(new AffinityThreadFactory(AffinityThreadFactory.ThreadAffinityMode.THREAD_AFFINITY_ENABLE_PER_LOGICAL_CORE))
	// .waitStrategy(CoreWaitStrategy.BUSY_SPIN)
	// .orderBookFactory(OrderBookDirectImpl::new);
}

func ThroughputPerformance() *Performance {
	return &Performance{
		RingBufSize:        64 * 1024,
		NumRiskEngines:     2,
		NumMatchingEngines: 4,
		MSGsInGroupLimit:   4096,
		MaxGroupDurationNS: 4000000,
		SendL2ForEveryCMD:  false,
		L2RefreshDepth:     8,
		CompressorFactory:  newHighCompressor,
	}
	// TODO
	// .threadFactory(new AffinityThreadFactory(AffinityThreadFactory.ThreadAffinityMode.THREAD_AFFINITY_ENABLE_PER_LOGICAL_CORE))
	// .waitStrategy(CoreWaitStrategy.BUSY_SPIN)
	// .orderBookFactory(OrderBookDirectImpl::new);
}
