package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

type coinNodeConfig struct {
	Name         string `json:"name"`
	RPC_URL      string `json:"rpc_url"`
	RPC_Username string `json:"rpc_username"`
	RPC_Password string `json:"rpc_password"`
	Timeout      string `json:"timeout"`
	NotifyURL    string `json:"block_notify_url"`
	RewardTo     string `json:"reward_to"`
}

type blockChainNodesConfigMap map[string][]coinNodeConfig // coin name => [] of blockNodes

type BlockChainOrder []string

func (b BlockChainOrder) GetPrimary() string {
	return b[0]
}

func (b BlockChainOrder) GetAux1() string {
	if len(b) < 2 {
		return ""
	}

	return b[1]
}

type sqlConfig struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}

type apiConfig struct {
	Port string `json:"port"`
}

type recipient struct {
	Address    string  `json:"address"`
	Percentage float64 `json:"percentage"`
}
type Chain struct {
	Name                 string
	RewardFrom           string      `json:"reward_from"`
	MinerMinimumPayment  float32     `json:"miner_min_payment"`
	PoolRewardRecipients []recipient `json:"pool_rewards"`
}

type Chains map[string]Chain // chainName => chain payout config

type PayoutsConfig struct {
	Interval string `json:"interval"`
	Scheme   string `json:"scheme"`
	Chains   `json:"chains"`
}

type Config struct {
	PoolName           string                   `json:"pool_name"`
	BlockSignature     string                   `json:"block_signature"`
	BlockchainNodes    blockChainNodesConfigMap `json:"blockchains"` // Map order in this config file determines primary vs aux nodes.
	Port               string                   `json:"port"`
	MaxConnections     int                      `json:"max_connections"`
	ConnectionTimeout  string                   `json:"connection_timeout"`
	PoolDifficulty     float64                  `json:"pool_difficulty"`
	BlockChainOrder    `json:"merged_blockchain_order"`
	ShareFlushInterval string        `json:"share_flush_interval"`
	HashrateWindow     string        `json:"hashrate_window"`
	PoolStatsInterval  string        `json:"pool_stats_interval"`
	Persister          sqlConfig     `json:"persistence"`
	API                apiConfig     `json:"api"`
	Payouts            PayoutsConfig `json:"payouts"`
	AppStatsInterval   string        `json:"app_stats_interval"`
}

func LoadConfig(fileName string) *Config {
	file, err := os.Open(fileName)
	logFatalOnError(err)
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	logFatalOnError(err)

	var c Config
	err = json.Unmarshal(fileBytes, &c)
	logFatalOnError(err)

	if len(c.BlockchainNodes) < 1 {
		panic("You need to configure coin nodes")
	}

	return &c
}

func logFatalOnError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
