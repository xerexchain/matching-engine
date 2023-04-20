package cfg

import (
	"github.com/pierrec/lz4/v4"
	"github.com/xerexchain/matching-engine/orderbook"
	"github.com/xerexchain/matching-engine/symbol"
)

type Compressor interface {
	// note: not thread safe
	CompressBlock(src []byte, dst []byte) (int, error)
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

func highCompFactory() Compressor {
	return &lz4.CompressorHC{}
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
		// OrderBookFactory:   orderbook.NewNaive,
		CompressorFactory: highCompFactory,
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
		CompressorFactory:  highCompFactory,
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
		CompressorFactory:  highCompFactory,
	}
	// TODO
	// .threadFactory(new AffinityThreadFactory(AffinityThreadFactory.ThreadAffinityMode.THREAD_AFFINITY_ENABLE_PER_LOGICAL_CORE))
	// .waitStrategy(CoreWaitStrategy.BUSY_SPIN)
	// .orderBookFactory(OrderBookDirectImpl::new);
}
