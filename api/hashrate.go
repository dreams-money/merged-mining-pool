package api

import (
	"fmt"
	"math"
)

type HashRate struct {
	Rate         string
	Denomination string
	Raw          float64
}

func floatToHashrate(hashrate float64) HashRate {
	var hashRateObject HashRate
	hashRateObject.Denomination = ""
	hashRateObject.Rate = "0"
	if hashrate < 0 {
		panic("Hashrate must be higher than 0")
	}
	if hashrate == 0 {
		return hashRateObject
	}

	thousand := 1000
	sizes := []string{"Hashes", "KH", "MH", "GH", "TH", "PH", "EH", "ZH", "YH"}
	thousanth := math.Floor(math.Log(float64(hashrate)) / math.Log(float64(thousand)))
	reducedAmount := float64(hashrate) / math.Pow(float64(thousand), thousanth)

	hashRateObject.Rate = fmt.Sprintf("%.2f", reducedAmount)
	hashRateObject.Denomination = sizes[int(thousanth)] + "/s"
	hashRateObject.Raw = hashrate

	return hashRateObject
}
