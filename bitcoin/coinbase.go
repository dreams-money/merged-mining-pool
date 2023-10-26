package bitcoin

import (
	"encoding/hex"
	"fmt"
	"log"
)

// https://developer.bitcoin.org/reference/transactions.html#coinbase-input-the-input-of-the-first-transaction-in-a-block

type CoinbaseInital struct {
	Version                     string
	NumberOfInputs              string
	PreviousOutputTransactionID string
	PreviousOutputIndex         string
	BytesInArbitrary            uint
	BytesInHeight               uint
	HeightHex                   string
}

func (t *Template) CoinbaseInitial(arbitraryByteLength uint) CoinbaseInital {
	heightBytes := eightLittleEndianBytes(t.Height)
	heightBytes = removeInsignificantBytesLittleEndian(heightBytes)
	heightHex := hex.EncodeToString(heightBytes)

	heightByteLen := uint(len(heightBytes))
	arbitraryByteLength = arbitraryByteLength + heightByteLen + 1 // 1 is for the heightByteLen byte

	if arbitraryByteLength > 100 {
		log.Printf("!!WARNING!! - Coinbase length too long - !!WARNING!! %v\n", arbitraryByteLength)
	}

	return CoinbaseInital{
		Version:                     "01000000", // Different from template version
		NumberOfInputs:              "01",
		PreviousOutputTransactionID: "0000000000000000000000000000000000000000000000000000000000000000",
		PreviousOutputIndex:         "ffffffff",
		BytesInArbitrary:            arbitraryByteLength,
		BytesInHeight:               heightByteLen,
		HeightHex:                   heightHex,
	}
}

func (i CoinbaseInital) Serialize() string {
	// debugCoinbaseInitialOutput(i)
	return i.Version +
		i.NumberOfInputs +
		i.PreviousOutputTransactionID +
		i.PreviousOutputIndex +
		varUint(i.BytesInArbitrary) +
		// These next two aren't arbitrary, but they are in the arbitrary section ;)
		varUint(i.BytesInHeight) +
		i.HeightHex
}

type CoinbaseFinal struct {
	TransactionInSequence string
	OutputCount           uint
	TxOuts                string
	TransactionLockTime   string
}

func (t *Template) CoinbaseFinal(poolPayoutPubScriptKey string) CoinbaseFinal {
	txOutputLen, txOutput := t.coinbaseTransactionOutputs(poolPayoutPubScriptKey)
	return CoinbaseFinal{
		TransactionInSequence: "00000000",
		OutputCount:           txOutputLen,
		TxOuts:                txOutput,
		TransactionLockTime:   "00000000",
	}
}

func (f CoinbaseFinal) Serialize() string {
	// debugCoinbaseFinalOutput(f)
	return f.TransactionInSequence +
		varUint(f.OutputCount) +
		f.TxOuts +
		f.TransactionLockTime
}

type Coinbase struct {
	CoinbaseInital string
	Arbitrary      string
	CoinbaseFinal  string
}

func (cb *Coinbase) Serialize() string {
	// debugCoinbaseOutput(cb)
	return cb.CoinbaseInital + cb.Arbitrary + cb.CoinbaseFinal
}

func (t *Template) coinbaseTransactionOutputs(poolPubScriptKey string) (uint, string) {
	outputsCount := uint(0)
	outputs := ""

	if t.DefaultWitnessCommitment != "" {
		outAmount := "0000000000000000"
		outputs = outputs + TransactionOut(outAmount, t.DefaultWitnessCommitment)
		outputsCount++
	}

	// Some alt coins may have additional outputs..

	// Pool reward output
	rewardAmount := fmt.Sprintf("%016x", t.CoinBaseValue)
	rewardAmount, _ = reverseHexBytes(rewardAmount)
	outputsCount++

	outputs = outputs + TransactionOut(rewardAmount, poolPubScriptKey)

	return outputsCount, outputs
}

func debugCoinbaseOutput(cb *Coinbase) {
	fmt.Println()
	fmt.Println("**Coinbase Parts**")
	fmt.Println()
	fmt.Println("Initial", cb.CoinbaseInital)
	fmt.Println("Arbitrary", cb.Arbitrary)
	fmt.Println("Final", cb.CoinbaseFinal)
	fmt.Println()
	fmt.Println("Coinbase", cb.CoinbaseInital+cb.Arbitrary+cb.CoinbaseFinal)
	fmt.Println()
}

func debugCoinbaseInitialOutput(i CoinbaseInital) {
	fmt.Println()
	fmt.Println("🧐 Coinbase Initial Parts ➔ ➔ ➔ ➔")
	fmt.Println()
	fmt.Println("Version", i.Version)
	fmt.Println("NumberOfInputs", i.NumberOfInputs)
	fmt.Println("PreviousOutputTransactionID", i.PreviousOutputTransactionID)
	fmt.Println("PreviousOutputIndex", i.PreviousOutputIndex)
	fmt.Println("BytesInArbitrary", i.BytesInArbitrary)
	fmt.Println("BytesInHeight", i.BytesInHeight)
	fmt.Println("HeightHex", i.HeightHex)
	fmt.Println()
	cbI := i.Version +
		i.NumberOfInputs +
		i.PreviousOutputTransactionID +
		i.PreviousOutputIndex +
		varUint(i.BytesInArbitrary) +
		varUint(i.BytesInHeight) +
		i.HeightHex
	fmt.Println("Coinbase Initial", cbI)
	fmt.Println()
}

func debugCoinbaseFinalOutput(f CoinbaseFinal) {
	fmt.Println()
	fmt.Println("➔ ➔ ➔ ➔ Coinbase Final Parts**")
	fmt.Println()
	fmt.Println("TransactionInSequence", f.TransactionInSequence)
	fmt.Println("OutputCount", f.OutputCount)
	fmt.Println("TxOuts", f.TxOuts)
	fmt.Println("TransactionLockTime", f.TransactionLockTime)
	fmt.Println()
	cbf := f.TransactionInSequence +
		varUint(f.OutputCount) +
		f.TxOuts +
		f.TransactionLockTime
	fmt.Println("Coinbase Final", cbf)
	fmt.Println()
}
