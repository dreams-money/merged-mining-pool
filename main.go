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
	// startPayoutService(configuration *config.Config)
	startStatsService(configuration)

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
	// TODO take out intervals to config
	hashrateWindow := time.Duration(10)
	statsRecordInterval := time.Duration(15)
	go persistence.UpdateStatsOnInterval(configuration.PoolName, time.Minute*hashrateWindow, time.Second*statsRecordInterval)
	log.Printf("Stat Manager running every %v seconds with a hashrate window of %v minutes\n", statsRecordInterval, hashrateWindow)
}

func startStatsService(configuration *config.Config) {
	for {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		fmt.Println("STATS START")
		log.Printf("Total CPU cores avaliable: %v", runtime.NumCPU())
		log.Printf("Total Goroutines: %v", runtime.NumGoroutine())
		log.Printf("Total System Memory: %v", memStats.Sys)
		log.Printf("Total Memory Allocated: %v", memStats.TotalAlloc)
		fmt.Println("STATS END")
		time.Sleep(time.Second * 10)
	}
}

func startPayoutService(configuration *config.Config) {

}
