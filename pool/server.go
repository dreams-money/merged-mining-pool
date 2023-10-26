package pool

import (
	"errors"
	"log"
	"time"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/config"
)

const maxHistory = 3

type PoolServer struct {
	config            *config.Config
	activeNodes       blockChainNodesMap
	connectionTimeout time.Duration
	templates         Pair
	workCache         bitcoin.Work
}

func NewServer(cfg *config.Config) *PoolServer {
	if len(cfg.PoolName) < 1 {
		log.Println("Pool must have a name")
	}
	if len(cfg.BlockchainNodes) < 1 {
		log.Println("Pool must have at least 1 blockchain node to work from")
	}
	if len(cfg.BlockChainOrder) < 1 {
		log.Println("Pool must have a blockchain order to tell primary vs aux")
	}

	pool := &PoolServer{config: cfg}

	return pool
}

func (pool *PoolServer) Start() {
	initiateSessions()
	pool.loadBlockchainNodes()
	pool.templates.AuxBlocks = make([]bitcoin.AuxBlock, len(pool.config.BlockChainOrder)-1)

	// Initial work creation
	panicOnError(pool.fetchRpcBlockTemplatesAndCacheWork())
	work, err := pool.generateWorkFromCache(false)

	go pool.listenForConnections()

	panicOnError(err)

	go pool.listenForConnections()
	pool.broadcastWork(work)

	// There after..
	panicOnError(pool.listenForBlockNotifications())
}

func (pool *PoolServer) broadcastWork(work bitcoin.Work) {
	request := miningNotify(work)
	err := notifyAllSessions(request)
	logOnError(err)
}

func (p *PoolServer) fetchAllBlockTemplatesFromRPC() (bitcoin.Template, *bitcoin.AuxBlock, error) {
	var template bitcoin.Template
	var auxBlock bitcoin.AuxBlock
	var err error

	primaryNode := p.activeNodes[p.config.GetPrimary()]
	aux1Node := p.activeNodes[p.config.GetAux1()]

	template, err = primaryNode.RPC.GetBlockTemplate()
	if err != nil {
		return template, nil, errors.New("RPC error: " + err.Error())
	}

	auxBlock, err = aux1Node.RPC.CreateAuxBlock(aux1Node.RewardAddress)
	if err != nil {
		log.Println("No aux block found: " + err.Error())
		return template, nil, nil
	}

	return template, &auxBlock, nil
}

func notifyAllSessions(request stratumRequest) error {
	for _, client := range sessions {
		err := sendPacket(request, client)
		logOnError(err)
	}
	log.Printf("Sent work to %v client(s)", len(sessions))
	return nil
}

func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

func logOnError(e error) {
	if e != nil {
		log.Println(e)
	}
}
