package pool

import (
	"context"
	"encoding/hex"
	"errors"
	"log"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/rpc"
	"github.com/go-zeromq/zmq4"
)

type blockChainNodesMap map[string]blockChainNode // "blockChainName" => activeNode

type blockChainNode struct {
	NotifyURL          string
	RPC                *rpc.RPCClient
	Network            string
	RewardPubScriptKey string // TODO - this is very bitcoin specific.  Abstract to interface.
	RewardAddress      string
	NetworkDifficulty  float64
}

func (p *PoolServer) GetPrimaryNode() blockChainNode {
	return p.activeNodes[p.config.GetPrimary()]
}

func (p *PoolServer) GetAux1Node() blockChainNode {
	return p.activeNodes[p.config.GetAux1()]
}

type hashblockCounterMap map[string]uint32 // "blockChainName" => hashblock msg counter

func (pool *PoolServer) loadBlockchainNodes() {
	pool.activeNodes = make(blockChainNodesMap)
	for _, blockChainName := range pool.config.BlockChainOrder {

		node := pool.config.BlockchainNodes[blockChainName][0] // Always try to return to the primary node

		// TODO - add node failover..

		rpcClient := rpc.NewRPCClient(node.Name, node.RPC_URL, node.RPC_Username, node.RPC_Password, node.Timeout)

		chainInfo, err := rpcClient.GetBlockChainInfo()
		logFatalOnError(err)

		address, err := rpcClient.ValidateAddress(node.RewardAddress)
		logFatalOnError(err)
		rewardPubScriptKey := address.ScriptPubKey

		newNode := blockChainNode{
			NotifyURL:          node.NotifyURL,
			RPC:                rpcClient,
			Network:            chainInfo.Chain,
			RewardPubScriptKey: rewardPubScriptKey,
			RewardAddress:      node.RewardAddress,
			NetworkDifficulty:  chainInfo.NetworkDifficulty,
		}
		pool.activeNodes[blockChainName] = newNode
	}
}

func (pool *PoolServer) listenForBlockNotifications() error {
	notifyChannel := make(chan hashBlockResponse)
	hashblockCounterMap := make(hashblockCounterMap)

	for blockChainName := range pool.activeNodes {
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

		err := pool.fetchRpcBlockTemplatesAndCacheWork()
		logOnError(err)
		work, err := pool.generateWorkFromCache(true)
		logOnError(err)
		pool.broadcastWork(work)
	}
}

// Ultimate program OUTPUT
func (p *PoolServer) submitBlockToChain(block bitcoin.BitcoinBlock, work bitcoin.Work, chainName string) error {
	submission, err := block.Submit()
	if err != nil {
		return err
	}

	submit := []any{
		any(submission),
	}
	success, err := p.activeNodes[chainName].RPC.SubmitBlock(submit)

	if !success || err != nil {
		return errors.New("⚠️  Node Rejection: " + err.Error())
	}

	return nil
}

func (p *PoolServer) submitAuxBlock(primaryBlock bitcoin.BitcoinBlock, aux1Block bitcoin.AuxBlock, chainName string) error {
	auxpow := bitcoin.MakeAuxPow(primaryBlock)
	success, err := p.activeNodes[chainName].RPC.SubmitAuxBlock(aux1Block.Hash, auxpow.Serialize())
	if !success {
		return errors.New("⚠️  Failed to submit aux block: " + err.Error())
	}
	return err
}

type hashBlockResponse struct {
	blockChainName    string
	previousBlockHash string
	blockHashCounter  uint32
}

func (p *PoolServer) createZMQSubscriptionToHashBlock(blockChainName string, hashBlockChannel chan hashBlockResponse) (zmq4.Socket, error) {
	sub := zmq4.NewSub(context.Background())

	url := p.activeNodes[blockChainName].NotifyURL
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
