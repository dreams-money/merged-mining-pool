package api

import (
	"log"
	"time"

	"designs.capital/dogepool/persistence"
)

func getDashboardStats(poolId, minerId string, chains []string) map[string]any {
	if minerId == "" {
		return map[string]any{}
	}

	report, err := persistence.Miners.GetMinerStatsReport(poolId, minerId, &persistence.Payments)
	if report == nil {
		return map[string]any{}
	}
	logOnError(err)
	hashrateFloat := float64(0)
	for _, stat := range report.Workers {
		hashrateFloat += stat.Hashrate
	}

	active, inactive := getWorkerCounts(poolId, minerId)

	balances := report.ChainAccounts.GetPendingAmounts()
	balances = padZeros(balances, chains)

	return map[string]any{
		"Balances":        balances,
		"CurrentHashrate": floatToHashrate(hashrateFloat),
		"ActiveWorkers":   active,
		"InactiveWorkers": inactive,
		"WorkerList":      minerWorkers(poolId, minerId),
	}
}

func getWorkerCounts(poolId, minerId string) (uint, uint) {
	report, err := persistence.Miners.GetWorkersLastSeen(poolId, minerId)
	logOnError(err)

	var active, inactive uint
	for _, worker := range report.Workers {
		if worker.Status == "Active" {
			active++
		} else if worker.Status == "Inactive" {
			inactive++
		} else {
			log.Println("getWorkerCounts: Unknown status: " + worker.Status)
		}
	}

	return active, inactive
}

type Worker struct {
	RigID           string
	Status          string
	CurrentHashrate HashRate
	Rating          int16
	LastSeen        time.Time
}

func minerWorkers(poolId, minerId string) []Worker {
	minerStats, err := persistence.Miners.GetMinerStatsReport(poolId, minerId, &persistence.Payments)
	logOnError(err)
	workerHashRates := minerStats.WorkersReport.Workers
	workerStats, err := persistence.Miners.GetWorkersLastSeen(poolId, minerId)
	logOnError(err)
	var workers []Worker
	for rigID, worker := range workerStats.Workers {
		stat, exists := workerHashRates[rigID]
		if !exists {
			stat = persistence.WorkerStat{}
		}
		workers = append(workers, Worker{
			RigID:           rigID,
			Status:          worker.Status,
			CurrentHashrate: floatToHashrate(stat.Hashrate),
			// Rating:          0,
			LastSeen: worker.LastSeen,
		})

	}

	return workers
}

func padZeros(balances map[string]float32, chains []string) map[string]float32 {
	for _, chain := range chains {
		_, exists := balances[chain]
		if !exists {
			balances[chain] = 0
		}
	}
	return balances
}
