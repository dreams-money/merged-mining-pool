package blockchain

import (
	"regexp"

	"golang.org/x/crypto/scrypt"
)

type Litecoin struct{}

func (Litecoin) ChainName() string {
	return "dogecoin"
}

func (Litecoin) BlockDigest(header []byte) ([]byte, error) {
	return scrypt.Key(header, header, 1024, 1, 1, 32)
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
