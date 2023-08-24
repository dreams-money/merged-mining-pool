package pool

import (
	"context"
	"encoding/hex"
	"errors"
	"log"

	"designs.capital/dogepool/block"
	"designs.capital/dogepool/rpc"
	"designs.capital/dogepool/template"
	"github.com/go-zeromq/zmq4"
)

type blockChainNodesMap map[string]blockChainNode // "blockChainName" => activeNode

type blockChainNode struct {
	NotifyURL string
	RPC       *rpc.RPCClient
	Network   string
}

type hashblockCounterMap map[string]uint32 // "blockChainName" => hashblock msg counter

func (pool *PoolServer) loadBlockchainNodes() {
	pool.coinNodes = make(blockChainNodesMap)
	for blockChainName, nodes := range pool.config.BlockchainNodes {
		node := nodes[0] // Always try to return to the primary node

		// TODO - add node failover..

		rpcClient := rpc.NewRPCClient(node.Name, node.RPC_URL, node.RPC_Username, node.RPC_Password, node.Timeout)

		chainInfo, err := rpcClient.GetBlockChainInfo()
		panicOnError(err)

		newNode := blockChainNode{
			NotifyURL: node.NotifyURL,
			RPC:       rpcClient,
			Network:   chainInfo.Chain,
		}
		pool.coinNodes[blockChainName] = newNode
	}
}

func (pool *PoolServer) listenForBlockNotifications() error {
	notifyChannel := make(chan hashBlockResponse)
	hashblockCounterMap := make(hashblockCounterMap)

	for blockChainName := range pool.coinNodes {
		subscription, err := pool.createZMQSubscriptionToHashBlock(blockChainName, notifyChannel)
		defer subscription.Close()
		if err != nil {
			return err
		}
	}

	for {
		msg := <-notifyChannel
		chainName := msg.blockChainName
		prevCount := hashblockCounterMap[chainName]
		newCount := msg.blockHashCounter
		prevBlockHash := msg.previousBlockHash

		m := "**New %v block: %v - %v**"
		log.Printf(m, chainName, newCount, prevBlockHash)

		if prevCount != 0 && (prevCount+1) != newCount {
			m = "We missed a %v block notification, previous count: %v current count: %v"
			log.Printf(m, chainName, prevCount, newCount)
		}

		hashblockCounterMap[chainName] = newCount

		pool.fetchAndCacheRpcBlockTemplates()
		work, err := pool.generateWorkFromCache(true)
		logOnError(err)
		pool.broadcastWork(work)
	}
}

// Ultimate program OUTPUT
func (p *PoolServer) submitBlockToChain(blockTemplate template.Block, work Work, chainName string) error {
	node, found := p.coinNodes[chainName]
	if !found {
		return errors.New("Chain node not found: " + chainName)
	}
	nodeConfig, found := p.config.BlockchainNodes[chainName]
	if !found {
		return errors.New("Chain config not found: " + chainName)
	}

	coin := block.GetChain(chainName, nodeConfig[0].RewardAddress, node.RPC)
	submission, err := coin.PrepareWorkForSubmission(blockTemplate, work)
	if err != nil {
		return err
	}

	success, err := p.coinNodes[chainName].RPC.SubmitBlock(submission)

	if !success || err != nil {
		return errors.New("Node Rejection: " + err.Error())
	}

	return nil
}

type hashBlockResponse struct {
	blockChainName    string
	previousBlockHash string
	blockHashCounter  uint32
}

func (p *PoolServer) createZMQSubscriptionToHashBlock(blockChainName string, hashBlockChannel chan hashBlockResponse) (zmq4.Socket, error) {
	sub := zmq4.NewSub(context.Background())

	url := p.coinNodes[blockChainName].NotifyURL
	err := sub.Dial(url)
	if err != nil {
		return sub, err
	}

	err = sub.SetOption(zmq4.OptionSubscribe, "hashblock")
	if err != nil {
		return sub, err
	}

	logErr := func(msg zmq4.Msg, err error) zmq4.Msg {
		if err != nil {
			log.Println(err)
		}

		return msg
	}

	go func() {
		for {
			msg := logErr(sub.Recv())

			if len(msg.Frames) > 2 {
				var blockHashCounter uint32
				blockHashCounter |= uint32(msg.Frames[2][0])
				blockHashCounter |= uint32(msg.Frames[2][1]) << 8
				blockHashCounter |= uint32(msg.Frames[2][2]) << 16
				blockHashCounter |= uint32(msg.Frames[2][3]) << 24

				hashBlockChannel <- hashBlockResponse{
					blockChainName:    blockChainName,
					previousBlockHash: hex.EncodeToString(msg.Frames[1]),
					blockHashCounter:  blockHashCounter,
				}
			}

		}
	}()

	return sub, nil
}
