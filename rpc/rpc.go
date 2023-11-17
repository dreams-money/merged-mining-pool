package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RPCClient struct {
	NodeUrl string
	Name    string
	client  *http.Client
}

func NewRPCClient(name, rpcURL, rpcUser, rpcPassword, timeout string) *RPCClient {
	urlParts := strings.Split(rpcURL, "://")
	rpcClient := &RPCClient{
		Name:    name,
		NodeUrl: urlParts[0] + "://" + rpcUser + ":" + rpcPassword + "@" + urlParts[1]}

	timeOutIntv, err := time.ParseDuration(timeout)
	if err != nil {
		panic("util: Can'blockTemplate parse duration `" + timeout + "`: " + err.Error())
	}

	rpcClient.client = &http.Client{
		Timeout: timeOutIntv,
	}

	return rpcClient
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  rpcError        `json:"error"`
	ID     int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *RPCClient) doRequest(method string, params []interface{}) (rpcResponse, int, error) {
	type rpcRequest struct {
		ID             int           `json:"id"`
		JsonRPCVersion string        `json:"jsonrpc"`
		Method         string        `json:"method"`
		Parameters     []interface{} `json:"params"`
	}

	var jsonStr rpcRequest
	jsonStr.ID = 1219
	jsonStr.JsonRPCVersion = "2.0"
	jsonStr.Method = method
	jsonStr.Parameters = params

	var rpcResp rpcResponse

	s, err := json.Marshal(jsonStr)
	if err != nil {
		return rpcResp, 0, err
	}

	req, err := http.NewRequest("POST", r.NodeUrl, bytes.NewBuffer(s))
	if err != nil {
		return rpcResp, 0, err
	}
	req.Header.Add("accept", "application/json")

	if params != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return rpcResp, 0, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&rpcResp)

	return rpcResp, resp.StatusCode, nil
}

func (r *RPCClient) GetPeerCount() (int64, error) { // getconnectioncount
	var n int64
	resp, status, err := r.doRequest("getconnectioncount", nil)
	if err != nil {
		return 0, err
	}

	if status != 200 {
		return 0, handleHttpError(resp, status)
	}

	json.Unmarshal(resp.Result, &n)

	return n, nil
}

func (r *RPCClient) GetBlockTemplate() (json.RawMessage, error) {
	params := make([]interface{}, 1)
	rules := make(map[string][]string)
	rules["rules"] = make([]string, 2)
	rules["rules"][0] = "mweb"
	rules["rules"][1] = "segwit"
	params[0] = rules
	resp, status, err := r.doRequest("getblocktemplate", params)
	if err != nil {
		return json.RawMessage{}, err
	}

	if status != 200 {
		return json.RawMessage{}, handleHttpError(resp, status)
	}

	return resp.Result, nil
}

func (r *RPCClient) CreateAuxBlock(rewardAddress string) (json.RawMessage, error) {
	params := make([]any, 1)
	params[0] = rewardAddress
	resp, status, err := r.doRequest("createauxblock", params)
	if err != nil {
		return json.RawMessage{}, err
	}
	if status != 200 {
		return json.RawMessage{}, handleHttpError(resp, status)
	}

	return resp.Result, nil
}

type GetBlockReplyPart struct {
	Height     uint64  `json:"height"`
	Difficulty float64 `json:"difficulty"`
}

type GetBlockReply struct {
	Hash         string   `json:"id"`
	Difficulty   float64  `json:"difficulty"`
	Timestamp    int      `json:"time"`
	Size         int      `json:"size"`
	Height       uint64   `json:"height"`
	ParentID     string   `json:"previousblockhash"`
	Nonce        string   `json:"nonce64"` // From Block Reply
	Miner        string   `json:"miner"`   // From Explorer API
	Transactions []string `json:"tx"`      // From Block Reply
}

