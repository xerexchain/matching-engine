package resultcode

type ResultCode int

const (
	New                    ResultCode = 0
	ValidForMatchingEngine ResultCode = 1

	Success  ResultCode = 100
	Accepted ResultCode = 110

	AuthInvalidUser  ResultCode = -1001
	AuthTokenExpired ResultCode = -1002

	InvalidSymbol         ResultCode = -1201
	InvalidPriceStep      ResultCode = -1202
	UnsupportedSymbolType ResultCode = -1203

	RiskNFS                    ResultCode = -2001
	RiskInvalidReservedBidPrice ResultCode = -2002
	RiskAskPriceLowerThanFee   ResultCode = -2003
	RiskMarginTradingDisabled  ResultCode = -2004

	MatchingUnknownOrderId         ResultCode = -3002
	MatchingDuplicateOrderId       ResultCode = -3003
	MatchingUnsupportedCommand     ResultCode = -3004
	MatchingInvalidOrderBookId     ResultCode = -3005
	MatchingOrderBookAlreadyExists ResultCode = -3006
	MatchingUnsupportedOrderType   ResultCode = -3007

	MatchingMoveRejectedDifferentPrice   ResultCode = -3040
	MatchingMoveFailedPriceOverRiskLimit ResultCode = -3041
	MatchingMoveFailedPriceInvalid ResultCode = -3042 // TODO

	MatchingReduceFailedWrongSize ResultCode = -3051

	UserMGMTUserAlreadyExists ResultCode = -4001

	UserMGMTAccountBalanceAdjustmentZero               ResultCode = -4100
	UserMGMTAccountBalanceAdjustmentAlreadyAppliedSame ResultCode = -4101
	UserMGMTAccountBalanceAdjustmentAlreadyAppliedMany ResultCode = -4102
	UserMGMTAccountBalanceAdjustmentNSF                ResultCode = -4103
	UserMGMTNonZeroAccountBalance                      ResultCode = -4104

	UserMGMTUserNotSuspendableHasPositions     ResultCode = -4130
	UserMGMTUserNotSuspendableNonEmptyAccounts ResultCode = -4131
	UserMGMTUserNotSuspended                   ResultCode = -4132
	UserMGMTUserAlreadySuspended               ResultCode = -4133

	UserMGMTUserNotFound ResultCode = -4201

	SymbolMGMTSymbolAlreadyExists ResultCode = -5001

	BinaryCommandFailed              ResultCode = -8001
	ReportQueryUnknownType           ResultCode = -8003
	StatePersistRiskEngineFailed     ResultCode = -8010
	StatePersistMatchingEngineFailed ResultCode = -8020

	DROP ResultCode = -9999

	// codes below -10000 are reserved for gateways
)

func MergeToFirstFailed(codes ...ResultCode) ResultCode {
	for _, c := range codes {
		if c != Success && c != Accepted {
			return c
		}
	}

	for _, c := range codes {
		if c == Success {
			return Success
		}
	}

	return Accepted
}
