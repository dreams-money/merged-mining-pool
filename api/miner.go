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
	nonZeroStatIndex := 0
	for i, stat := range hourStats {
		if !stat.Created.IsZero() {
			statDate = stat.Created
			statDenomination = floatToHashrate(stat.HashRate.Raw).Denomination
			nonZeroStatIndex = i
			break
		}
	}

	var statsNoZeros []HourStat
	l := len(hourStats)

	for i, stat := range hourStats {
		if stat.Created.IsZero() {
			statNoZero := HourStat{
				Created: time.Date(statDate.Year(), statDate.Month(), statDate.Day()-1, i, 0, 0, 0, statDate.Location()),
				HashRate: HashRate{
					Raw:          0,
					Denomination: statDenomination,
					Rate:         "0",
				},
			}

			if i > nonZeroStatIndex {
				for j := i; j < l; j++ {
					forwardStat := hourStats[j]
					if !forwardStat.Created.IsZero() {
						statNoZero.Created.AddDate(0, 0, 1)
						break
					}
				}
			}

			statsNoZeros = append(statsNoZeros, statNoZero)
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
