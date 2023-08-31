package bitcoin

import "encoding/hex"

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

	return hex.EncodeToString(blockHeader), nil
}
