package market

const (
	L2Size = 32
)

// TODO check anywhere this struct is compared and exclude `timestamp` and `totalVolumeBid`
type L2MarketData interface {
	TotalOrderBookVolumeAsk() int64
	TotalOrderBookVolumeBid() int64
}

// TODO merge to Ask and Bid
type l2MarketData struct {
	askPrices  []int64
	askVolumes []int64
	askOrders  []int64
	bidPrices  []int64
	bidVolumes []int64
	bidOrders  []int64
}

func (l *l2MarketData) TotalOrderBookVolumeAsk() int64 {
	var t int64 = 0

	for _, v := range l.askVolumes {
		t += v
	}

	return t
}

func (l *l2MarketData) TotalOrderBookVolumeBid() int64 {
	var t int64 = 0

	for _, v := range l.bidVolumes {
		t += v
	}

	return t
}

func NewL2MarketData(askSize, bidSize int) L2MarketData {
	return &l2MarketData{
		askPrices:  make([]int64, askSize),
		askVolumes: make([]int64, askSize),
		askOrders:  make([]int64, askSize),
		bidPrices:  make([]int64, bidSize),
		bidVolumes: make([]int64, bidSize),
		bidOrders:  make([]int64, bidSize),
	}
}
