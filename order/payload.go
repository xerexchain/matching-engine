package order

import "github.com/xerexchain/matching-engine/order/action"

// TODO EqualsAndHashCode
// TODO rename or delete
type placePayload struct {
	orderId      int64 // TODO Is it redundant?
	userId       int64
	price        int64
	quantity     int64
	reservePrice int64
	symbolId     int32
	userCookie   int32
	action       action.Action
	t            Type
	timestamp    int64
	_            struct{}
}

// TODO EqualsAndHashCode
// TODO rename or delete
type cancelPayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
// TODO rename or delete
type movePayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	newPrice  int64
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
// TODO rename or delete
type reducePayload struct {
	orderId        int64
	userId         int64
	symbolId       int32 // TODO Is it redundant?
	reduceQuantity int64
	timestamp      int64
	_              struct{}
}
