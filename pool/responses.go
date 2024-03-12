package pool

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"designs.capital/dogepool/bitcoin"
	"github.com/google/uuid"
)

type stratumResponse struct {
	Id      json.RawMessage       `json:"id"`
	Version string                `json:"jsonrpc,omitempty"`
	Result  interface{}           `json:"result"`
	Error   *stratumErrorResponse `json:"error,omitempty"`
}

type stratumErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (pool *PoolServer) respondToStratumClient(client *stratumClient, requestPayload []byte) error {
	var request stratumRequest
	err := json.Unmarshal(requestPayload, &request)
	if err != nil {
		markMalformedRequest(client, requestPayload)
		log.Println("Malformed stratum request from: " + client.ip)
		return err
	}

	timeoutTime := time.Now().Add(pool.connectionTimeout)
	client.connection.SetDeadline(timeoutTime)

	response, err := handleStratumRequest(&request, client, pool)
	if err != nil {
		return err
	}

	return sendPacket(response, client)
}

func handleStratumRequest(request *stratumRequest, client *stratumClient, pool *PoolServer) (any, error) {
	switch request.Method {
	case "mining.subscribe":
		return miningSubscribe(request, client)
	case "mining.authorize":
		return miningAuthorize(request, client, pool)
	case "mining.extranonce.subscribe":
		return miningExtranonceSubscribe(request, client)
	case "mining.submit":
		return miningSubmit(request, client, pool)
	case "mining.multi_version":
		return nil, nil // ignored
	default:
		return stratumResponse{}, errors.New("unknown stratum request method: " + request.Method)
	}
}

func miningSubscribe(request *stratumRequest, client *stratumClient) (stratumResponse, error) {
	var response stratumResponse

	if isBanned(client.ip) {
		return response, errors.New("client blocked: " + client.ip)
	}

	requestParamsJson, err := request.Params.MarshalJSON()
	if err != nil {
		return response, err
	}

	var requestParams []string
	json.Unmarshal(requestParamsJson, &requestParams)
	if len(requestParams) > 0 {
		clientType := requestParams[0]
		log.Println("New subscription from client type: " + clientType)
		client.userAgent = clientType
	}

	client.sessionID = uuid.NewString()

	var subscriptions []interface{}
	difficulty := interface{}([]string{"mining.set_difficulty", client.sessionID})
	notify := interface{}([]string{"mining.notify", client.sessionID})
	extranonce1 := interface{}(client.extranonce1)
	extranonce2Length := interface{}(4)

	subscriptions = append(subscriptions, difficulty)
	subscriptions = append(subscriptions, notify)

	var responseResult []interface{}
	responseResult = append(responseResult, subscriptions)
	responseResult = append(responseResult, extranonce1)
	responseResult = append(responseResult, extranonce2Length)

	response.Id = request.Id
	response.Result = responseResult

	return response, nil
}

func miningAuthorize(request *stratumRequest, client *stratumClient, pool *PoolServer) (any, error) {
	var reply stratumRequest

	if isBanned(client.ip) {
		return reply, errors.New("banned client attempted to access: " + client.ip)
	}

	var params []string
	err := json.Unmarshal(request.Params, &params)
	if err != nil {
		return reply, err
	}
	if len(params) < 1 {
		return reply, errors.New("invalid parameters")
	}

	authResponse := stratumResponse{
		Result: interface{}(false),
		Id:     request.Id,
	}

	loginString := params[0]
	loginParts := strings.Split(loginString, ".")
	minerAddressesString := loginParts[0]
	// minerAddressString format: primarycoinAddress-auxcoinAddress-auxcoinAddress.rigID
	minerAddresses := strings.Split(minerAddressesString, "-")
	if len(minerAddresses) != len(pool.config.BlockChainOrder) {
		return authResponse, errors.New("not enough miner addresses to login")
	}

	rigID := loginParts[1]

	// The config has the primarycoinAddress-auxcoinAddress-auxcoinAddress order we need
	blockchainIndex := 0
	for _, blockChainName := range pool.config.BlockChainOrder {
		blockChain := bitcoin.GetChain(blockChainName)
		inputBlockChainAddress := minerAddresses[blockchainIndex]

		network := pool.activeNodes[blockChainName].Network
		if (network == "test" && !blockChain.ValidTestnetAddress(inputBlockChainAddress)) ||
			(network == "main" && !blockChain.ValidMainnetAddress(inputBlockChainAddress)) {
			m := "invalid %v %vnet miner address from %v: %v"
			m = fmt.Sprintf(m, blockChainName, network, client.ip, inputBlockChainAddress)
			return authResponse, errors.New(m)
		}

		blockchainIndex++
	}

	log.Printf("Authorized rig: %v mining to addresses: %v", rigID, minerAddresses)

	client.login = loginString

	addSession(client)

	authResponse.Result = interface{}(true)

	err = sendPacket(authResponse, client) // Mining.Auth replies with three packets (1)
	if err != nil {
		return reply, err
	}

	err = sendPacket(miningSetDifficulty(pool.config.PoolDifficulty), client) // Mining.Auth replies with three packets (2)
	if err != nil {
		return reply, err
	}

	work, err := pool.generateWorkFromCache(false)
	if err != nil {
		return reply, err
	}

	reply = miningNotify(work) // Mining.Auth replies with three packets (3)

	return reply, nil
}

func miningExtranonceSubscribe(request *stratumRequest, client *stratumClient) (stratumResponse, error) {
	var response stratumResponse

	// TODO - I need to find a good example for this one

	return response, nil
}

func miningSubmit(request *stratumRequest, client *stratumClient, pool *PoolServer) (stratumResponse, error) {
	response := stratumResponse{
		Result: interface{}(false),
		Id:     request.Id,
	}

	var work bitcoin.Work
	err := json.Unmarshal(request.Params, &work)
	if err != nil {
		return response, err
	}

	err = pool.recieveWorkFromClient(work, client)
	if err != nil {
		log.Println(err)
	}

	response.Result = interface{}(true)

	return response, nil
}
