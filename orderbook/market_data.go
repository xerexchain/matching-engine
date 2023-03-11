package orderbook

const (
	L2Size = 32
)

// TODO equals overriden
// TODO check anywhere this struct is compared and exclude `timestamp` and `referenceSeq`
type L2MarketData interface {
	TotalOrderBookQuantityAsk() int64
	TotalOrderBookQuantityBid() int64
}

// TODO equals overriden
// TODO check anywhere this struct is compared and exclude `timestamp` and `referenceSeq`
type l2MarketData struct {
	askPrices     []int64
	askQuantites  []int64
	askOrders     []int64
	bidPrices     []int64
	bidQuantities []int64
	bidOrders     []int64
	timestamp     int64
	referenceSeq  int64
	_             struct{}
}

func (l *l2MarketData) TotalOrderBookQuantityAsk() int64 {
	var t int64 = 0

	for _, v := range l.askQuantites {
		t += v
	}

	return t
}

func (l *l2MarketData) TotalOrderBookQuantityBid() int64 {
	var t int64 = 0

	for _, v := range l.bidQuantities {
		t += v
	}

	return t
}

func NewL2MarketData(askSize, bidSize int) L2MarketData {
	return &l2MarketData{
		askPrices:     make([]int64, askSize),
		askQuantites:  make([]int64, askSize),
		askOrders:     make([]int64, askSize),
		bidPrices:     make([]int64, bidSize),
		bidQuantities: make([]int64, bidSize),
		bidOrders:     make([]int64, bidSize),
	}
}
