package bitcoin

func (b BitcoinBlock) ValidMainnetAddress(address string) bool {
	return b.chain.ValidMainnetAddress(address)
}
func (b BitcoinBlock) ValidTestnetAddress(address string) bool {
	return b.chain.ValidTestnetAddress(address)
}
