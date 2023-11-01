Why Bitcoin?
============

Because both Dogecoin and Litecoin are forks of Bitcoin.  As such, their block generation and RPC methods are the same.

It can be said that both Dogecoin and Litecoin are in the Bitcoin family of coins.

type Generator interface{} has a good amount of the methods the pool relies on.
type Blockchain interface{} has the variances for each specific coin like Dogecoin and Litecoin, which is consumed by the generator

Note: though I've coded out most of Bitcoin's block generation, a specific Blockchain interface{} has not been coded out or tested for Bitcoin itself.
