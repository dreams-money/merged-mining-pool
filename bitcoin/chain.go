package bitcoin

const BitcoinMinConfirmations = 102

type Blockchain interface {
	ChainName() string
	CoinbaseDigest(coinbase string) (string, error)
	HeaderDigest(header string) (string, error)
	ShareMultiplier() float64
	MinimumConfirmations() uint

	ValidMainnetAddress(address string) bool
	ValidTestnetAddress(address string) bool
}

func GetChain(chainName string) Blockchain {
	switch chainName {
	case "dogecoin":
		return Dogecoin{}
	case "litecoin":
		return Litecoin{}
	default:
		panic("Unknown blockchain: " + chainName)
	}
}
