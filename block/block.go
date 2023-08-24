package block

import (
	"math/big"

	"designs.capital/dogepool/blockchain"
	"designs.capital/dogepool/rpc"
	"designs.capital/dogepool/template"
)

func GetChain(chainName, poolRewardAddress string, rpcClient *rpc.RPCClient) BlockchainProcessor {
	processor := bitcoin{}

	var chain blockchain.Blockchain

	switch chainName {
	case "dogecoin":
		chain = blockchain.Dogecoin{}
	case "litecoin":
		chain = blockchain.Litecoin{}
	default:
		panic("Unknown blockchain: " + chainName)
	}

	err := processor.init(rpcClient, poolRewardAddress, chain)
	if err != nil {
		panic(err)
	}

	return &processor
}

type BlockchainProcessor interface {
	GenerateWork(template *template.Block, arbitrary string) ([]interface{}, error)
	GenerateAuxHeader(template *template.Block, signature string) (string, error)
	CalculateSum(template *template.Block, work []interface{}) (*big.Int, error)
	PrepareWorkForSubmission(template template.Block, work []interface{}) ([]interface{}, error)
	ShareMultiplier() float32

	ValidMainnetAddress(address string) bool
	ValidTestnetAddress(address string) bool

	// TODO - This signature isn't portable.. fix.  remove reward address.
	init(rpcClient *rpc.RPCClient, rewardAddress string, chain blockchain.Blockchain) error
}
