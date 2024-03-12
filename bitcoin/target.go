package bitcoin

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

// https://developer.bitcoin.org/reference/block_chain.html#target-nbits

type Target string

const highestTarget = "00000000ffff0000000000000000000000000000000000000000000000000000"

func (t *Target) ToBig() (*big.Int, bool) {
	return new(big.Int).SetString(string(*t), 16)
}

func (t *Target) ToDifficulty() (float64, big.Accuracy) {
	highestTargetBig, success := new(big.Int).SetString(highestTarget, 16)
	if !success {
		panic("Failed to convert highest target value to big int")
	}
	highestTargetBigFloat := new(big.Float).SetInt(highestTargetBig)

	targetBig, success := t.ToBig()
	if !success {
		panic("Failed to convert target to big int")
	}
	targetBigFloat := new(big.Float).SetInt(targetBig)
	difficulty := new(big.Float).Quo(highestTargetBigFloat, targetBigFloat)

	return difficulty.Float64()
}

func TargetFromDifficulty(difficulty float64) (Target, big.Accuracy) {
	highestTargetBig, success := new(big.Int).SetString(highestTarget, 16)
	if !success {
		panic("Failed to convert highest target value to big int")
	}
	highestTargetBigFloat := new(big.Float).SetInt(highestTargetBig)

	difficultyBigFloat := new(big.Float).SetFloat64(difficulty)

	targetBigFloat := new(big.Float).Quo(highestTargetBigFloat, difficultyBigFloat)

	targetBigInt, bigAccuracy := targetBigFloat.Int(nil)
	targetHex := targetBigInt.Text(16)

	return Target(targetHex), bigAccuracy
}

func TargetFromBits(bitsHex string) (Target, error) {
	l := len(bitsHex)
	if l < 8 {
		return "", errors.New("string too short")
	}
	if l%2 == 0 {
		return "", errors.New("string must be even length")
	}

	exponent := bitsHex[:2]
	exponentValue, err := strconv.ParseInt(exponent, 16, 8)
	if err != nil {
		return "", err
	}
	significand := bitsHex[3:]
	significandValue, err := strconv.ParseInt(significand, 16, 8)
	if err != nil {
		return "", err
	}

	target := significandValue*256 ^ (exponentValue - 3)
	targetHex := fmt.Sprintf("%x", target)

	return Target(targetHex), nil
}
