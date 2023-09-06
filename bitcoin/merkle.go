package bitcoin

import (
	"encoding/hex"
)

// https://github.com/zone117x/node-stratum-pool/blob/master/lib/merkleTree.js#L9

func (t *Template) MerkleSteps() ([]string, error) {
	transactionIDs := make([]string, len(t.Transactions))
	for i, transaction := range t.Transactions {
		// Little endian writes
		idReversed, _ := reverseHexBytes(transaction.ID)
		transactionIDs[i] = idReversed
	}

	return templateMerkleBranchSteps(transactionIDs)
}

func templateMerkleBranchSteps(transactionIDs []string) ([]string, error) {
	steps := []string{}
	levelLength := len(transactionIDs)

	if levelLength == 0 {
		return steps, nil
	}

	var level []string
	startJoinAt := 2

	rightShift := []string{""}
	level = append(rightShift, transactionIDs...)
	levelLength++

	for {
		if levelLength == 1 {
			break
		}

		steps = append(steps, level[1])

		if levelLength%2 == 1 {
			level = append(level, level[len(level)-1])
		}

		var levelJoins []string
		for i := startJoinAt; i < levelLength; i += 2 {
			joined, err := join(level[i], level[i+1])
			if err != nil {
				return steps, err
			}
			levelJoins = append(levelJoins, joined)
		}
		level = append(rightShift, levelJoins...)
		levelLength = len(level)
	}

	return steps, nil
}

func join(one, two string) (string, error) {
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
