package rpc

type Config struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Timeout  string `json:"timeout"`
}
