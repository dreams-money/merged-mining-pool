package api

import (
	"time"

	"designs.capital/dogepool/persistence"
)

type WorkerStat struct {
	HashRate
	SharesPerSecond float64
}

type HourStat struct {
	Created time.Time
	HashRate
	SharesPerSecond float64
	Workers         map[string]WorkerStat
}

func getMinerHistory(poolId, minerId string) []HourStat {
	minerRepo := persistence.Miners
	oneDayAgo := time.Now().Add(-1 * time.Hour * 24)
	now := time.Now()
	averages, err := minerRepo.GetMinerHourlyAveragesBetween(poolId, minerId, oneDayAgo, now)
	logOnError(err)
	if averages == nil {
		return nil
	}

	hourStats := make(map[int]HourStat)
	for rigID, average := range averages {
		stat, exists := hourStats[average.Created.Hour()]
		if !exists {
			stat = HourStat{}
			stat.Workers = make(map[string]WorkerStat)
		}

		stat.HashRate.Raw = stat.HashRate.Raw + average.AverageHashrate
		stat.SharesPerSecond = stat.SharesPerSecond + average.AverageSharesPerSecond
		stat.Created = average.Created
		stat.Workers[rigID] = WorkerStat{
			HashRate:        floatToHashrate(average.AverageHashrate),
			SharesPerSecond: average.AverageSharesPerSecond,
		}

		hourStats[average.Created.Hour()] = stat
	}

	statsNoKey := make([]HourStat, 0, len(hourStats))
	for _, stat := range hourStats {
		stat.HashRate = floatToHashrate(stat.HashRate.Raw)
		statsNoKey = append(statsNoKey, stat)
	}

	return statsNoKey
}
