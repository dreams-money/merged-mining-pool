package bitcoin

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/scrypt"
)

func DoubleSha256(input string) (string, error) {
	inputBytes, err := hex.DecodeString(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(inputBytes)
	sum = sha256.Sum256(sum[:])

	return hex.EncodeToString(sum[:]), nil
}

func ScryptDigest(input string) (string, error) {
	digest, err := hex.DecodeString(input)
	if err != nil {
		return "", err
	}
	digest, err = scrypt.Key(digest, digest, 1024, 1, 1, 32)
	return hex.EncodeToString(digest), nil
}
