package block

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"time"
	"unsafe"
)

func doubleSha256(input []byte) [32]byte {
	sum := sha256.Sum256(input)
	sum = sha256.Sum256(sum[:])
	return sum
}

func reverseHex4Bytes(s string) (string, error) {
	if len(s)%8 != 0 {
		return "", errors.New("String must be divisible by 8 to represent 4 byte array")
	}

	var o string

	for l, i := len(s), 0; i < l/8; i++ {
		o = o + s[l-8*(i+1):(l-(8*i))]
	}

	return o, nil
}

func fourLittleEndianBytes(value interface{}) []byte {
	fourByteBuffer := make([]byte, 4)
	switch binaryValue := value.(type) {
	case int:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case int16:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case int32:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case uint:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case uint16:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case float32:
		binary.LittleEndian.PutUint32(fourByteBuffer, uint32(binaryValue))
	case uint32:
		binary.LittleEndian.PutUint32(fourByteBuffer, binaryValue)
	default:
		log.Fatalln("Unable to write 4 byte stream: " + value.(string))
	}

	return fourByteBuffer
}

func eightLittleEndianBytes(value interface{}) []byte {
	eightByteBuffer := make([]byte, 8)
	switch binaryValue := value.(type) {
	case int:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case int16:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case int32:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case int64:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case uint:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case uint16:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case uint32:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case uint64:
		binary.LittleEndian.PutUint64(eightByteBuffer, binaryValue)
	case float32:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	case float64:
		binary.LittleEndian.PutUint64(eightByteBuffer, uint64(binaryValue))
	default:
		log.Fatalln("Unable to write 8 byte stream: " + value.(string))
	}

	return eightByteBuffer
}

func varUint(value uint) ([]byte, error) {
	if value <= 252 {
		return []byte{byte(value)}, nil
	} else if value > 0xfd && value <= 0xffff {
		buffer := make([]byte, 2)
		binary.LittleEndian.PutUint16(buffer, uint16(value))
		return append([]byte{0xfd}, buffer...), nil
	} else if value > 0xffff && value <= 0xffffffff {
		buffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(buffer, uint32(value))
		return append([]byte{0xfe}, buffer...), nil
	} else if value > 0xffffffff && value <= 0xffffffffffffffff {
		buffer := make([]byte, 8)
		binary.LittleEndian.PutUint64(buffer, uint64(value))
		return append([]byte{0xff}, buffer...), nil
	} else {
		return nil, errors.New("Too large to stream")
	}
}

func removeInsignificantBytes(bytes []byte) []byte {
	var cleaned []byte
	for _, b := range bytes {
		if b != 0 {
			cleaned = append(cleaned, b)
		}
	}
	return cleaned
}

func significantBytesWithLengthHeader(bytes []byte) []byte {
	cleaned := removeInsignificantBytes(bytes)
	return bytesWithLengthHeader(cleaned)
}

func bytesWithLengthHeader(bytes []byte) []byte {
	lenHeader := []byte{byte(len(bytes))}
	return append(lenHeader, bytes...)
}

func reverse(b []byte) []byte {
	r := make([]byte, len(b))
	length := len(b)
	lengthMinusOne := (length - 1)
	for index := range b {
		r[lengthMinusOne-index] = b[index]
	}
	return r
}

func randString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
