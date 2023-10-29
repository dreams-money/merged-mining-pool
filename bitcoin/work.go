package bitcoin

// Interface to stratum JSON work packets
type Work []any

func (b BitcoinBlock) NonceSubmissionSlot() (slotID int) {
	return 4
}

func (b BitcoinBlock) NonceTimeSubmissionSlot() (slotID int) {
	return 3
}

func (b BitcoinBlock) Extranonce2SubmissionSlot() (slotID int, exists bool) {
	return 2, true
}

func (b BitcoinBlock) ShareMultiplier() float64 {
	return b.chain.ShareMultiplier()
}
