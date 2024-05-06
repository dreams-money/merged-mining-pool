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

	makeMinerStats(poolID, miners, now, timeFrom, hashRateCalculationWindow)

	return nil
}

func makeNewPoolStat(poolID string, hashRateCalculationWindow time.Duration, workers MinerWorkerHashAccumulationResultSet, minerCount uint, now time.Time) error {
	poolStat := PoolStat{
		PoolID:  poolID,
		Created: now,
	}

	if workers != nil {
		poolStat.ConnectedMiners = minerCount
		poolStat.ConnectedWorkers = uint(len(workers))
		poolStat.PoolHashrate, poolStat.SharesPerSecond = getHashrateAndSharesPerSecond(workers, hashRateCalculationWindow)
		poolStat.PoolHashrate, poolStat.SharesPerSecond = math.Floor(poolStat.PoolHashrate), roundToThreeDigits(poolStat.SharesPerSecond)
	} else {
		poolStat.ConnectedMiners, poolStat.ConnectedWorkers, poolStat.PoolHashrate, poolStat.SharesPerSecond = 0, 0, 0, 0
	}

	return Pool.InsertPoolStat(poolStat)
}

func makeMinerStats(poolID string, miners map[string][]MinerWorkerHashAccumulation, now, timeFrom time.Time, hashRateCalculationWindow time.Duration) int {
	minerStat := MinerStat{
		PoolID:  poolID,
		Created: now,
	}
	var err error
	var successCount int
	for miner, workers := range miners {
		differences := getWindowDifferences(now, timeFrom, workers)

		// Adjust hash window if it's not full
		adjustedWindow := adjustHashWindow(differences, hashRateCalculationWindow)

		for _, worker := range workers {
			minerStat.Miner = miner
			minerStat.Worker = worker.Worker
			minerStat.Hashrate = math.Floor(hashrateFromShares(worker.SumDifficulty, adjustedWindow))

			sharesPerSecond := float64(worker.ShareCount) / adjustedWindow
			minerStat.SharesPerSecond = roundToThreeDigits(sharesPerSecond)

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

func adjustHashWindow(window StatsCalcWindow, hashRateCalculationWindow time.Duration) float64 {
	minerHashTimeFrame := hashRateCalculationWindow.Seconds()
	statsStartedAfterHashrateWindow := window.startDifference.Seconds() >= (hashRateCalculationWindow.Seconds() * 0.1)
	statsEndedBeforeHashrateWindow := window.endDifference.Seconds() >= (hashRateCalculationWindow.Seconds() * 0.1)

	if statsStartedAfterHashrateWindow && statsEndedBeforeHashrateWindow {
		minerHashTimeFrame = hashRateCalculationWindow.Seconds() - window.startDifference.Seconds() + window.endDifference.Seconds()
	} else if statsStartedAfterHashrateWindow {
		minerHashTimeFrame = hashRateCalculationWindow.Seconds() - window.startDifference.Seconds()
	} else if statsEndedBeforeHashrateWindow {
		minerHashTimeFrame = hashRateCalculationWindow.Seconds() + window.endDifference.Seconds()
	}

	minerHashTimeFrame = math.Floor(minerHashTimeFrame)

	if minerHashTimeFrame < 1 {
		minerHashTimeFrame = 1
	}

	return minerHashTimeFrame
}

func getHashrateAndSharesPerSecond(hashSummaries MinerWorkerHashAccumulationResultSet, hashRateCalculationWindow time.Duration) (float64, float64) {
	sumShares, sharesPerSecond := float64(0), float64(0)
	for _, summary := range hashSummaries {
		sumShares += summary.SumDifficulty
		sharesPerSecond += float64(summary.ShareCount)
	}
	hashRate := hashrateFromShares(sumShares, hashRateCalculationWindow.Seconds())

	return math.Floor(hashRate), sharesPerSecond / float64(hashRateCalculationWindow)
}

type StatsCalcWindow struct {
	startDifference time.Duration
	endDifference   time.Duration
}

func getWindowDifferences(now, timeFrom time.Time, workers []MinerWorkerHashAccumulation) StatsCalcWindow {
	// See README.md for illustration
	var first, last time.Time
	first = workers[0].FirstShare
	for _, worker := range workers {
		if first.After(worker.FirstShare) {
			first = worker.FirstShare
		}
		if last.Before(worker.LastShare) {
			last = worker.LastShare
		}
	}

	window := StatsCalcWindow{
		startDifference: first.Sub(timeFrom),
		endDifference:   now.Sub(last),
	}

	return window
}

func hashrateFromShares(shareSum, interval float64) float64 {
	// TODO move to coin package
	bitcoinConstantMultiplier := math.Pow(2, 32)
	hashrate := shareSum * bitcoinConstantMultiplier / interval

	return hashrate
}

func roundToThreeDigits(x float64) float64 {
	return math.Round(x*1000) / 1000
}
