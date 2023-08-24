package template

import "designs.capital/dogepool/rpc"

type Block struct {
	BlockchainName   string
	RpcBlockTemplate rpc.BlockTemplate
	RequestedWork    []interface{}
}
