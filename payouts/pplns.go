package payouts

import (
	"errors"
	"fmt"
	"log"
	"time"

	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
)

type PPLNS struct {
	config *config.Config
}

func (scheme PPLNS) UpdateMinerBalances(poolID string, blockReward float32, confirmed persistence.Found) (time.Time, error) {
	emptyTime, cutoffTime := time.Time{}, time.Time{}
	before := confirmed.Created
	inclusive := true
	currentPage := 0
	pageSize := 100000

	// TODO - move this to config
	// PPLNS window (see https://bitcointalk.org/index.php?topic=39832)
	NWindow := float64(2)

	done := false
	remainingReward := blockReward
	accumlatedScore := float64(0)
	minerRewards := make(map[string]float32)
	for !done {
		page, err := persistence.Shares.GetSharesBefore(poolID, before, inclusive, pageSize)
		if err != nil {
			return emptyTime, err
		}

		inclusive = false
		currentPage++

		log.Printf("PPLNS Payouts: paging through page %v of shares\n", currentPage)

		for _, share := range page {
			// TODO: Adjust share difficulty if coin needs it.
			adjustedShare := share.Difficulty

			score := adjustedShare / share.NetworkDifficulty

			if accumlatedScore+score >= NWindow {
				score = NWindow - accumlatedScore
				cutoffTime = share.Created
				done = true
			}

			reward := float32(score * float64(blockReward) / NWindow)
			minerRewards[share.Miner] += reward
			remainingReward -= reward

			if remainingReward <= 0 {
				return emptyTime, errors.New("PPLNS payout overflow! - we awarded more than we have.  Awards not persisted")
			}

		}

		pageLength := len(page)
		if pageLength < pageSize {
			done = true
			break
		}

		before = page[pageLength-1].Created
	}

	var err error
	for miner, reward := range minerRewards {
		log.Printf("Awarding %v %v PPLNS reward to miner %v\n", reward, confirmed.Chain, confirmed.Miner)

		usage := "PPLNS REWARD FOR BLOCK %v"
		usage = fmt.Sprintf(usage, confirmed.BlockHeight)
		err = persistence.Balances.AddAmount(poolID, confirmed.Chain, miner, usage, reward)
		if err != nil {
			context := errors.New("Failed to add balances: ")
			return emptyTime, errors.Join(context, err)
		}
	}

	return cutoffTime, nil
}
