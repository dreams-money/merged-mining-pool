package payouts

import (
	"errors"
	"fmt"
	"log"
	"time"

	"designs.capital/dogepool/persistence"
)

type PROP struct{}

func (PROP) UpdateMinerBalances(poolID string, blockReward float64, confirmed persistence.Found) (time.Time, error) {
	emptyTime, cutoffTime := time.Time{}, time.Time{}
	before := confirmed.Created
	minerShares, minerScores := make(map[string]float64), make(map[string]float64)
	inclusive := true
	currentPage := 0
	pageSize := 100000

	done := false
	accumlatedScore := float64(0)
	for !done {
		page, err := persistence.Shares.GetSharesBefore(poolID, before, inclusive, pageSize)
		if err != nil {
			return emptyTime, err
		}

		inclusive = false
		currentPage++

		log.Printf("PROP Payouts: paging through page %v of shares for %v block %v\n", currentPage, confirmed.Chain, confirmed.BlockHeight)

		for _, share := range page {
			// TODO: Adjust share difficulty if coin needs it.
			adjustedShare := share.Difficulty
			minerShares[share.Miner] += adjustedShare

			score := adjustedShare / share.NetworkDifficulty

			minerScores[share.Miner] += score

			accumlatedScore += score
			if cutoffTime.IsZero() || share.Created.After(cutoffTime) {
				cutoffTime = share.Created
			}
		}

		pageLength := len(page)
		if pageLength < pageSize {
			done = true
			break
		}

		before = page[pageLength-1].Created
	}

	rewardPerScorePoint := blockReward / accumlatedScore

	remainingReward := blockReward

	minerRewards := make(map[string]float64)
	for miner, score := range minerScores {
		reward := score * rewardPerScorePoint
		minerRewards[miner] += reward
		remainingReward -= reward
	}

	if remainingReward <= 0 {
		return emptyTime, errors.New("PROP payout overflow! - we awarded more than we have.  Awards not persisted")
	}

	var err error
	for miner, reward := range minerRewards {
		log.Printf("Awarding %v %v PROP reward to miner %v for work on %v block %v\n",
			reward, confirmed.Chain, confirmed.Miner, confirmed.Chain, confirmed.BlockHeight)

		usage := "PROP REWARD FOR BLOCK %v"
		usage = fmt.Sprintf(usage, confirmed.BlockHeight)
		err = persistence.Balances.AddAmount(poolID, confirmed.Chain, miner, usage, reward)
		if err != nil {
			return emptyTime, err
		}
	}

	return cutoffTime, nil
}
