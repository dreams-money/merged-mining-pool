package pool

import (
	"errors"
	"fmt"
	"log"

	"designs.capital/dogepool/bitcoin"
)

func (p *PoolServer) generateMergedWorkFromTemplates(coinTemplates Pair, signature string) (bitcoin.Work, error) {

	if len(signature) > 96 {
		return nil, errors.New("Signature length is too long")
	}

	primaryCoinName := p.config.BlockChainOrder.GetPrimary()
	primaryCoin := coinTemplates.GetPrimary()

	// TODO - w/ merged mining, it's possible to reuse the previously built work packet for any auxillary/primary coins
	extranonceByteLength := 8
	block, work, err := bitcoin.GenerateWork(primaryCoin.Template, primaryCoinName, signature, p.activeNodes[primaryCoinName].RewardPubScriptKey, extranonceByteLength)

	p.templates[0] = *block

	if err != nil {
		return work, err
	}

	return work, nil
}

// Main OUTPUT
func (p *PoolServer) recieveWorkFromClient(share bitcoin.Work, client *stratumClient) error {
	templates := p.templates
	primaryBlockTemplate := templates.GetPrimary()

	if primaryBlockTemplate.Template == nil {
		return errors.New("Primary block template not yet set")
	}

	primaryBlockHeight := primaryBlockTemplate.Template.Height

	nonce := share[primaryBlockTemplate.NonceSubmissionSlot()].(string)

	slot, _ := primaryBlockTemplate.Extranonce2SubmissionSlot()
	extranonce2 := share[slot].(string)

	nonceTime := share[primaryBlockTemplate.NonceTimeSubmissionSlot()].(string)

	extranonce := client.extranonce1 + extranonce2

	_, err := primaryBlockTemplate.Header(extranonce, nonce, nonceTime)
	if err != nil {
		return err
	}

	status := verifyShare(primaryBlockTemplate, share, p.config.PoolDifficulty)

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
		err := p.submitBlockToChain(primaryBlockTemplate, share, primaryBlockTemplate.ChainName())
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

func (pool *PoolServer) generateWorkFromCache(refresh bool) (bitcoin.Work, error) {
	work := append(pool.workCache, interface{}(refresh))

	return work, nil
}
