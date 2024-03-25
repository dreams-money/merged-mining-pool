package payouts

import (
	"errors"
	"log"
	"strings"

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

	mergedMining := len(config.BlockChainOrder) > 1

	for _, balance := range balances {
		chainBalances, exists := transactionsGroupedByChain[balance.Chain]
		if !exists {
			chainBalances = make(map[string]float32)
		}

		address := balance.Address

		// TODO - move this to REWARDS
		if mergedMining {
			addresses := strings.Split(address, "-")
			found := false
			i := 0
			for _, chain := range config.BlockChainOrder {
				if chain == balance.Chain {
					found = true
					break
				}
				i++
			}

			if !found {
				return errors.New("chain address not found: " + balance.Chain)
			}

			address = addresses[i]
		}

		chainBalances[address] = balance.Amount
		transactionsGroupedByChain[balance.Chain] = chainBalances
	}

	for chain, transactions := range transactionsGroupedByChain {
		client, exists := rpcManagers[chain]
		if !exists {
			return errors.New("payouts.bitcoinTryManyPayments() - failed to find chain rpc: " + chain)
		}
		node := client.GetActiveClient()
		transactionID, err := node.SendMany(transactions)
		if err != nil {
			return err
		}
		log.Printf("%v Payouts Transaction ID: %v\n", chain, transactionID)

	}

	return nil
}
