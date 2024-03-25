package payouts

import (
	"errors"
	"fmt"
	"log"
	"time"

	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

func calculateBlockRewards(confirmed persistence.Found, config *config.Config, rpcManager *rpc.Manager) (time.Time, error) {
	remainingReward, err := calculatePoolReward(confirmed, config, rpcManager)
	if err != nil {
		return time.Time{}, err
	}

	return calculateMinerRewards(remainingReward, confirmed, config)
}

func calculatePoolReward(confirmed persistence.Found, config *config.Config, rpcManager *rpc.Manager) (float64, error) {
	remainingReward := confirmed.Reward
	payoutConfig, exists := config.Payouts.Chains[confirmed.Chain]
	if !exists {
		return 0, errors.New("calculatePoolReward(): failed to find payout config for: " + confirmed.Chain)
	}

	for _, poolRecipient := range payoutConfig.PoolRewardRecipients {
		recipientAmount := poolRecipient.Percentage * confirmed.Reward
		remainingReward -= recipientAmount

		chain, exists := config.BlockchainNodes[confirmed.Chain]
		if !exists {
			return 0, errors.New("calculatePoolReward(): failed to get blockchain node for : " + confirmed.Chain)
		}
		node := chain[rpcManager.GetIndex()]
		if poolRecipient.Address == node.RewardTo { // The block chain reward address is the same as the pool reward address
			continue
		}

		log.Printf("Crediting %v with %v %v", poolRecipient.Address, recipientAmount, confirmed.Chain)
		usage := "Reward for block %v"
		usage = fmt.Sprintf(usage, confirmed.BlockHeight)
		err := persistence.Balances.AddAmount(config.PoolName, confirmed.Chain, poolRecipient.Address, usage, recipientAmount)
		if err != nil {
			return 0, err
		}
	}

	return remainingReward, nil
}

func calculateMinerRewards(remainingReward float64, confirmed persistence.Found, config *config.Config) (time.Time, error) {
	payoutSchemeName := config.Payouts.Scheme
	payoutScheme := payoutSchemeFactory(payoutSchemeName, config)
	return payoutScheme.UpdateMinerBalances(config.PoolName, remainingReward, confirmed)
}
