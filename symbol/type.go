package symbol

type Type int8

const (
	CurrencyExchangePair Type = iota
	FutureContract
	Option
)
