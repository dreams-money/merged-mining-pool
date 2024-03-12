package bitcoin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type rpcClient struct {
	nodeURL  string
	sickRate int
	client   *http.Client
}

var client *rpcClient

func makeRPCClient(url, timeout string) *rpcClient {
	timeOut, err := time.ParseDuration(timeout)
	if err != nil {
		panic("util: Can'blockTemplate parse duration `" + timeout + "`: " + err.Error())
	}

	c := &rpcClient{}
	c.nodeURL = url
	c.client = &http.Client{
		Timeout: timeOut,
	}

	return c
}

func (r *rpcClient) Call(method string, args, reply any) (error, int) {
	type rpcRequest struct {
		ID             int    `json:"id"`
		JsonRPCVersion string `json:"jsonrpc"`
		Method         string `json:"method"`
		Parameters     any    `json:"params"`
	}

	var jsonStr rpcRequest
	jsonStr.ID = 1219
	jsonStr.JsonRPCVersion = "2.0"
	jsonStr.Method = method
	jsonStr.Parameters = args

	s, err := json.Marshal(jsonStr)
	if err != nil {
		return err, 0
	}

	req, err := http.NewRequest("POST", r.nodeURL, bytes.NewBuffer(s))
	if err != nil {
		return err, 0
	}
	req.Header.Add("accept", "application/json")
	if args != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.sickRate++
		return err, 0
	}

	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(reply), resp.StatusCode
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

func RpcTemplate(rpcURL, timeout string) *Template {
	type rpcResponse struct {
		Result json.RawMessage `json:"result"`
		Error  rpcError        `json:"error"`
		ID     int             `json:"id"`
	}

	var resp rpcResponse
	var err error
	var template Template

	if client == nil {
		client = makeRPCClient(rpcURL, timeout)
	}

	args := make([]interface{}, 1)
	rules := make(map[string][]string)
	rules["rules"] = make([]string, 2)
	rules["rules"][0] = "mweb"
	rules["rules"][1] = "segwit"
	args[0] = rules

	err, _ = client.Call("getblocktemplate", args, &resp)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(resp.Result, &template)
	if err != nil {
		panic(err)
	}

	return &template
}

func (b BitcoinBlock) RpcSubmit(rpcURL, timeout, submission string) error {
	type rpcResponse struct {
		Result json.RawMessage `json:"result"`
		Error  rpcError        `json:"error"`
		ID     int             `json:"id"`
	}

	var resp rpcResponse
	var err error

	if client == nil {
		client = makeRPCClient(rpcURL, timeout)
	}

	args := make([]string, 1)
	args[0] = submission

	err, status := client.Call("submitblock", args, resp)
	if err != nil {
		return err
	}
	if status != 200 {
		m := "http status: %v, %v"
		m = fmt.Sprintf(m, strconv.Itoa(status), string(resp.Result))
		return errors.New(m)
	}

	return nil
}
