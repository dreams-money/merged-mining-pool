package config

import (
	"encoding/json"
	"io"
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

type Config struct {
	PoolName          string                   `json:"pool_name"`
	BlockSignature    string                   `json:"block_signature"`
	BlockchainNodes   blockChainNodesConfigMap `json:"blockchains"` // Map order in this config file determines primary vs aux nodes.
	Port              string                   `json:"port"`
	MaxConnections    int                      `json:"max_connections"`
	ConnectionTimeout string                   `json:"connection_timeout"`
	PoolDifficulty    float32                  `json:"pool_difficulty"`
	BlockChainOrder   []string                 `json:"merged_blockchain_order"`
}

func LoadConfig(fileName string) *Config {
	file, err := os.Open(fileName)
	panicOnError(err)
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	panicOnError(err)

	var c Config
	json.Unmarshal(fileBytes, &c)

	if len(c.BlockchainNodes) < 1 {
		panic("You need to configure coin nodes")
	}

	return &c
}

func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}
