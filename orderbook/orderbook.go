package orderbook

import (
	"log"

	"github.com/google/btree"
	"github.com/xerexchain/matching-engine/order"
	"github.com/xerexchain/matching-engine/orderbook/bucket"
	"github.com/xerexchain/matching-engine/orderbook/event"
	"github.com/xerexchain/matching-engine/serialization"
	"github.com/xerexchain/matching-engine/state"
	"github.com/xerexchain/matching-engine/symbol"
)

// FIX rece condition, concurrency

type OrderBook interface {
	state.Hashable
	serialization.Marshalable
	Symbol() symbol.Symbol
	PlaceGTCOrder(order.Order) MatcherResult
	PlaceIOCOrder(order.Order) MatcherResult
	PlaceFOKBudgetOrder(order.Order) MatcherResult
	// MoveOrder(order.Move)
	// ReduceOrder(order.Reduce)
	// CancelOrder(order.Cancel)
	// UserOrders(userId int64) []order.Order
	// AskOrders() []order.Order
	// BidOrders() []order.Order
	// FillAsks(size int32, data L2MarketData) // TODO quantity?
	// FillBids(size int32, data L2MarketData) // TODO quantity?
	// TotalAskBuckets(limit int32) // TODO int32?
	// TotalBidBuckets(limit int32) // TODO int32?
	// L2MarketDataSnapshot(size int64) L2MarketData  // TODO quantity?
	// PublishL2MarketDataSnapshot(data L2MarketData)
}

type NaiveOrderBook interface {
	OrderBook
}

type MatcherResult interface {
	EventHead() event.Event
	EventTail() event.Event
	SetEventHead(event.Event)
}

type naiveOrderBook struct {
	askBuckets *btree.BTree
	bidBuckets *btree.BTree
	symbol     symbol.Symbol
	orders     map[int64]order.Order
}

type matcherResult struct {
	head event.Event
	tail event.Event
	_    struct{}
}

func (m *matcherResult) EventHead() event.Event {
	return m.head
}

func (m *matcherResult) SetEventHead(e event.Event) {
	m.head = e
}

func (m *matcherResult) EventTail() event.Event {
	return m.tail
}

func (n *naiveOrderBook) prependRejectEvent(
	orderId int64,
	price int64,
	rejectedQuantity int64,
	action order.Action,
	res MatcherResult,
) {
	rejectEvent := event.NewRejectEvent(
		orderId,
		price,
		rejectedQuantity,
		action,
	)

	rejectEvent.SetNext(res.EventHead())
	res.SetEventHead(rejectEvent)
}

func (n *naiveOrderBook) targetBuckets(
	act order.Action,
) *btree.BTree {
	if act == order.Ask {
		return n.bidBuckets
	} else {
		return n.askBuckets
	}
}

func (n *naiveOrderBook) findBucket(
	price int64,
	buckets *btree.BTree,
) (bucket.NaiveOrderBucket, bool) {
	var res bucket.NaiveOrderBucket
	pivot := bucket.NewDumpNaiveOrderBucket(price)

	buckets.AscendGreaterOrEqual(
		pivot,
		func(item btree.Item) bool {
			buck := item.(bucket.NaiveOrderBucket)

			if price == buck.Price() {
				res = buck
			}

			return false
		},
	)

	ok := res != nil

	return res, ok
}

