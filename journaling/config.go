package journaling

import "github.com/pierrec/lz4/v4"

type Compressor interface {
	// note: not thread safe
	CompressBlock(src []byte, dst []byte) (int, error)
}

/*
 * note: Using LZ4 HIGH will require about twice more time.
 * note: Using LZ4 HIGH is not recommended because of
 * very high impact on throughput.
 */
type Config struct {
	// TODO validate // TODO string, int32 or uuid?
	// Should not have special characters because it is used for file names.
	exchangeID      string
	baseSnapshotID  int64
	baseSnapshotSeq int64

	/*
	 * When loading from journal, it will stop replaying commands
	 * as soon as this timestamp reached.
	 * Set to 0 to ignore the journal, or int64 MAX_VALUE to
	 * read full available journal (or until reading error).
	 */
	journalTimestampNS int64

	storageFolder           string
	journalBufSizeBytes     int32
	journalFileMaxSizeBytes int64
	journalCompressor       Compressor
	snapshotCompressor      Compressor

	/*
	 * Use LZ4 compression if batch size exceeds this value for batches threshold.
	 * Average batch size depends on traffic and disk write delay and can reach up to
	 * 20-100 kilobytes (3M TPS and 0.15ms disk write delay).
	 * Under moderate load for single messages compression is never used.
	 */
	journalBatchCompressThresholdBytes int32

	_ struct{}
}

func defaultConfig() *Config {
	return &Config{
		exchangeID:         "myexchange",
		storageFolder:      "./dumps",
		journalCompressor:  &lz4.Compressor{},
		snapshotCompressor: &lz4.Compressor{},

		// 256 KB - TODO calculate based on ringBufferSize
		journalBufSizeBytes: 256 * 1024,

		// 4 GB
		journalFileMaxSizeBytes: 4000 * 1024 * 1024,

		// 2048 B
		journalBatchCompressThresholdBytes: 2048,
	}
}
