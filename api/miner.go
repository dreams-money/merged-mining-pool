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
	rigs, err := minerRepo.GetMinerHourlyAveragesBetween(poolId, minerId, oneDayAgo, now)
	logOnError(err)
	if rigs == nil {
		return nil
	}

	hourStats := make([]HourStat, 24)
	now = now.UTC()

	// Start with a 0 stat array
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	for i := 0; i < 24; i++ {
		stat := hourStats[i]
		stat.HashRate.Raw = 0
		stat.SharesPerSecond = 0
		stat.Created = now.Add(-1 * (time.Hour * time.Duration(i)))
		stat.Workers = make(map[string]WorkerStat)
		hourStats[i] = stat
	}

	// Next, fill the stats that exist
	for rigID, averages := range rigs {
		for _, average := range averages {
			difference := now.Sub(average.Created)
			i := int(difference.Hours())
			if i > 23 { // Ignore older stats
				continue
			}
			stat := hourStats[i]
			stat.HashRate.Raw = stat.HashRate.Raw + average.AverageHashrate
			stat.SharesPerSecond = stat.SharesPerSecond + average.AverageSharesPerSecond
			stat.Created = average.Created
			stat.Workers[rigID] = WorkerStat{
				HashRate:        floatToHashrate(average.AverageHashrate),
				SharesPerSecond: average.AverageSharesPerSecond,
			}

			hourStats[i] = stat
		}
	}

	// Now put it into appropriate order
	for i, j := 0, len(hourStats)-1; i < j; i, j = i+1, j-1 {
		hourStats[i], hourStats[j] = hourStats[j], hourStats[i]
	}

	return hourStats
}