func (n *naiveOrderBook) budgetToFill(
	toCollect int64,
	act order.Action,
) (int64, int64) {
	collected := int64(0)
	budget := int64(0)

	f := func(item btree.Item) bool {
		if toCollect == collected {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		totalQuantity := buck.TotalQuantity()
		price := buck.Price()

		if totalQuantity <= toCollect {
			budget += totalQuantity * price
			collected += totalQuantity
		} else {
			budget += toCollect * price
			collected += toCollect
		}

		return true
	}

	if act == order.Ask {
		n.bidBuckets.Descend(f)
	} else {
		n.askBuckets.Ascend(f)
	}

	return budget, collected
}

func (n *naiveOrderBook) tryMatchInstantly(
	ord order.Order,
) MatcherResult {
	pivot := bucket.NewDumpNaiveOrderBucket(ord.Price())
	// collected := int64(0)
	emptyBucks := []bucket.NaiveOrderBucket{}
	var head, tail event.TradeEvent

	f := func(item btree.Item) bool {
		if ord.Remained() == 0 {
			return false
		}

		buck := item.(bucket.NaiveOrderBucket)
		res := buck.Match(
			ord.Remained(),
			ord.ReservedBidPrice(),
		)

		for _, orderId := range res.RemovedOrders {
			delete(n.orders, orderId)
		}

		if tail == nil {
			head = res.EventHead
		} else {
			tail.SetNext(res.EventHead)
		}

		tail = res.EventTail

		ord.Fill(res.CollectedQuantity)
		// collected += res.CollectedQuantity()

		if buck.TotalQuantity() == 0 {
			emptyBucks = append(emptyBucks, buck)
		}

		return true
	}

	if ord.Action() == order.Ask {
		n.bidBuckets.AscendGreaterOrEqual(pivot, f)
	} else {
		n.askBuckets.DescendLessOrEqual(pivot, f)
	}

	targetBucks := n.targetBuckets(ord.Action())

	// TODO Is it necessary?
	for _, buck := range emptyBucks {
		targetBucks.Delete(buck)
	}

	return &matcherResult{
		head: head,
		tail: tail,
	}
}

func (n *naiveOrderBook) Symbol() symbol.Symbol {
	return n.symbol
}

func (n *naiveOrderBook) PlaceGTCOrder(
	ord order.Order,
) MatcherResult {
	res := n.tryMatchInstantly(ord)

	if ord.Remained() == 0 {
		return res
	}

	if _, ok := n.orders[ord.Id()]; ok {
		log.Printf("warn: duplicate order id: %v\n", ord.Id())

		n.prependRejectEvent(
			ord.Id(),
			ord.Price(),
			ord.Remained(),
			ord.Action(),
			res,
		)

		return res
	}

	targetBucks := n.targetBuckets(ord.Action())
	buck, ok := n.findBucket(ord.Price(), targetBucks)

	if !ok {
		buck = bucket.NewNaiveOrderBucket(ord.Price())
		targetBucks.ReplaceOrInsert(buck)
	}

	buck.Put(ord)
	n.orders[ord.Id()] = ord

	return res
}

func (n *naiveOrderBook) PlaceIOCOrder(
	ord order.Order,
) MatcherResult {
	res := n.tryMatchInstantly(ord)

	if ord.Remained() == 0 {
		return res
	}

	n.prependRejectEvent(
		ord.Id(),
		ord.Price(),
		ord.Remained(),
		ord.Action(),
		res,
	)

	return res
}

func (n *naiveOrderBook) PlaceFOKBudgetOrder(
	ord order.Order,
) MatcherResult {
	budget, collected := n.budgetToFill(ord.Remained(), ord.Action())

	// TODO logic
	if collected == ord.Remained() || ((ord.Price() == budget) ||
		((ord.Action() == order.Ask) && (budget <= ord.Price())) ||
		((ord.Action() == order.Bid) && (budget > ord.Price()))) {
		return n.tryMatchInstantly(ord)
	} else {
		rejectEvent := event.NewRejectEvent(
			ord.Id(),
			ord.Price(),
			ord.Remained(),
			ord.Action(),
		)

		return &matcherResult{
			head: rejectEvent,
			tail: rejectEvent,
		}
	}
}

// TODO order.uid == cmd.uid
// func (n *naiveOrderBook) CancelOrder(orderId int64) {
// 	ord, ok := n.orders[orderId]

// 	if !ok {
// 		// TODO
// 		return
// 	}

// 	delete(n.orders, orderId)

// 	targetBucks := n.targetBuckets(ord.Action())
// 	buck, ok := n.findBucket(ord.Price(), targetBucks)

// 	if !ok {
// 		panic(
// 			fmt.Sprintf(
// 				"warn: can not find bucket for order price=%v"+
// 					" for order %v\n",
// 				ord.Price(),
// 				ord,
// 			),
// 		)
// 	}

// 	buck.Remove(orderId)

// 	// TODO Is it necessary?
// 	if buck.TotalQuantity() == 0 {
// 		targetBucks.Delete(buck)
// 	}
// }
