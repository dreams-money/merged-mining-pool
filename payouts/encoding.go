package payouts

import (
	"errors"
	"math"
)

func reverseHexBytes(hex string) (string, error) {
	if len(hex)%2 != 0 {
		return "", errors.New("string must be divisible by 2 to be a byte string")
	}
	o := ""
	l := len(hex)
	for i := l; i > 0; i = i - 2 {
		o = o + hex[i-2:i]
	}
	return o, nil
}

func roundToThreeDigits(x float32) float32 {
	return float32(math.Round(float64(x)*1000) / 1000)
}
