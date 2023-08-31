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

func (Dogecoin) ShareMultiplier() float32 {
	return 65536
}

func (Dogecoin) ValidMainnetAddress(address string) bool {
	// Apparently a base58 decode is the best way to validate.. TODO.
	// return regexp.MustCompile("^(D|9|A){1}[5-9A-HJ-NP-U]{1}[1-9A-HJ-NP-Za-km-z]{32}$").MatchString(address)
	return regexp.MustCompile("^[a-z0-9]{34}$").MatchString(address)
}

func (Dogecoin) ValidTestnetAddress(address string) bool {
	// return regexp.MustCompile("^(n|2){1}[a-z0-9]{33}$").MatchString(address)
	return regexp.MustCompile("[a-zA-Z0-9]{34}").MatchString(address)
}
