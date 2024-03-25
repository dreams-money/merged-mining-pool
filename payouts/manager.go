package payouts

import (
	"log"
	"time"

	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

func RunManager(config *config.Config, rpcManagers map[string]*rpc.Manager, interval time.Duration) {
	var blocks persistence.FoundBlocks
	var err error
	var cutoffTime time.Time
	for {
		time.Sleep(interval)

		log.Println("Checking block confirmations")

		// Unlock Loop
		blocks, err = unlockBlocks(config.PoolName, rpcManagers)
		if err != nil {
			log.Println(err)
			continue
		}

		// Calculate Rewards loop
		for _, confirmed := range blocks.GetConfirmed() {
			rpcManager, exists := rpcManagers[confirmed.Chain]
			if !exists {
				panic("payouts.Manager: Blockchain not found - " + confirmed.Chain)
			}

			cutoffTime, err = calculateBlockRewards(confirmed, config, rpcManager)
			if err != nil {
				log.Println(err)
				continue
			}

			// Clear old shares
			err = persistence.Shares.DeleteSharesBefore(config.PoolName, cutoffTime)
			if err != nil {
				log.Println("payouts.DeleteShares(): " + err.Error())
			}
		}

		// Save block updates
		for _, block := range blocks {
			err = persistence.Blocks.Update(block)
			if err != nil {
				log.Println(err)
			}
		}

		// Actual payouts
		err = payoutBalances(config, rpcManagers)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
