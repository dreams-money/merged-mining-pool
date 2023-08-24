package blockchain

type Blockchain interface {
	ChainName() string
	BlockDigest(header []byte) ([]byte, error)
	ShareMultiplier() float32
	ValidMainnetAddress(address string) bool
	ValidTestnetAddress(address string) bool
}
