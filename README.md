# matching-engine
Implementation of an order matching engine


## Note
The project is still in development. The project is unstable and lacks proper testing. PLEASE DON'T USE IN PRODUCTION until tests are complete and first version is release.


## Desc
The project is inspired from [exchange-core](https://github.com/exchange-core/exchange-core). The goal is to port that project in golang then add new features like write-only-read-only instances. Integration with blockchain wallets is on the plan too.


##  TODO
- [exchange-core](https://github.com/exchange-core/exchange-core) TODO.
- side effect of java classes overriden `Equals`, `hashCode`, and `compareTo`. proper handling in Golang code base
- `orderbook`, `orderbucket`, and other packages have concurrency issues like race conditions.
- apply clean-code and naming conventions
- Add documentation for GTC, IOC, FOK, ReduceOrder
- Proper risk handling, balance handling, auth in upper levels
- Use different instanes for calling `UserOrders` of `orderbook` and other poor performance operations. add to doc
- Move `min(int64, int64)` to proper scope.
- Fix project logging.
- Recover from panics.
- debugging config of orderbook
