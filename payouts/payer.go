package payouts

import (
	"errors"
	"log"

	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

// The coin should handle it's own paying. TODO.
func payoutBalances(config *config.Config, rpcManagers map[string]*rpc.Manager) error {
	var balances []persistence.Balance
	for _, chain := range config.BlockChainOrder {
		payoutConfig, exists := config.Payouts.Chains[chain]
		if !exists {
			return errors.New("payouts.payoutBalances() - failed to find chain payout config: " + chain)
		}
		b, err := persistence.Balances.GetPoolBalancesOverThreshold(config.PoolName, chain, payoutConfig.MinerMinimumPayment)
		if err != nil {
			return err
		}
		balances = append(balances, b...)
	}

	return bitcoinTryManyPayments(balances, config, rpcManagers)
}

// TODO move to bitcoin aka the chain package.
func bitcoinTryManyPayments(balances []persistence.Balance, config *config.Config, rpcManagers map[string]*rpc.Manager) error {
	transactionsGroupedByChain := make(map[string]map[string]float32)
	for _, balance := range balances {
		transactionsGroupedByChain[balance.Chain][balance.Address] = balance.Amount
	}

	for chain, transactions := range transactionsGroupedByChain {
		client, exists := rpcManagers[chain]
		if !exists {
			return errors.New("payouts.bitcoinTryManyPayments() - failed to find chain rpc: " + chain)
		}
		node := client.GetActiveClient()
		config := config.Payouts.Chains[chain]
		transactionID, err := node.SendMany(transactions, config.RewardFrom)
		if err != nil {
			return err
		}
		log.Printf("%v Payouts Transaction ID: %v\n", chain, transactionID)

	}

	return nil
}
