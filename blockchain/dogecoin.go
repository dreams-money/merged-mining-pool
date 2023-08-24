package blockchain

import (
	"regexp"

	"golang.org/x/crypto/scrypt"
)

type Dogecoin struct{}

func (Dogecoin) ChainName() string {
	return "dogecoin"
}

func (Dogecoin) BlockDigest(header []byte) ([]byte, error) {
	return scrypt.Key(header, header, 1024, 1, 1, 32)
}

func (Dogecoin) ShareMultiplier() float32 {
	return 65536
}

func (Dogecoin) ValidMainnetAddress(address string) bool {
	// Apparently a base58 decode is the best way to validate.. todo.
	// return regexp.MustCompile("^(D|9|A){1}[5-9A-HJ-NP-U]{1}[1-9A-HJ-NP-Za-km-z]{32}$").MatchString(address)
	return regexp.MustCompile("^[a-z0-9]{34}$").MatchString(address)
}

func (d Dogecoin) ValidTestnetAddress(address string) bool {
	// return regexp.MustCompile("^(n|2){1}[a-z0-9]{33}$").MatchString(address)
	return regexp.MustCompile("[a-zA-Z0-9]{34}").MatchString(address)
}
