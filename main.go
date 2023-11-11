package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"designs.capital/dogepool/api"
	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/pool"
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

	startPoolServer(configuration)
	startStatManager(configuration)
	startAPIServer(configuration)
	startStatsService(configuration)
	// startPayoutService(configuration *config.Config)

	// blocker := make(chan struct{})
	// <-blocker
}

func parseCommandLineOptions() string {
	flag.Parse()
	return flag.Arg(0)
}

func startPoolServer(configuration *config.Config) {
	poolServer := pool.NewServer(configuration)
	go poolServer.Start()
	log.Println("Started Pool on port: " + configuration.Port)
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

func startPayoutService(configuration *config.Config) {

}

func startStatsService(configuration *config.Config) {
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

func mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}
