package payouts

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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

	// Send payments
	transactionConfirmation, err := bitcoinTryManyPayments(balances, config, rpcManagers)
	if err != nil {
		return err
	}

	for _, balance := range balances {
		confirmation, found := transactionConfirmation[balance.Chain]
		if !found {
			m := "failed to find payment confirmation for %v on chain %v"
			m = fmt.Sprintf(balance.Address, balance.Chain)
			return errors.New(m)
		}

		// Record Payments
		address, err := findBalanceAddress(balance, config)
		if err != nil {
			return err
		}

		err = persistence.Payments.Insert(persistence.Payment{
			PoolID:                      balance.PoolID,
			Chain:                       balance.Chain,
			Address:                     address,
			Amount:                      balance.Amount,
			Created:                     time.Now(),
			TransactionConfirmationData: confirmation,
		})
		if err != nil {
			return err
		}

		// Reset Balance
		usage := "Paid balance to miner"
		err = persistence.Balances.AddAmount(config.PoolName, balance.Chain, balance.Address, usage, balance.Amount*-1)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO move to bitcoin aka the chain package.
func bitcoinTryManyPayments(balances []persistence.Balance, config *config.Config, rpcManagers map[string]*rpc.Manager) (map[string]string, error) {
	transactionsGroupedByChain := make(map[string]map[string]float64)
	transactionConfirmationByChain := make(map[string]string)

	for _, balance := range balances {
		chainBalances, exists := transactionsGroupedByChain[balance.Chain]
		if !exists {
			chainBalances = make(map[string]float64)
		}

		address, err := findBalanceAddress(balance, config)
		if err != nil {
			return transactionConfirmationByChain, err
		}

		chainBalances[address] = balance.Amount
		transactionsGroupedByChain[balance.Chain] = chainBalances
	}

	for chain, transactions := range transactionsGroupedByChain {
		client, exists := rpcManagers[chain]
		if !exists {
			return transactionConfirmationByChain, errors.New("payouts.bitcoinTryManyPayments() - failed to find chain rpc: " + chain)
		}
		node := client.GetActiveClient()
		transactionID, err := node.SendMany(transactions)
		if err != nil {
			m := "failed to send %v payments"
			m = fmt.Sprintf(m, chain)
			context := errors.New(m)
			err = errors.Join(context, err)
			return transactionConfirmationByChain, err
		}

		transactionConfirmationByChain[chain] = transactionID

		log.Printf("%v Payouts Transaction ID: %v\n", chain, transactionID)
	}

	return transactionConfirmationByChain, nil
}

// TODO - move this to REWARDS?
func findBalanceAddress(balance persistence.Balance, config *config.Config) (string, error) {
	mergedMining := len(config.BlockChainOrder) > 1
	address := balance.Address
	if mergedMining {
		addresses := strings.Split(balance.Address, "-")

		if len(addresses) == 1 { // Pool reward address
			return addresses[0], nil
		}

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
			return "", errors.New("chain address not found: " + balance.Chain)
		}

		address = addresses[i]
	}

	return address, nil
}
