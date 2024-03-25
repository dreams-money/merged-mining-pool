package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"designs.capital/dogepool/api"
	"designs.capital/dogepool/config"
	"designs.capital/dogepool/payouts"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/pool"
	"designs.capital/dogepool/rpc"
)

func main() {
	configFileName := parseCommandLineOptions()
	if configFileName == "" {
		configFileName = "config.json"
	}
	configuration := config.LoadConfig(configFileName)

	err := persistence.MakePersister(configuration)
	if err != nil {
		log.Fatal(err)
	}

	rpcManagers := makeRPCManagers(configuration)
	startPoolServer(configuration, rpcManagers)
	startStatManager(configuration)
	startAPIServer(configuration)
	startPayoutService(configuration, rpcManagers)
	startAppStatsService(configuration)
}

func parseCommandLineOptions() string {
	flag.Parse()
	return flag.Arg(0)
}

func startPoolServer(configuration *config.Config, managers map[string]*rpc.Manager) *pool.PoolServer {
	poolServer := pool.NewServer(configuration, managers)
	go poolServer.Start()
	log.Println("Started Pool on port: " + configuration.Port)
	return poolServer
}

func startAPIServer(configuration *config.Config) {
	go api.ListenAndServe(configuration)
	log.Println("Started API on port: " + configuration.API.Port)
}

func startStatManager(configuration *config.Config) {
	hashrateWindow := mustParseDuration(configuration.HashrateWindow)
	statsRecordInterval := mustParseDuration(configuration.PoolStatsInterval)
	go persistence.UpdateStatsOnInterval(configuration.PoolName, hashrateWindow, statsRecordInterval)
	log.Printf("Stat Manager running every %v with a hashrate window of %v\n", statsRecordInterval, hashrateWindow)
}

func startPayoutService(configuration *config.Config, manager map[string]*rpc.Manager) {
	interval := mustParseDuration(configuration.Payouts.Interval)
	go payouts.RunManager(configuration, manager, interval)
	log.Printf("Payouts manager running every %v\n", interval)
}

func startAppStatsService(configuration *config.Config) {
	interval := mustParseDuration(configuration.AppStatsInterval)
	for {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		fmt.Println("STATS START")
		log.Printf("Total CPU cores avaliable: %v", runtime.NumCPU())
		log.Printf("Total Goroutines: %v", runtime.NumGoroutine())
		log.Printf("Total System Memory: %v", memStats.Sys)
		log.Printf("Total Memory Allocated: %v", memStats.TotalAlloc)
		fmt.Println("STATS END")
		time.Sleep(interval)
	}
}

func makeRPCManagers(configuration *config.Config) map[string]*rpc.Manager {
	managers := make(map[string]*rpc.Manager)
	for _, chain := range configuration.BlockChainOrder {
		nodeConfigs := configuration.BlockchainNodes[chain]
		rpcConfig := make([]rpc.Config, len(nodeConfigs))
		for i, nodeConfig := range nodeConfigs {
			rpcConfig[i] = rpc.Config{
				Name:     nodeConfig.Name,
				URL:      nodeConfig.RPC_URL,
				Username: nodeConfig.RPC_Username,
				Password: nodeConfig.RPC_Password,
				Timeout:  nodeConfig.Timeout,
			}
		}
		// TODO move interval to config if accepted
		manager := rpc.MakeRPCManager(chain, rpcConfig, "1h")
		managers[chain] = &manager
	}
	return managers
}

func mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}
