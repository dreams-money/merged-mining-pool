package bitcoin

import (
	"encoding/hex"
	"fmt"
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
	heightBytes = removeInsignificantBytes(heightBytes)
	heightHex := hex.EncodeToString(heightBytes)

	heightByteLen := uint(len(heightBytes))
	arbitraryByteLength = arbitraryByteLength + heightByteLen + 1 // 1 is for the heightByteLen byte

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
	return i.Version +
		i.NumberOfInputs +
		i.PreviousOutputTransactionID +
		i.PreviousOutputIndex +
		varUint(i.BytesInArbitrary) +
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
	return cb.CoinbaseInital + cb.Arbitrary + cb.CoinbaseFinal
}

func (t *Template) coinbaseTransactionOutputs(poolPubScriptKey string) (uint, string) {
	outputsCount := uint(0)
	outputs := ""

	if t.DefaultWitnessCommitment != "" {
		outputs = outputs + TransactionOut("00", t.DefaultWitnessCommitment) // LE?
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
