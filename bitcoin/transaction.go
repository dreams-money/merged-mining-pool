package bitcoin

func TransactionOut(amount, pubScriptKey string) string {
	lengthBytes := uint(len(pubScriptKey) / 2)
	return amount + varUint(lengthBytes) + pubScriptKey
}
