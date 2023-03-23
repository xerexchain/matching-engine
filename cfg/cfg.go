package cfg

import "github.com/pierrec/lz4/v4"

// TODO move to init?
var (
	lz4Fast = &fastCompressor{}
	lz4High = &highCompressor{}
)

type InitialState interface {
	IsEmptySnapshot() bool // rename?
	SnapshotId() int64
	PanicIfSnapshotNotFound() bool
	JournalTimestampNs() int64
}

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
	StorageFolder      string
	JournalBuffSize    int32
	JournalFileMaxSize int64
	SnapshotCompressor Compressor
	JournalCompressor  Compressor

	// use LZ4 compression if batch size (in bytes) exceeds this value for batches threshold
	// average batch size depends on traffic and disk write delay and can reach up to 20-100 kilobytes (3M TPS and 0.15ms disk write delay)
	// under moderate load for single messages compression is never used
	JournalBatchCompressThreshold int32
	_                             struct{}
}

func DefaultDiskProc() *DiskProc {
	return &DiskProc{
		StorageFolder:                 "./dumps",
		JournalBuffSize:               256 * 1024,         // 256 KB - TODO calculate based on ringBufferSize
		JournalFileMaxSize:            4000 * 1024 * 1024, // 4 GB
		JournalBatchCompressThreshold: 2048,               // 2048 B
		SnapshotCompressor:            lz4Fast,
		JournalCompressor:             lz4Fast,
	}
}
