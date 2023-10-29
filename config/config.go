package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

type coinNodeConfig struct {
	Name          string `json:"name"`
	RPC_URL       string `json:"rpc_url"`
	RPC_Username  string `json:"rpc_username"`
	RPC_Password  string `json:"rpc_password"`
	NotifyURL     string `json:"block_notify_url"`
	Timeout       string `json:"timeout"`
	RewardAddress string `json:"reward_address"`
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

type Config struct {
	PoolName           string                   `json:"pool_name"`
	BlockSignature     string                   `json:"block_signature"`
	BlockchainNodes    blockChainNodesConfigMap `json:"blockchains"` // Map order in this config file determines primary vs aux nodes.
	Port               string                   `json:"port"`
	MaxConnections     int                      `json:"max_connections"`
	ConnectionTimeout  string                   `json:"connection_timeout"`
	PoolDifficulty     float32                  `json:"pool_difficulty"`
	BlockChainOrder    `json:"merged_blockchain_order"`
	ShareFlushInterval string    `json:"share_flush_interval"`
	Persister          sqlConfig `json:"persistence"`
}

func LoadConfig(fileName string) *Config {
	file, err := os.Open(fileName)
	logFatalOnError(err)
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	logFatalOnError(err)

	var c Config
	json.Unmarshal(fileBytes, &c)

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
