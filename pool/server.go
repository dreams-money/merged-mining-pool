package pool

import (
	"errors"
	"log"
	"time"

	"designs.capital/dogepool/config"
	"designs.capital/dogepool/template"
)

const maxHistory = 3

type PoolServer struct {
	config            *config.Config
	coinNodes         blockChainNodesMap
	connectionTimeout time.Duration
	templatesHistory  template.TemplatesHistory
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

	pool.templatesHistory = template.NewTemplatesHistory(maxHistory)

	return pool
}

func (pool *PoolServer) Start() {
	initiateSessions()
	pool.loadBlockchainNodes()
	go pool.listenForConnections()

	// Initial work creation
	pool.fetchAndCacheRpcBlockTemplates()
	work, err := pool.generateWorkFromCache(false)
	panicOnError(err)
	pool.broadcastWork(work)

	// There after..
	panicOnError(pool.listenForBlockNotifications())
}

func (pool *PoolServer) broadcastWork(work Work) {
	request := miningNotify(work)
	err := notifyAllSessions(request)
	logOnError(err)
}

func (p *PoolServer) fetchAllBlockTemplatesFromRPC() (template.MergedCoinPairs, error) { // This
	var templates template.MergedCoinPairs

	for _, blockchainName := range p.config.BlockChainOrder {
		node := p.coinNodes[blockchainName]
		rpcBlockTemplate, err := node.RPC.GetBlockTemplate()
		if err != nil {
			return templates, errors.New("RPC error: " + err.Error())
		}
		coinTemplate := template.Block{
			BlockchainName:   blockchainName,
			RpcBlockTemplate: rpcBlockTemplate,
		}
		templates = append(templates, coinTemplate)
	}

	return templates, nil
}

func (p *PoolServer) fetchAndCacheRpcBlockTemplates() {
	templates, err := p.fetchAllBlockTemplatesFromRPC()
	if err != nil {
		// TODO rpc.MarkSick()
		log.Println(err)
	}
	p.templatesHistory.AddMergedCoinTemplatePairs(templates)
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
