package api

import (
	"log"
	"strings"
	"time"

	"designs.capital/dogepool/persistence"
)

func getPoolIndex(poolID string, chains []string) map[string]any {
	stat, err := persistence.Pool.GetLastStat(poolID)
	logOnError(err)

	return map[string]any{
		"PoolHashRate":  floatToHashrate(stat.PoolHashrate),
		"ActiveMiners":  stat.ConnectedMiners,
		"Workers":       stat.ConnectedWorkers,
		"BlocksPerHour": blocksPerHour(poolID),
		"LatestBlocks":  recentBlocks(poolID, chains),
	}
}

func blocksPerHour(poolID string) uint {
	rate, err := persistence.Blocks.PoolBlocksPerHour(poolID)
	logOnError(err)
	return rate
}

type Block struct {
	Chain      string `json:"chain"`
	Height     int    `json:"blockHeight"`
	HashHeader string `json:"hash"`
	Created    string `json:"created"`
	MinutesAgo int
}

func recentBlocks(poolId string, chains []string) []Block {
	var allChains []persistence.Found
	for _, chain := range chains {
		found, err := persistence.Blocks.PageBlocks(poolId, chain, []string{persistence.StatusConfirmed}, 0, 5)
		if err != nil {
			log.Println(err)
			continue
		}
		allChains = append(allChains, found...)
	}

	now := time.Now()
	recent := make([]Block, len(allChains))
	for i, block := range allChains {
		recent[i] = Block{
			Chain:      strings.ToUpper(block.Chain[:1]) + block.Chain[1:],
			Height:     int(block.BlockHeight),
			HashHeader: block.Hash,
			Created:    block.Created.String(),
			MinutesAgo: int(now.Sub(block.Created).Minutes()),
		}
	}

	return recent
}

func logOnError(err error) {
	if err != nil {
		log.Println(err)
	}
}
