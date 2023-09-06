package bitcoin

type Transaction struct {
	Data string `json:"data"`
	ID   string `json:"txid"`
	Fee  int    `json:"fee"`
}

type Template struct {
	Version                  uint   `json:"version"`
	PrevBlockHash            string `json:"previousblockhash"`
	Height                   uint   `json:"height"`
	CoinBaseValue            uint   `json:"coinbasevalue"`
	DefaultWitnessCommitment string `json:"default_witness_commitment"`
	Bits                     string `json:"bits"`
	Target                   `json:"target"`
	Transactions             []Transaction `json:"transactions"`
	CurrentTime              uint          `json:"curtime"`
	MimbleWimble             string        `json:"mweb"`
}
