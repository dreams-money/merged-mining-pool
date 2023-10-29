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
	log.Println("Started Pool")
}

func startAPIServer(configuration *config.Config) {
	go api.ListenAndServe()
	log.Println("Started API")
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
