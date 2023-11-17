package bitcoin

import (
	"regexp"
)

type Litecoin struct{}

func (Litecoin) ChainName() string {
	return "litecoin"
}

func (Litecoin) CoinbaseDigest(coinbase string) (string, error) {
	return DoubleSha256(coinbase)
}

func (Litecoin) HeaderDigest(header string) (string, error) {
	return ScryptDigest(header)
}

func (Litecoin) ShareMultiplier() float64 {
	return 65536
}

func (Litecoin) ValidMainnetAddress(address string) bool {
	return regexp.MustCompile("^(L|M)[A-Za-z0-9]{33}$|^(ltc1)[0-9A-Za-z]{39}$").MatchString(address)
}

func (Litecoin) ValidTestnetAddress(address string) bool {
	return regexp.MustCompile("[a-z0-9]{44}").MatchString(address)
}

func (Litecoin) MinimumConfirmations() uint {
	return uint(BitcoinMinConfirmations)
}
