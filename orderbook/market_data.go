package orderbook

const (
	L2Size = 32
)

// TODO equals overriden
// TODO check anywhere this struct is compared and exclude `timestamp` and `referenceSeq`
type L2MarketData struct {
	askPrices     []int64
	askQuantites  []int64
	numAskOrders  []int32
	bidPrices     []int64
	bidQuantities []int64
	numBidOrders  []int32
	timestamp     int64
	referenceSeq  int64
	_             struct{}
}

func (l *L2MarketData) AskSize() int32 {
	return int32(len(l.askPrices))
}

func (l *L2MarketData) BidSize() int32 {
	return int32(len(l.bidPrices))
}

func (l *L2MarketData) LimitAskViewTo(size int32) {
	l.askPrices = l.askPrices[:size]
	l.askQuantites = l.askQuantites[:size]
	l.numAskOrders = l.numAskOrders[:size]
}

func (l *L2MarketData) LimitBidViewTo(size int32) {
	l.bidPrices = l.bidPrices[:size]
	l.bidQuantities = l.bidQuantities[:size]
	l.numBidOrders = l.numBidOrders[:size]
}

func (l *L2MarketData) TotalAskQuantity() int64 {
	var t int64 = 0

	for _, v := range l.askQuantites {
		t += v
	}

	return t
}

func (l *L2MarketData) TotalBidQuantity() int64 {
	var t int64 = 0

	for _, v := range l.bidQuantities {
		t += v
	}

	return t
}

func (l *L2MarketData) SetAskPriceAt(index int32, price int64) {
	l.askPrices[index] = price
}

func (l *L2MarketData) SetAskQuantityAt(index int32, quantity int64) {
	l.askQuantites[index] = quantity
}

func (l *L2MarketData) SetNumAskOrdersAt(index int32, num int32) {
	l.numAskOrders[index] = num
}

func (l *L2MarketData) SetBidPriceAt(index int32, price int64) {
	l.bidPrices[index] = price
}

func (l *L2MarketData) SetBidQuantityAt(index int32, quantity int64) {
	l.bidQuantities[index] = quantity
}
func (l *L2MarketData) SetNumBidOrdersAt(index int32, num int32) {
	l.numBidOrders[index] = num
}

func NewL2MarketData(askSize, bidSize int32) *L2MarketData {
	return &L2MarketData{
		askPrices:     make([]int64, askSize),
		askQuantites:  make([]int64, askSize),
		numAskOrders:  make([]int32, askSize),
		bidPrices:     make([]int64, bidSize),
		bidQuantities: make([]int64, bidSize),
		numBidOrders:  make([]int32, bidSize),
	}
}
