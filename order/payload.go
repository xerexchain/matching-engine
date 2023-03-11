package order

// TODO EqualsAndHashCode
// TODO rename
type placePayload struct {
	orderId      int64 // TODO Is it redundant?
	userId       int64
	price        int64
	quantity     int64
	reservePrice int64
	symbolId     int32
	userCookie   int32
	action       Action
	t            Type
	timestamp    int64
	_            struct{}
}

// TODO EqualsAndHashCode
// TODO rename
type cancelPayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
// TODO rename
type movePayload struct {
	orderId   int64
	userId    int64
	symbolId  int32 // TODO Is it redundant?
	newPrice  int64
	timestamp int64
	_         struct{}
}

// TODO EqualsAndHashCode
// TODO rename
type reducePayload struct {
	orderId        int64
	userId         int64
	symbolId       int32 // TODO Is it redundant?
	reduceQuantity int64
	timestamp      int64
	_              struct{}
}
