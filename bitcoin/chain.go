package bitcoin

type Blockchain interface {
	ChainName() string
	CoinbaseDigest(coinbase string) (string, error)
	HeaderDigest(header string) (string, error)
	ShareMultiplier() float32

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
