package rpc

import (
	"encoding/json"
	"time"
)

func (r *RPCClient) GetWalletBalance() (float64, error) {
	resp, status, err := r.doRequest("getbalance", nil)
	if err != nil {
		return 0, err
	}

	if status != 200 {
		return 0, handleHttpError(resp, status)
	}

	var balance float64
	json.Unmarshal(resp.Result, &balance)

	return balance, nil
}

func (r *RPCClient) isWalletUnlocked() (bool, error) {
	return false, nil
}

func (r *RPCClient) lockWallet() error {
	return nil
}

func (r *RPCClient) SendTransaction(to string, value float64) (string, error) {
	rpcParams := make([]interface{}, 2)
	rpcParams[0] = to
	rpcParams[1] = value
	resp, status, err := r.doRequest("sendtoaddress", rpcParams)
	if err != nil {
		return "", err
	}

	if status != 200 {
		return "", handleHttpError(resp, status)
	}

	var receiptHash string
	json.Unmarshal(resp.Result, &receiptHash)

	return receiptHash, nil
}

type Tx struct {
	Hash string
	Fees int
}

type TxReceipt struct {
	BlockHeight    uint64    `json:""`
	BlockHash      string    `json:"blockhash"`
	BlockTime      time.Time `json:"blocktime"`
	Fee            float32   `json:"fee"`
	ConfirmedCount int64     `json:"confirmations"`
	TxId           string    `json:"txid"`
}

func (r *TxReceipt) Confirmed() bool {
	return r.ConfirmedCount > 1
}

func (r *TxReceipt) Successful() bool {
	return r.Confirmed()
}

func (r *RPCClient) GetTxReceipt(txId string) (*TxReceipt, error) {
	var rcpt TxReceipt
	rpcParams := make([]interface{}, 1)
	rpcParams[0] = txId
	resp, status, err := r.doRequest("gettransaction", rpcParams)
	if err != nil {
		return &rcpt, err
	}

	if status != 200 {
		return &rcpt, handleHttpError(resp, status)
	}

	respStr := ""
	for _, s := range resp.Result {
		respStr = respStr + string(s)
	}

	json.Unmarshal(resp.Result, &rcpt)

	block, err := r.GetBlockByHash(rcpt.BlockHash)
	if err != nil {
		return &rcpt, err
	}

	rcpt.BlockHeight = block.Height

	return &rcpt, nil
}
