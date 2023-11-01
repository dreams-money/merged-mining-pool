package persistence

import (
	"log"
	"math"
	"time"
)

func UpdateStatsOnInterval(poolID string, hashRateCalculationWindow, interval time.Duration) {
	var err error
	for {
		time.Sleep(interval)

		err = insertManyNewMinerStatsAndOnePoolStat(poolID, hashRateCalculationWindow)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("Successfully saved stats")
		}
	}
}

func insertManyNewMinerStatsAndOnePoolStat(poolID string, hashRateCalculationWindow time.Duration) error {
	now := time.Now()
	timeFrom := time.Now().Add(-hashRateCalculationWindow)

	workers, err := Shares.GetWorkerHashAccumulationBetween(poolID, timeFrom, now)
	if err != nil {
		return err
	}
	miners := workers.GroupByMiner()

	err = makeNewPoolStat(poolID, hashRateCalculationWindow, workers, uint(len(miners)), now)
	if err != nil {
		log.Println(err)
	}

	// Connect miner address back to rigID

	makeMinerStats(poolID, miners, now, timeFrom, hashRateCalculationWindow)

	// Do Orphans

	// previousMinerWorkerHashrates, err := Pool.MinerWorkerHashrates(poolID)
	// if err != nil {
	// 	return err
	// }
	// // Get "non zeros"

	return nil
}

func makeNewPoolStat(poolID string, hashRateCalculationWindow time.Duration, workers MinerWorkerHashSummaryResultSet, minerCount uint, now time.Time) error {
	poolStat := PoolStat{
		PoolID:  poolID,
		Created: now,
	}

	if workers != nil {
		poolStat.ConnectedMiners = minerCount
		poolStat.ConnectedWorkers = uint(len(workers))
		poolStat.PoolHashrate, poolStat.SharesPerSecond = getHashrateAndSharesPerSecond(workers, hashRateCalculationWindow)
	} else {
		poolStat.ConnectedMiners, poolStat.ConnectedWorkers, poolStat.PoolHashrate, poolStat.SharesPerSecond = 0, 0, 0, 0
	}

	return Pool.InsertPoolStat(poolStat)
}

func makeMinerStats(poolID string, miners map[string][]MinerWorkerHashSummary, now, timeFrom time.Time, hashRateCalculationWindow time.Duration) int {
	minerStat := MinerStat{
		PoolID:  poolID,
		Created: now,
	}
	var err error
	var successCount int
	for miner, workers := range miners {
		window := getStatsCalcWindows(now, timeFrom, hashRateCalculationWindow, workers)

		// Adjust hash window if it's not full
		minerHashTimeFrame := hashRateCalculationWindow
		if window.timeFrameBeforeFirstShare >= (hashRateCalculationWindow / 10) {
			minerHashTimeFrame = hashRateCalculationWindow - window.timeFrameBeforeFirstShare
		}
		if window.timeFrameAfterLastShare >= (hashRateCalculationWindow / 10) {
			minerHashTimeFrame = hashRateCalculationWindow + window.timeFrameAfterLastShare
		}
		if (window.timeFrameBeforeFirstShare >= (hashRateCalculationWindow / 10)) && (window.timeFrameAfterLastShare >= (hashRateCalculationWindow / 10)) {
			minerHashTimeFrame = hashRateCalculationWindow - window.timeFrameBeforeFirstShare + window.timeFrameAfterLastShare
		}

		if minerHashTimeFrame < 1 {
			minerHashTimeFrame = 1
		}

		roundToThreeDigits := func(x float64) float64 {
			return math.Round(x*1000) / 1000
		}
		for _, worker := range workers {
			minerStat.Miner = miner
			minerStat.Worker = worker.Worker
			minerStat.Hashrate = hashrateFromShares(worker.Difficulty, minerHashTimeFrame)
			minerStat.SharesPerSecond = roundToThreeDigits(float64(time.Duration(worker.ShareCount) / minerHashTimeFrame))

			err = Miners.InsertMinerWorkerPerformanceStats(minerStat)
			if err != nil {
				log.Println(err)
			} else {
				successCount++
			}
		}
	}

	return successCount
}

func getHashrateAndSharesPerSecond(hashSummaries MinerWorkerHashSummaryResultSet, hashRateCalculationWindow time.Duration) (float64, float64) {
	sumShares, sharesPerSecond := float64(0), float64(0)
	for _, summary := range hashSummaries {
		sumShares += summary.Difficulty
		sharesPerSecond += float64(summary.ShareCount)
	}
	hashRate := hashrateFromShares(sumShares, hashRateCalculationWindow)
	return math.Floor(hashRate), sharesPerSecond / float64(hashRateCalculationWindow)
}

type StatsCalcWindow struct {
	timeFrameBeforeFirstShare time.Duration
	timeFrameAfterLastShare   time.Duration
	timeFrameFirstLastShare   time.Duration
}

func getStatsCalcWindows(now, timeFrom time.Time, hashRateCalculationWindow time.Duration, workers []MinerWorkerHashSummary) StatsCalcWindow {
	var first, last time.Time
	first = workers[0].FirstShare
	for _, worker := range workers {
		if first.After(worker.FirstShare) {
			first = worker.FirstShare
		}
		if last.Before(worker.LastShare) {
			last = worker.FirstShare
		}
	}

	window := StatsCalcWindow{
		timeFrameBeforeFirstShare: first.Sub(timeFrom),
		timeFrameAfterLastShare:   now.Sub(last),
	}
	window.timeFrameFirstLastShare = hashRateCalculationWindow - window.timeFrameBeforeFirstShare - window.timeFrameAfterLastShare

	return window
}

func hashrateFromShares(shareSum float64, interval time.Duration) float64 {
	bitcoinConstantMultiplier := math.Pow(2, 32)
	hashrate := shareSum * bitcoinConstantMultiplier / interval.Seconds()

	// Possible "hashrate multiplier" depending on coin
	return hashrate

}
