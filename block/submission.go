package block

import (
	"encoding/hex"
)

/* S/O - https://developer.bitcoin.org/reference/block_chain.html#serialized-blocks */
func createSubmissionHex(header, coinbase []byte, transactionPool []string) (string, error) {
	transactionCount := uint(len(transactionPool) + 1) // 1 for coinbase
	transactionCountBytes, err := varUint(transactionCount)
	if err != nil {
		return "", err
	}
	transactionBuffer, err := buildTransactionBuffer(transactionPool)
	if err != nil {
		return "", err
	}

	submission := header
	submission = append(submission, transactionCountBytes...)
	submission = append(submission, coinbase...)
	submission = append(submission, transactionBuffer...)

	return hex.EncodeToString(submission), nil
}

func buildTransactionBuffer(transactionPool []string) ([]byte, error) {
	var buffer []byte
	for _, transaction := range transactionPool {
		rawTransactions, err := hex.DecodeString(transaction)
		if err != nil {
			return buffer, err
		}
		buffer = append(buffer, rawTransactions...)
	}
	return buffer, nil
}