func (r *RPCClient) GetLatestBlock() (GetBlockReplyPart, error) {
	var reply GetBlockReplyPart

	resp, status, err := r.doRequest("getbestblockhash", nil)
	if err != nil {
		return reply, err
	}

	if status != 200 {
		return reply, handleHttpError(resp, status)
	}

	var blockHash string
	json.Unmarshal(resp.Result, &blockHash)

	block, err := r.GetBlockByHash(blockHash)
	if err != nil {
		return reply, err
	}

	reply.Height = block.Height
	reply.Difficulty = block.Difficulty

	return reply, nil
}

func (r *RPCClient) GetBlockByHash(hash string) (*GetBlockReply, error) {
	var reply GetBlockReply
	params := make([]interface{}, 1)
	params[0] = hash
	resp, status, err := r.doRequest("getblock", params)

	if err != nil {
		return &reply, err
	}

	if status != 200 {
		return &reply, handleHttpError(resp, status)
	}

	json.Unmarshal(resp.Result, &reply)

	return &reply, nil
}

func (r *RPCClient) GetBlockByHeight(height int64) (*GetBlockReply, error) {
	var reply GetBlockReply
	rpcParams := make([]interface{}, 1)
	rpcParams[0] = height
	resp, status, err := r.doRequest("getblockhash", rpcParams)
	if err != nil {
		return &reply, err
	}

	if status != 200 {
		return &reply, handleHttpError(resp, status)
	}

	var blockHash string
	json.Unmarshal(resp.Result, &blockHash)

	block, err := r.GetBlockByHash(blockHash)
	if err != nil {
		return &reply, err
	}

	return block, nil
}

func (r *RPCClient) SubmitBlock(submission []interface{}) (bool, error) {
	rpcParams := make([]interface{}, 1)

	// This ultimately will be the point of inversion for each chain block...
	// Each chain block will have it's own rpc.SubmitBlock.. well, all RPC methods really
	rpcParams[0] = submission[0].(string)

	resp, status, err := r.doRequest("submitblock", rpcParams)
	if err != nil {
		return false, err
	}

	result := string(resp.Result)
	if status != 200 || result != "null" {
		m := "HTTP (%v) %v error-msg: %v"
		m = fmt.Sprintf(m, status, result, resp.Error.Message)
		return false, errors.New(m)
	}

	return true, nil
}

func (r *RPCClient) SubmitAuxBlock(auxBlockHash string, primaryAuxPow string) (bool, error) {
	rpcParams := make([]any, 2)

	rpcParams[0] = auxBlockHash
	rpcParams[1] = primaryAuxPow

	resp, status, err := r.doRequest("submitauxblock", rpcParams)
	if err != nil {
		return false, err
	}
	result := string(resp.Result)
	if status != 200 || result != "true" {
		m := "HTTP (%v) %v error-msg: %v"
		m = fmt.Sprintf(m, status, result, resp.Error.Message)
		return false, errors.New(m)
	}

	return true, nil
}

type validateAddressResponse struct {
	ScriptPubKey string `json:"scriptPubKey"`
}

func (r *RPCClient) ValidateAddress(address string) (validateAddressResponse, error) {
	var response validateAddressResponse

	rpcParams := make([]interface{}, 1)
	rpcParams[0] = address

	resp, status, err := r.doRequest("validateaddress", rpcParams)
	if err != nil {
		return response, err
	}
	if status != 200 {
		return response, errors.New("RPC: `validateaddress` failed HTTP 200")
	}

	err = json.Unmarshal(resp.Result, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

type blockChainInfoResponse struct {
	Chain             string  `json:"chain"`
	NetworkDifficulty float64 `json:"difficulty"`
}

func (r *RPCClient) GetBlockChainInfo() (blockChainInfoResponse, error) {
	var response blockChainInfoResponse

	resp, status, err := r.doRequest("getblockchaininfo", nil)
	if err != nil {
		return response, err
	}
	if status != 200 {
		return response, errors.New("RPC: `getblockchaininfo` failed HTTP 200")
	}

	err = json.Unmarshal(resp.Result, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func handleHttpError(response rpcResponse, status int) error {
	return errors.New("HTTP " + strconv.Itoa(status) + ": " + response.Error.Message)
}
