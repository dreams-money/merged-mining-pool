package pool

import "encoding/hex"

func hexStringToByteString(hexStr string) string {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func reverseHexBytes(hex string) string {
	if len(hex)%2 != 0 {
		panic("String must be divisible by 2 to be a byte string")
	}
	o := ""
	l := len(hex)
	for i := l; i > 0; i = i - 2 {
		o = o + hex[i-2:i]
	}
	return o
}
