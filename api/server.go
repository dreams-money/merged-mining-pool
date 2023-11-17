package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"designs.capital/dogepool/config"
)

const JavascriptISOFormat = "2006-01-02T15:04:05.999Z07:00"

func minerIndex(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, fmt.Sprintf("method %s is not allowed", request.Method), http.StatusMethodNotAllowed)
		return
	}
	minerId := request.URL.Query().Get("id")
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Access-Control-Allow-Origin", "*")
	err := json.NewEncoder(response).Encode(getDashboardStats(serverConfig.PoolName, minerId, serverConfig.BlockChainOrder))
	if err != nil {
		http.Error(response, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
	}
}

func minerHistory(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, fmt.Sprintf("method %s is not allowed", request.Method), http.StatusMethodNotAllowed)
		return
	}

	minerId := request.URL.Query().Get("id")
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Access-Control-Allow-Origin", "*")
	err := json.NewEncoder(response).Encode(getMinerHistory(serverConfig.PoolName, minerId))
	if err != nil {
		http.Error(response, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
	}
}

func poolIndex(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, fmt.Sprintf("method %s is not allowed", request.Method), http.StatusMethodNotAllowed)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Access-Control-Allow-Origin", "*")
	err := json.NewEncoder(response).Encode(getPoolIndex(serverConfig.PoolName, serverConfig.BlockChainOrder))
	if err != nil {
		http.Error(response, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
	}
}

var serverConfig *config.Config

func ListenAndServe(configuration *config.Config) {
	serverConfig = configuration

	http.HandleFunc("/miner", minerIndex)
	http.HandleFunc("/miner-history", minerHistory)
	http.HandleFunc("/pool", poolIndex)

	log.Fatal(http.ListenAndServe(":"+configuration.API.Port, nil))
}
