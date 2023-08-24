package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"designs.capital/dogepool/api"
	"designs.capital/dogepool/config"
	"designs.capital/dogepool/pool"
)

func main() {
	configFileName := parseCommandLineOptions()
	if configFileName == "" {
		configFileName = "config.json"
	}

	go startPoolServer(configFileName)
	go startAPIServer()
	// go startPayoutService()

	startStatsService()

	// blocker := make(chan struct{})
	// <-blocker
}

func parseCommandLineOptions() string {
	flag.Parse()
	return flag.Arg(0)
}

func startPoolServer(configFileName string) {
	poolServer := pool.NewServer(config.LoadConfig(configFileName))
	go poolServer.Start()
	log.Println("Started Pool")
}

func startAPIServer() {
	api.ListenAndServe()
	log.Println("Started API")
}

func startStatsService() {
	for {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		fmt.Println("STATS START")
		log.Printf("Total CPU cores avaliable: %v", runtime.NumCPU())
		log.Printf("Total Goroutines: %v", runtime.NumGoroutine())
		log.Printf("Total System Memory: %v", memStats.Sys)
		log.Printf("Total Memory Allocated: %v", memStats.TotalAlloc)
		fmt.Println("STATS END")
		time.Sleep(time.Minute * 5)
	}
}

func startPayoutService() {

}
