package order

import "time"

type placePayload struct {
	orderId      int64 // TODO
	userId       int64
	size         int64
	price        int64
	reservePrice int64
	symbolId     int32
	userCookie   int32
	action       Action
	t            Type
	timestamp    time.Time
	_            struct{}
}

type cancelPayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	timestamp time.Time
	_         struct{}
}

type movePayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	newPrice  int64
	timestamp time.Time
	_         struct{}
}

type reducePayload struct {
	orderId    int64
	userId     int64
	symbolId   int32 // TODO Is it redundant?
	reduceSize int64
	timestamp  time.Time
	_          struct{}
}
