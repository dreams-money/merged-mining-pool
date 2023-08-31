package pool

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"designs.capital/dogepool/bitcoin"
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
		return stratumResponse{}, errors.New("Unknown stratum request method: " + request.Method)
	}
}

func miningSubscribe(request *stratumRequest, client *stratumClient) (stratumResponse, error) {
	var response stratumResponse

	if isBanned(client.ip) {
		return response, errors.New("Client blocked: " + client.ip)
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
	}

	// TODO - make this random.  This currently profiles the amount of connections we have
	sessionID := fmt.Sprintf("%08x", client.sessionID)
	// TODO confirm this is session ID.  Or "Subscription ID.."

	var subscriptions []interface{}
	difficulty := interface{}([]string{"mining.set_difficulty", sessionID})
	notify := interface{}([]string{"mining.notify", sessionID})
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
		return reply, errors.New("Banned client attempted to access: " + client.ip)
	}

	var params []string
	err := json.Unmarshal(request.Params, &params)
	if err != nil {
		return reply, err
	}
	if len(params) < 1 {
		return reply, errors.New("Invalid parameters")
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
	rigID := loginParts[1]

	if len(minerAddresses) != len(pool.config.BlockchainNodes) {
		return authResponse, errors.New("Not enough miner addresses to login")
	}

	// The config has the primarycoinAddress-auxcoinAddress-auxcoinAddress order we need
	blockchainIndex := 0
	for _, blockChainName := range pool.config.BlockChainOrder {
		blockChain := bitcoin.GetChain(blockChainName)
		inputBlockChainAddress := minerAddresses[blockchainIndex]

		if (pool.activeNodes[blockChainName].Network == "test" &&
			!blockChain.ValidTestnetAddress(inputBlockChainAddress)) ||
			(pool.activeNodes[blockChainName].Network == "main" &&
				!blockChain.ValidMainnetAddress(inputBlockChainAddress)) {
			return authResponse, errors.New("Not a valid miner address")
		}

		blockchainIndex++
	}

	log.Printf("Authorized rig: %v mining to addresses: %v", rigID, minerAddresses)

	client.login = loginString

	addSession(client)

	// TODO
	// Write to persistence

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
