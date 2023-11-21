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
	for rigID, averages := range rigs {
		for _, average := range averages {
			stat := hourStats[average.Created.Hour()]
			stat.HashRate.Raw = stat.HashRate.Raw + average.AverageHashrate
			stat.SharesPerSecond = stat.SharesPerSecond + average.AverageSharesPerSecond
			stat.Created = average.Created
			if stat.Workers == nil {
				stat.Workers = make(map[string]WorkerStat)
			}
			stat.Workers[rigID] = WorkerStat{
				HashRate:        floatToHashrate(average.AverageHashrate),
				SharesPerSecond: average.AverageSharesPerSecond,
			}

			hourStats[average.Created.Hour()] = stat
		}
	}

	// Find the non empty stat
	statDate := time.Time{}
	statDenomination := ""
	for _, stat := range hourStats {
		if !stat.Created.IsZero() {
			statDate = stat.Created
			statDenomination = floatToHashrate(stat.HashRate.Raw).Denomination
			break
		}
	}

	var statsNoZeros []HourStat
	for i, stat := range hourStats {
		if stat.Created.IsZero() {
			statsNoZeros = append(statsNoZeros, HourStat{
				Created: time.Date(statDate.Year(), statDate.Month(), statDate.Day(), i, 0, 0, 0, statDate.Location()),
				HashRate: HashRate{
					Raw:          0,
					Denomination: statDenomination,
					Rate:         "0",
				},
			})
		} else {
			statsNoZeros = append(statsNoZeros, HourStat{
				Created:         stat.Created,
				SharesPerSecond: stat.SharesPerSecond,
				HashRate:        floatToHashrate(stat.HashRate.Raw),
			})
		}
	}

	newDaySplit := 0
	previousDate := statsNoZeros[0].Created
	for i, l := 1, len(statsNoZeros); i < l; i++ {
		currentDate := statsNoZeros[i].Created
		if previousDate.Day() != currentDate.Day() {
			newDaySplit = i
			break
		}
		previousDate = currentDate
	}

	statsWithCorrectOrderedTimeStamps := append(statsNoZeros[newDaySplit:], statsNoZeros[0:newDaySplit]...)

	return statsWithCorrectOrderedTimeStamps
}
