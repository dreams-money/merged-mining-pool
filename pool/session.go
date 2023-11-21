package pool

type sessionMap map[string]*stratumClient

var sessions sessionMap

func initiateSessions() {
	sessions = make(sessionMap)
}

func addSession(client *stratumClient) {
	sessions[client.sessionID] = client
}

func removeSession(sessionID string) {
	delete(sessions, sessionID)
}
