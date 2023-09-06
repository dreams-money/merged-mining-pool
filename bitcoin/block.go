package bitcoin

type Block interface {
	ChainName() string

	NonceSubmissionSlot() int
	NonceTimeSubmissionSlot() int

	Extranonce2SubmissionSlot() (int, bool)
	ShareMultiplier() float32

	ValidMainnetAddress(address string) bool
	ValidTestnetAddress(address string) bool

	init(Blockchain)
}
