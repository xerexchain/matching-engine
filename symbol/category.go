package symbol

type Category int8

const (
	_currencyExchangePair Category = iota + 1
	_futureContract
	_option
)

var (
	_categories = map[int8]Category{
		int8(_currencyExchangePair): _currencyExchangePair,
		int8(_futureContract):       _futureContract,
		int8(_option):               _option,
	}

	_factory = map[int8]func() _Symbol{
		int8(_currencyExchangePair): func() _Symbol {
			return &Symbol{}
		},
		int8(_futureContract): func() _Symbol {
			return &FutureContract{}
		},
		int8(_option): func() _Symbol {
			// TODO panic?
			panic("not implemented")
		},
	}
)

func categoryFrom(code int8) (Category, bool) {
	category, ok := _categories[code]

	return category, ok
}
