package payouts

import (
	"errors"
	"fmt"
	"time"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

func unlockBlocks(poolID string, rpcManager map[string]*rpc.Manager) (persistence.FoundBlocks, error) {
	pending, err := persistence.Blocks.PendingBlocksForPool(poolID)
	if err != nil {
		return nil, err
	}

	// Get chain based on block type
	blocks, err := classifyBlocks(pending, rpcManager)
	if err != nil {
		return nil, err
	}

	return calculateBlockEffort(blocks, poolID)
}

// TODO - This is very bitcoin/chain specific
// We eventually have to let the chain package consume the RPC package, and handle all chain related logic there.
// ^ That will take care of a lot of TODOs related to seperation of concerns
func classifyBlocks(blocks persistence.FoundBlocks, rpcManagers map[string]*rpc.Manager) (persistence.FoundBlocks, error) {
	var rpcManager *rpc.Manager
	var exists bool
	for i, localBlock := range blocks {
		rpcManager, exists = rpcManagers[localBlock.Chain]
		if !exists {
			return nil, errors.New("unlocker failed to find node for: " + localBlock.Chain)
		}

		remoteBlock, err := rpcManager.GetActiveClient().GetBlockByHash(localBlock.Hash)
		if err != nil {
			m := "unlocker failed to find remote block for %v block %v, %v"
			m = fmt.Sprintf(m, localBlock.Chain, localBlock.BlockHeight, localBlock.Hash)
			context := errors.New(m)
			err = errors.Join(context, err)
			return nil, err
		}

		if len(remoteBlock.Transactions) < 1 {
			m := "unlocker failed to fetch transaction confirmation for %v block %v, %v"
			m = fmt.Sprintf(m, localBlock.Chain, localBlock.BlockHeight, localBlock.Hash)
			return nil, errors.New(m)
		}

		remoteCoinbaseTransactionHash := remoteBlock.Transactions[0]
		remoteCoinbaseTransactionHash, err = reverseHexBytes(remoteCoinbaseTransactionHash)
		if err != nil {
			return nil, err
		}
		if localBlock.TransactionConfirmationData != "" {
			if localBlock.TransactionConfirmationData != remoteCoinbaseTransactionHash {
				// Likely an orphan
				m := "⚠️  Our confirmation data for %v height %v does not match the blockchains: (local) %v <> (remote) %v"
				m = fmt.Sprintf(m, localBlock.Chain, localBlock.BlockHeight, localBlock.TransactionConfirmationData, remoteCoinbaseTransactionHash)
				return nil, errors.New(m)
			}
		} else { // Aux blocks do not return coinbase data
			localBlock.TransactionConfirmationData = remoteCoinbaseTransactionHash
		}

		localConfirmationDataLittleEndian, err := reverseHexBytes(localBlock.TransactionConfirmationData)
		if err != nil {
			return nil, err
		}
		coinbaseTransaction, err := rpcManager.GetActiveClient().GetTransaction(localConfirmationDataLittleEndian)
		if err != nil {
			m := "%v Block %v: (confirmation) %v"
			m = fmt.Sprintf(m, localBlock.Chain, localBlock.BlockHeight, localBlock.TransactionConfirmationData)
			context := errors.New(m)
			return nil, errors.Join(context, err)
		}

		switch coinbaseTransaction.Details[0].Category {
		case "immature":
			min := bitcoin.GetChain(localBlock.Chain).MinimumConfirmations()
			blocks[i].ConfirmationProgress = float32(coinbaseTransaction.Confirmations) / float32(min)
			blocks[i].ConfirmationProgress = roundToThreeDigits(blocks[i].ConfirmationProgress)
			blocks[i].Reward = coinbaseTransaction.Amount
		case "generate":
			blocks[i].Status = persistence.StatusConfirmed
			blocks[i].ConfirmationProgress = 1
			blocks[i].Reward = coinbaseTransaction.Amount
		default:
			blocks[i].Status = persistence.StatusOrphaned
			blocks[i].Reward = 0
		}
	}

	return blocks, nil
}

func calculateBlockEffort(blocks persistence.FoundBlocks, poolID string) (persistence.FoundBlocks, error) {
	from, to := time.Time{}, time.Time{}
	statuses := []string{
		persistence.StatusConfirmed,
		persistence.StatusOrphaned,
		persistence.StatusPending,
	}
	for i, block := range blocks {
		to = block.Created
		lastBlock, err := persistence.Blocks.BlockBefore(poolID, statuses, block.Created)
		if err != nil {
			return nil, errors.New("blockEffort BlockBefore: " + err.Error())
		}
		if lastBlock != nil {
			from = lastBlock.Created
		}

		blocks[i].Effort, err = persistence.Shares.GetEffectiveAccumulatedShareDifficultyBetween(poolID, from, to)
		if err != nil {
			return nil, errors.New("blockEffort GetEffectiveAccumulatedShareDifficultyBetween: " + err.Error())
		}
		// TODO - there's a chain.AdjustEffort() interface method to make.
	}

	return blocks, nil
}
