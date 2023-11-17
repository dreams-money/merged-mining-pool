package bitcoin

import (
	"regexp"
)

type Dogecoin struct{}

func (Dogecoin) ChainName() string {
	return "dogecoin"
}

func (Dogecoin) CoinbaseDigest(coinbase string) (string, error) {
	return DoubleSha256(coinbase)
}

func (Dogecoin) HeaderDigest(header string) (string, error) {
	return ScryptDigest(header)
}

func (Dogecoin) ShareMultiplier() float64 {
	return 65536
}

func (Dogecoin) ValidMainnetAddress(address string) bool {
	// Apparently a base58 decode is the best way to validate.. TODO.
	return regexp.MustCompile("^(D|A|9)[a-km-zA-HJ-NP-Z1-9]{33,34}$").MatchString(address)
}

func (Dogecoin) ValidTestnetAddress(address string) bool {
	return regexp.MustCompile("^(n|2)[a-km-zA-HJ-NP-Z1-9]{33}$").MatchString(address)
}

func (Dogecoin) MinimumConfirmations() uint {
	return uint(251)
}
