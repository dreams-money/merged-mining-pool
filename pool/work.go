package pool

import (
	"errors"
	"fmt"
	"log"

	"designs.capital/dogepool/block"
	"designs.capital/dogepool/template"
)

type Work []interface{}

func (p *PoolServer) generateMergedWorkFromTemplates(coinTemplates template.MergedCoinPairs, signature string) (Work, error) {

	if len(signature) > 96 {
		return nil, errors.New("Signature length is too long")
	}

	primaryCoinTemplate := coinTemplates.GetPrimary()
	primaryCoinConfig, configFound := p.config.BlockchainNodes[primaryCoinTemplate.BlockchainName]
	if !configFound {
		return nil, errors.New("Config not found: " + primaryCoinTemplate.BlockchainName)
	}

	primaryNode := p.coinNodes[primaryCoinTemplate.BlockchainName]

	// TODO - need to go to active rpc config for fallback, not config[0] every time
	primaryChain := block.GetChain(primaryCoinTemplate.BlockchainName, primaryCoinConfig[0].RewardAddress, primaryNode.RPC)

	// TODO - w/ merged mining, it's possible to reuse the previously built work packet for any auxillary/primary coins
	work, err := primaryChain.GenerateWork(&coinTemplates[0], signature)

	if err != nil {
		return work, err
	}

	return work, nil
}

// Main OUTPUT
func (p *PoolServer) recieveWorkFromClient(share Work, client *stratumClient) error {
	templates, err := p.templatesHistory.GetLatest()
	if err != nil {
		return err
	}

	primaryBlockTemplate := templates.GetPrimary()                                    // Maybe we check both?
	primaryBlockTemplate.RequestedWork[block.Extranonce1CacheID] = client.extranonce1 // This isn't very invertable.  It's Bitcoin block specific
	primaryChainName := primaryBlockTemplate.BlockchainName
	primaryChain := block.GetChain(primaryChainName, "", nil)
	primaryBlockHeight := primaryBlockTemplate.RpcBlockTemplate.Height

	status := verifyShare(primaryChain, primaryBlockTemplate, share, p.config.PoolDifficulty)

	if status == shareValid || status == blockCandidate {
		m := "Valid share for block %v from %v"
		m = fmt.Sprintf(m, primaryBlockHeight, client.ip)
		log.Println(m)
	}

	switch status {
	case shareInvalid:
		m := "Invalid share for block %v from %v"
		m = fmt.Sprintf(m, primaryBlockHeight, client.ip)
		return errors.New(m)
	case shareValid:
		return nil
	case blockCandidate:
		// write share to persistence - block candidate
		err = p.submitBlockToChain(primaryBlockTemplate, share, primaryChainName)
		if err != nil {
			m := "Block submission error of block %v from %v, %v"
			m = fmt.Sprintf(m, primaryBlockHeight, client.ip, err.Error())
			return errors.New(m)
		}
		log.Printf("💪😎👍 Successful submission of block %v from: %v", primaryBlockHeight, client.ip)
		// write share to persistence - submission block
		return nil
	default:
		return errors.New("Unkown share status")
	}
}

func (pool *PoolServer) generateWorkFromCache(refresh bool) (Work, error) {
	var work Work
	templates, err := pool.templatesHistory.GetLatest()
	if err != nil {
		return work, err
	}
	work, err = pool.generateMergedWorkFromTemplates(templates, pool.config.BlockSignature)

	if err != nil {
		return work, err
	}

	work = append(work, interface{}(refresh))

	return work, nil
}
