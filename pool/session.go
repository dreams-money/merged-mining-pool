package pool

type sessionMap map[int]*stratumClient

var sessions sessionMap
var sessionIndex int = 0

func initiateSessions() {
	sessions = make(sessionMap)
}

func addSession(client *stratumClient) {
	sessions[sessionIndex] = client
}

func removeSession(sessionID int) {
	delete(sessions, sessionID)
}
