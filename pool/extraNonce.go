package pool

import (
	"math/rand"
	"time"
)

var extranonces map[string]bool

func uniqueExtranonce(length int) string {

	if extranonces == nil {
		extranonces = make(map[string]bool)
	}

	extranonce := randomHex(length)
	for {
		_, alreadyUsed := extranonces[extranonce]
		if alreadyUsed {
			extranonce = randomHex(length)
		} else {
			break
		}
	}
	extranonces[extranonce] = true
	return extranonce
}

func randomHex(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "0123456789abcdef"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
