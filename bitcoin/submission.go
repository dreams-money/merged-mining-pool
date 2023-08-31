package bitcoin

type Submission struct {
	Header            string
	TransactionCount  string
	Coinbase          string
	TransactionBuffer string
}

// https://developer.bitcoin.org/reference/block_chain.html#serialized-blocks

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

	return submission.Serialize()
}

func (b *BitcoinBlock) buildTransactionBuffer() string {
	buffer := ""
	for _, transaction := range b.Template.Transactions {
		buffer = buffer + transaction.Data
	}
	return buffer
}
