package bitcoin

import (
	"encoding/hex"
	"fmt"
)

// https://developer.bitcoin.org/reference/block_chain.html#block-headers

func blockHeader(version uint, previousBlockHash, merkleRootHex, nTime, bits, nonce string) (string, error) {
	versionBytes := fourLittleEndianBytes(version)

	prevBlockHash, err := hex.DecodeString(previousBlockHash)
	if err != nil {
		return "", err
	}

	merkleRoot, err := hex.DecodeString(merkleRootHex)
	if err != nil {
		return "", err
	}

	// Often, this is the same value as blockTemplate.CurrTime
	nonceTime, err := hex.DecodeString(nTime)
	if err != nil {
		return "", err
	}

	bitsHex, err := hex.DecodeString(bits)
	if err != nil {
		return "", err
	}

	nonceHex, err := hex.DecodeString(nonce)
	if err != nil {
		return "", err
	}

	blockHeader := versionBytes
	blockHeader = append(blockHeader, reverse(prevBlockHash)...)
	blockHeader = append(blockHeader, merkleRoot...)
	blockHeader = append(blockHeader, reverse(nonceTime)...)
	blockHeader = append(blockHeader, reverse(bitsHex)...)
	blockHeader = append(blockHeader, reverse(nonceHex)...)

	// headerDebugOutput(versionBytes, prevBlockHash, merkleRoot, nonceTime, bitsHex, nonceHex, blockHeader)
	return hex.EncodeToString(blockHeader), nil
}

func headerDebugOutput(versionBytes, prevBlockHash, merkleRoot, nonceTime, bitsHex, nonceHex, blockHeader []byte) {
	fmt.Println()
	fmt.Println("**Block HEADER**")
	fmt.Println()
	fmt.Println("version", hex.EncodeToString(versionBytes))
	fmt.Println("prevBlockHash", hex.EncodeToString(reverse(prevBlockHash)))
	fmt.Println("merkleRoot", hex.EncodeToString(merkleRoot))
	fmt.Println("nonceTime", hex.EncodeToString(reverse(nonceTime)))
	fmt.Println("bitsHex", hex.EncodeToString(reverse(bitsHex)))
	fmt.Println("nonceHex", hex.EncodeToString(reverse(nonceHex)))
	fmt.Println()
	fmt.Println("Header", hex.EncodeToString(blockHeader))
	fmt.Println()
}
