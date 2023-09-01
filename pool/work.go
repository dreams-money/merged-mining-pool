package pool

import (
	"errors"
	"fmt"
	"log"

	"designs.capital/dogepool/bitcoin"
)

func (p *PoolServer) fetchRpcBlockTemplatesAndCacheWork() {
	var block *bitcoin.BitcoinBlock
	var err error
	templates, err := p.fetchAllBlockTemplatesFromRPC()

	primaryTemplate := templates[0]
	primaryName := p.config.BlockChainOrder.GetPrimary()

	auxillary := p.config.BlockSignature
	if len(templates) > 1 {
		block, auxillary, err = p.generateAuxHeader(templates[1], auxillary)
		if err != nil {
			log.Print(err)
		}
		p.templates[1] = *block
	}

	rewardPubScriptKey := p.activeNodes[primaryName].RewardPubScriptKey
	extranonceByteReservationLength := 8

	block, p.workCache, err = bitcoin.GenerateWork(&primaryTemplate,
		primaryName, auxillary, rewardPubScriptKey, extranonceByteReservationLength)
	if err != nil {
		log.Print(err)
	}

	p.templates[0] = *block
}

// Main OUTPUT
func (p *PoolServer) recieveWorkFromClient(share bitcoin.Work, client *stratumClient) error {
	var err error

	primaryBlockTemplate := p.templates.GetPrimary()
	primaryBlockHeight := primaryBlockTemplate.Template.Height

	nonce := share[primaryBlockTemplate.NonceSubmissionSlot()].(string)

	slot, _ := primaryBlockTemplate.Extranonce2SubmissionSlot()
	extranonce2 := share[slot].(string)

	extranonce := client.extranonce1 + extranonce2

	_, err = primaryBlockTemplate.Header(extranonce, nonce)
	if err != nil {
		return err
	}

	heightMessage := fmt.Sprintf("%v", primaryBlockHeight)

	var aux1BlockTemplate bitcoin.BitcoinBlock
	if len(p.templates) > 1 {
		aux1BlockTemplate = p.templates.GetAux1()
		aux1BlockHeight := aux1BlockTemplate.Template.Height
		heightMessage = fmt.Sprintf("%v, %v", heightMessage, aux1BlockHeight)

		_, err := aux1BlockTemplate.Header("", nonce)
		if err != nil {
			return err
		}
	}

	status := verifyShare(primaryBlockTemplate, aux1BlockTemplate, share, p.config.PoolDifficulty)

	if status == shareInvalid {
		m := "Invalid share for block %v from %v"
		m = fmt.Sprintf(m, heightMessage, client.ip)
		return errors.New(m)
	}

	m := "Valid share for block %v from %v"
	m = fmt.Sprintf(m, heightMessage, client.ip)
	log.Println(m)

	if status == shareValid {
		return nil
	}

	statusReadable := statusMap[status]

	m = "%v block candidate for block %v from %v"
	m = fmt.Sprintf(m, statusReadable, heightMessage, client.ip)
	log.Println(m)

	var aux1Error error
	if status == aux1Candidate || status == dualCandidate {
		err = p.submitBlockToChain(aux1BlockTemplate, share, aux1BlockTemplate.ChainName())
		if err != nil {
			m := "Aux1 block submission error of block %v from %v, %v"
			m = fmt.Sprintf(m, heightMessage, client.ip, err.Error())
			aux1Error = errors.New(m)
			// We still need to check primary
		} else {
			// write share to persistence - submission block
		}
	}

	if status == primaryCandidate || status == dualCandidate {
		err = p.submitBlockToChain(primaryBlockTemplate, share, primaryBlockTemplate.ChainName())
		if err == nil {
			m := "Primary block submission error of block %v from %v, %v"
			m = fmt.Sprintf(m, heightMessage, client.ip, err.Error())
			err = errors.New(m)
		} else {
			// write share to persistence - submission block
		}
	}

	if err != nil || aux1Error != nil {
		return errors.Join(err, aux1Error)
	}

	log.Printf("💪😎👍 Successful %v submission of block %v from: %v", statusReadable, heightMessage, client.ip)

	return nil
}

func (pool *PoolServer) generateWorkFromCache(refresh bool) (bitcoin.Work, error) {
	work := append(pool.workCache, interface{}(refresh))

	// TODO - I need to get lower of two bits..

	return work, nil
}
