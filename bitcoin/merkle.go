package bitcoin

import (
	"encoding/hex"
)

// https://developer.bitcoin.org/reference/block_chain.html#merkle-trees

/*******************************************
*******  Merkle Illustration ***************
********************************************


a   b c   d	e	f <- level1
 \ /   \ /   \ /
  G     H     I   G  <- double up if odd (level2)
   \   /       \ /
     J          K <- level3,
	             ... level4,
			     ... levelN
      \___   __/
	      \ /
           L

	Steps is G, J, L

Little endian writes..

********************************************/

func (t *Template) MerkleSteps() ([]string, error) {
	transactionIDs := make([]string, len(t.Transactions))
	for i, transaction := range t.Transactions {
		idReversed, _ := reverseHexBytes(transaction.ID)
		transactionIDs[i] = idReversed
	}

	return templateMerkleBranchSteps(transactionIDs)
}

func templateMerkleBranchSteps(transactionIDs []string) ([]string, error) {
	steps := []string{}
	l := len(transactionIDs)

	if l == 0 {
		return steps, nil
	}

	_, steps, err := getMerkleRoot(transactionIDs)
	if err != nil {
		return steps, err
	}

	return steps, nil
}

func getMerkleRoot(transactionIDs []string) (string, []string, error) {
	l := len(transactionIDs)
	var steps []string

	if l == 0 {
		var empty []byte
		slice := doubleSha256Bytes(empty)
		return hex.EncodeToString(slice[:]), steps, nil
	} else if l == 1 {
		steps = append(steps, transactionIDs[0])
		return transactionIDs[0], steps, nil
	} else if l%2 == 1 {
		transactionIDs = append(transactionIDs, transactionIDs[l-1]) // Last or first?
		l++
	}

	if l == 2 {
		mergedHex, err := mergeHex(transactionIDs[0], transactionIDs[1])
		steps = append(steps, mergedHex)
		return mergedHex, steps, err
	}

	level := transactionIDs
	for l > 1 {
		level, err := scanMerkleLevel(level, steps)
		if err != nil {
			return "", steps, err
		}
		l = len(level)
	}

	return level[0], steps, nil
}

func scanMerkleLevel(pairs, steps []string) ([]string, error) {
	var level []string
	l := len(pairs)
	for i := 0; i < l; i = i + 2 {
		merged, err := mergeHex(pairs[i], pairs[i+1])
		if err != nil {
			return level, err
		}
		if i == 0 {
			steps = append(steps, merged)
		}
		level = append(level, merged)
	}
	return level, nil
}

func mergeHex(one, two string) (string, error) {
	oneBytes, err := hex.DecodeString(one)
	if err != nil {
		return "", err
	}
	twoBytes, err := hex.DecodeString(two)
	if err != nil {
		return "", err
	}

	merged := doubleSha256Bytes(append(oneBytes, twoBytes...))

	mergedHex := hex.EncodeToString(merged[:])
	return mergedHex, nil
}

func makeHeaderMerkleRoot(coinbase string, merkleBranchSteps []string) (string, error) {
	block, err := hex.DecodeString(coinbase)
	if err != nil {
		return "", err
	}
	for _, branch := range merkleBranchSteps {
		branchBytes, err := hex.DecodeString(branch)
		if err != nil {
			return "", err
		}
		joined := doubleSha256Bytes(append(block, branchBytes...))
		block = joined[:]
	}

	return hex.EncodeToString(block), nil
}
