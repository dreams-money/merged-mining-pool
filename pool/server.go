package pool

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

type PoolServer struct {
	sync.RWMutex
	config            *config.Config
	activeNodes       BlockChainNodesMap
	rpcManagers       map[string]*rpc.Manager
	connectionTimeout time.Duration
	templates         Pair
	workCache         bitcoin.Work
	shareBuffer       []persistence.Share
}

func NewServer(cfg *config.Config, rpcManagers map[string]*rpc.Manager) *PoolServer {
	if len(cfg.PoolName) < 1 {
		log.Println("Pool must have a name")
	}
	if len(cfg.BlockchainNodes) < 1 {
		log.Println("Pool must have at least 1 blockchain node to work from")
	}
	if len(cfg.BlockChainOrder) < 1 {
		log.Println("Pool must have a blockchain order to tell primary vs aux")
	}

	pool := &PoolServer{
		config:      cfg,
		rpcManagers: rpcManagers,
	}

	return pool
}

func (pool *PoolServer) Start() {
	initiateSessions()
	pool.loadBlockchainNodes()
	pool.startBufferManager()

	amountOfChains := len(pool.config.BlockChainOrder) - 1
	pool.templates.AuxBlocks = make([]bitcoin.AuxBlock, amountOfChains)

	// Initial work creation
	panicOnError(pool.fetchRpcBlockTemplatesAndCacheWork())
	work, err := pool.generateWorkFromCache(false)
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
	var err error
	response, err := p.GetPrimaryNode().RPC.GetBlockTemplate()
	if err != nil {
		return template, nil, errors.New("RPC error: " + err.Error())
	}

	err = json.Unmarshal(response, &template)
	if err != nil {
		return template, nil, err
	}

	var auxBlock bitcoin.AuxBlock

	if p.config.GetAux1() != "" {
		response, err = p.GetAux1Node().RPC.CreateAuxBlock(p.GetAux1Node().RewardTo)
		if err != nil {
			log.Println("No aux block found: " + err.Error())
			return template, nil, nil
		}

		err = json.Unmarshal(response, &auxBlock)
		if err != nil {
			return template, nil, err
		}
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

func logFatalOnError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
