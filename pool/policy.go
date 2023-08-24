package pool

func isBanned(ip string) bool {
	return false
}

func surpassedLimitPolicy(ip string) bool {
	return false
}

func banClient(client *stratumClient) {
	removeSession(client.sessionID)

	// BAN IP?  BAN Miner address?
}

func markMalformedRequest(client *stratumClient, jsonPayload []byte) {

}
