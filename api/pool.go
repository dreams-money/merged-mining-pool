package api

import (
	"log"

	"designs.capital/dogepool/persistence"
)

func getPoolIndex(poolID string) map[string]any {
	stat, err := persistence.Pool.GetLastStat(poolID)
	logOnError(err)

	return map[string]any{
		"PoolHashRate":  stat.PoolHashrate,
		"ActiveMiners":  stat.ConnectedMiners,
		"Workers":       stat.ConnectedWorkers,
		"BlocksPerHour": blocksPerHour(poolID),
		"LatestBlocks":  recentBlocks(poolID),
	}
}

func blocksPerHour(poolID string) uint {
	rate, err := persistence.Blocks.PoolBlocksPerHour(poolID)
	logOnError(err)
	return rate
}

func recentBlocks(poolId string) []persistence.Found {
	found, err := persistence.Blocks.PageBlocks(poolId, persistence.StatusConfirmed, 0, 100)
	logOnError(err)
	return found
}

func logOnError(err error) {
	if err != nil {
		log.Println(err)
	}
}
