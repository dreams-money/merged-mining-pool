package pool

import "designs.capital/dogepool/bitcoin"

type Pair []bitcoin.BitcoinBlock

func (p Pair) GetPrimary() bitcoin.BitcoinBlock {
	return p[0]
}

func (p Pair) GetAux1() bitcoin.BitcoinBlock {
	return p[1]
}
