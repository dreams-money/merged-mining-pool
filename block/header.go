package block

import (
	"encoding/hex"
)

/*
S/O - https://developer.bitcoin.org/reference/block_chain.html#block-headers

02000000 ........................... Block version: 2

b6ff0b1b1680a2862a30ca44d346d9e8
910d334beb48ca0c0000000000000000 ... Hash of previous block's header
9d10aa52ee949386ca9385695f04ede2
70dda20810decd12bc9b048aaab31471 ... Merkle root

24d95a54 ........................... [Unix time][unix epoch time]: 1415239972
30c31b18 ........................... Target: 0x1bc330 * 256**(0x18-3)
fe9f0864 ........................... Nonce
*/

func blockHeader(version uint, previousBlockHash, merkleRootHex, nTime, bits, nonce string) ([]byte, error) {
	versionBytes := fourLittleEndianBytes(version)

	prevBlockHash, err := hex.DecodeString(previousBlockHash)
	if err != nil {
		return nil, err
	}

	merkleRoot, err := hex.DecodeString(merkleRootHex)
	if err != nil {
		return nil, err
	}

	// Often, this is the same value as blockTemplate.CurrTime
	nonceTime, err := hex.DecodeString(nTime)
	if err != nil {
		return nil, err
	}

	bitsHex, err := hex.DecodeString(bits)
	if err != nil {
		return nil, err
	}

	nonceHex, err := hex.DecodeString(nonce)
	if err != nil {
		return nil, err
	}

	blockHeader := versionBytes
	blockHeader = append(blockHeader, reverse(prevBlockHash)...)
	blockHeader = append(blockHeader, merkleRoot...)
	blockHeader = append(blockHeader, reverse(nonceTime)...)
	blockHeader = append(blockHeader, reverse(bitsHex)...)
	blockHeader = append(blockHeader, reverse(nonceHex)...)

	return blockHeader, nil
}
