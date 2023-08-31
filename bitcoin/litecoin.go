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

func (Litecoin) ShareMultiplier() float32 {
	return 65536
}

func (Litecoin) ValidMainnetAddress(address string) bool {
	return regexp.MustCompile("^[a-z0-9]{44}$").MatchString(address)
}

func (d Litecoin) ValidTestnetAddress(address string) bool {
	return regexp.MustCompile("[a-z0-9]{44}").MatchString(address)
}
