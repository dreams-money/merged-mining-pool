package payouts

import "errors"

func reverseHexBytes(hex string) (string, error) {
	if len(hex)%2 != 0 {
		return "", errors.New("String must be divisible by 2 to be a byte string")
	}
	o := ""
	l := len(hex)
	for i := l; i > 0; i = i - 2 {
		o = o + hex[i-2:i]
	}
	return o, nil
}
