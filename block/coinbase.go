package block

import (
	"encoding/hex"
	"fmt"
	"time"

	"designs.capital/dogepool/rpc"
	"designs.capital/dogepool/template"
)

/* S/O - https://developer.bitcoin.org/reference/transactions.html#coinbase-input-the-input-of-the-first-transaction-in-a-block

(start coinbase initial)	01000000 .............................. Version

							01 .................................... Number of inputs
							| 00000000000000000000000000000000
							| 00000000000000000000000000000000 ...  Previous outpoint TXID
							| ffffffff ............................ Previous outpoint index
							|
							| 29 .................................. Bytes in coinbase
							| |
(start signature initial)   | | 03 ................................ Bytes in height
(end signature initial 		| | | 4e0105 .......................... Height: 328014
 && end coinbase initial)   | |
(start coinbase final)   	| | 062f503253482f0472d35454085fffed
(includes signature)    	| | f2400000f90f54696d65202620486561
							| | 6c74682021 ........................ Arbitrary data
							| 00000000 ............................ Sequence

							01 .................................... Output count
							| 2c37449500000000 .................... Satoshis (25.04275756 BTC)
							| 1976a914a09be8040cbf399926aeb1f4
							| 70c37d1341f3b46588ac ................ P2PKH script
							| 00000000 ............................ Locktime

**/

func makeCoinbaseParts(template rpc.BlockTemplate, userArbitrary []byte, poolPubScriptKey string, extranonceLength int) ([]byte, []byte, error) {
	// Produce coinbase final first because we need it's length for coinbaseInitial
	coinbaseFinal, err := coinbaseFinal(userArbitrary, template, poolPubScriptKey)
	if err != nil {
		return nil, nil, err
	}

	arbitraryInitial := getApplicationArbitrary()
	signatureInitial := signatureInitial(template, arbitraryInitial)

	arbitraryLength := len(signatureInitial) + len(userArbitrary) + extranonceLength

	coinbaseInitial, err := coinbaseInitial(signatureInitial, arbitraryLength)
	if err != nil {
		return nil, nil, err
	}

	return coinbaseInitial, coinbaseFinal, nil
}

func coinbaseFinal(arbitrary []byte, template rpc.BlockTemplate, poolPubScriptKey string) ([]byte, error) {
	transactionInSequence := fourLittleEndianBytes(0)
	txOuts, err := coinbaseTransactionOutputsForCoin(template, poolPubScriptKey)
	if err != nil {
		return nil, err
	}
	transactionLockTime := fourLittleEndianBytes(0)

	coinbaseFinal := arbitrary
	coinbaseFinal = append(coinbaseFinal, transactionInSequence...)
	coinbaseFinal = append(coinbaseFinal, txOuts...)
	coinbaseFinal = append(coinbaseFinal, transactionLockTime...)

	return coinbaseFinal, nil
}

func coinbaseInitial(signatureInitial []byte, arbitraryLength int) ([]byte, error) {
	arbitraryLen, err := varUint(uint(arbitraryLength))
	if err != nil {
		return nil, err
	}

	coinbaseInitial := transactionVersion()
	coinbaseInitial = append(coinbaseInitial, coinbaseInputs()...)
	coinbaseInitial = append(coinbaseInitial, arbitraryLen...)
	coinbaseInitial = append(coinbaseInitial, signatureInitial...)

	return coinbaseInitial, nil
}

// After work has been processed, we can assemble the real coinbase.
func finalizeCoinbase(coinbaseInitial, arbitrary, coinbaseFinal []byte, t *template.Block) []byte {
	coinbase := coinbaseInitial
	coinbase = append(coinbase, arbitrary...)
	coinbase = append(coinbase, coinbaseFinal...)

	t.RequestedWork[coinbaseCacheID] = coinbase

	coinbaseHash := doubleSha256(coinbase)

	return coinbaseHash[:]
}

func transactionVersion() []byte {
	return fourLittleEndianBytes(1)
}

func coinbaseInputs() []byte {
	// Proof of Stake conditionals when we get there right here
	transactionInputCount := []byte{1} // 1 Input
	var transactionInputs []byte
	for i := 0; i < 32; i++ {
		transactionInputs = append(transactionInputs, 0)
	}
	txInPrevOutIndex, err := hex.DecodeString("ffffffff")
	if err != nil {
		panic(err)
	}

	coinbaseInputs := transactionInputCount
	coinbaseInputs = append(coinbaseInputs, transactionInputs...)
	coinbaseInputs = append(coinbaseInputs, txInPrevOutIndex...)

	return coinbaseInputs
}

func signatureInitial(template rpc.BlockTemplate, arbitrary []byte) []byte {
	// Though this is in the "arbitrary" section, it's still required.
	heightBytes := eightLittleEndianBytes(template.Height)
	heightBytes = significantBytesWithLengthHeader(heightBytes)
	signatureInitial := heightBytes

	signatureInitial = append(signatureInitial, arbitrary...)

	return signatureInitial
}

func getApplicationArbitrary() []byte {
	// In case we ever need to differ blocktemplatetime from headertime; headertime is "pool processsed block at this time"
	now := eightLittleEndianBytes(time.Now().Unix())
	now = significantBytesWithLengthHeader(now)

	placeHolder := []byte{0}

	return append(now, placeHolder...)
}

// "Coinbase final" input - transaction outputs
func coinbaseTransactionOutputsForCoin(t rpc.BlockTemplate, poolPubScriptKey string) ([]byte, error) {
	outputsCount := 0

	// Witness commit comment
	var witnessCommitOutput []byte
	if t.DefaultWitnessCommitment != "" {
		witnessCommit, err := hex.DecodeString(t.DefaultWitnessCommitment)
		if err != nil {
			return nil, err
		}

		amount := []byte{0}
		witnessCommitLen := []byte{byte(len(witnessCommit))}

		witnessCommitOutput = amount
		witnessCommit = append(witnessCommit, witnessCommitLen...)
		witnessCommit = append(witnessCommit, witnessCommit...)

		outputsCount++
	}

	// if(coin has payee, masternodes, founderOutputs, or minerFunds) {customFunctions()}
	// See https://github.com/oliverw/miningcore/blob/master/src/Miningcore/Blockchain/Bitcoin/BitcoinJob.cs#L235
	// if you mine alt coins that have those features ^, there's also post processing on coinbase.

	// Pool reward output
	coinBaseValueHex := fmt.Sprintf("%016x", t.CoinBaseValue)
	amount, err := hex.DecodeString(coinBaseValueHex)
	if err != nil {
		return nil, err
	}
	poolPubScriptKeyBytes, err := hex.DecodeString(poolPubScriptKey)
	if err != nil {
		return nil, err
	}

	poolPubScriptKeyLen := []byte{byte(len(poolPubScriptKeyBytes))}

	rewardOutput := reverse(amount) // Abstract this out to TxOut(amount, pubScriptKey)
	rewardOutput = append(rewardOutput, poolPubScriptKeyLen...)
	rewardOutput = append(rewardOutput, poolPubScriptKeyBytes...)
	outputsCount++

	// Put all the outputs together
	outputsCountBytes := []byte{byte(outputsCount)}
	txs := witnessCommitOutput
	txs = append(txs, rewardOutput...)
	stream := append(outputsCountBytes, txs...)

	return stream, nil
}
