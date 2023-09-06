package bitcoin

import "fmt"

type Submission struct {
	Header            string
	TransactionCount  string
	Coinbase          string
	TransactionBuffer string
}

// https://developer.bitcoin.org/reference/block_chain.html#serialized-blocks
// https://en.bitcoin.it/wiki/BIP_0022#Appendix:_Example_Rejection_Reasons

func (s *Submission) Serialize() string {
	return s.Header +
		s.TransactionCount +
		s.Coinbase +
		s.TransactionBuffer
}

func (b *BitcoinBlock) createSubmissionHex() string {
	transactionCount := uint(len(b.Template.Transactions) + 1) // 1 for coinbase

	submission := Submission{
		Header:            b.header,
		TransactionCount:  varUint(transactionCount),
		Coinbase:          b.coinbase,
		TransactionBuffer: b.buildTransactionBuffer(),
	}

	// submissionDebugOutput(submission.Header, submission.TransactionCount, submission.Coinbase, submission.TransactionBuffer, submission.Serialize())
	return submission.Serialize()
}

func (b *BitcoinBlock) buildTransactionBuffer() string {
	buffer := ""
	for _, transaction := range b.Template.Transactions {
		buffer = buffer + transaction.Data
	}
	return buffer
}

func submissionDebugOutput(header, transactionCount, coinbase, transactionBuffer, submission string) {
	fmt.Println()
	fmt.Println("**😱SUBMISSION PARTS😱**")
	fmt.Println()
	fmt.Println("Header", header)
	fmt.Println("TransactionCount", transactionCount)
	fmt.Println("Coinbase", coinbase)
	fmt.Println("TransactionBuffer", transactionBuffer)
	fmt.Println()
	fmt.Println("Submission", submission)
	fmt.Println()
}
