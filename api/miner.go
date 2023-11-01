package api

import (
	"time"

	"designs.capital/dogepool/persistence"
)

type WorkerStat struct {
	Hashrate        float64
	SharesPerSecond float64
}

type HourStat struct {
	Created         string
	Hashrate        float64
	SharesPerSecond float64
	Workers         map[string]WorkerStat
}

func getMinerHistory(poolId, minerId string) []HourStat {
	minerRepo := persistence.Miners
	oneDayAgo := time.Now().Add(-1 * time.Hour * 24)
	now := time.Now()
	performance, err := minerRepo.GetMinerPerformaceBetweenTimeAtXMinuteIntervals(poolId, minerId, oneDayAgo, now, time.Hour)
	logOnError(err)
	if performance == nil {
		return nil
	}

	var hourStats []HourStat
	for rigID, stat := range performance.Workers {
		hourStat := HourStat{
			Created:         stat.Created.Format(JavascriptISOFormat),
			Hashrate:        0,
			SharesPerSecond: 0,
		}
		hourStat.Workers = make(map[string]WorkerStat)
		hourStat.Workers[rigID] = WorkerStat{
			Hashrate:        stat.Hashrate,
			SharesPerSecond: stat.SharesPerSecond,
		}

		hourStats = append(hourStats, hourStat)
	}

	return hourStats
}
