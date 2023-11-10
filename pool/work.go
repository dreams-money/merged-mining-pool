package pool

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/persistence"
)

// Main INPUT
func (p *PoolServer) fetchRpcBlockTemplatesAndCacheWork() error {
	var block *bitcoin.BitcoinBlock
	var err error
	template, auxblock, err := p.fetchAllBlockTemplatesFromRPC()
	if err != nil {
		return err
	}

	auxillary := p.config.BlockSignature
	if auxblock != nil {
		mergedPOW := auxblock.GetWork()
		auxillary = auxillary + hexStringToByteString(mergedPOW)

		p.templates.AuxBlocks = []bitcoin.AuxBlock{*auxblock}
	}

	primaryName := p.config.GetPrimary()
	rewardPubScriptKey := p.GetPrimaryNode().RewardPubScriptKey
	extranonceByteReservationLength := 8

	block, p.workCache, err = bitcoin.GenerateWork(&template, auxblock,
		primaryName, auxillary, rewardPubScriptKey,
		extranonceByteReservationLength)
	if err != nil {
		log.Print(err)
	}

	p.templates.BitcoinBlock = *block

	return nil
}

// Main OUTPUT
func (p *PoolServer) recieveWorkFromClient(share bitcoin.Work, client *stratumClient) error {
	primaryBlockTemplate := p.templates.GetPrimary()
	if primaryBlockTemplate.Template == nil {
		return errors.New("Primary block template not yet set")
	}
	auxBlock := p.templates.GetAux1()

	var err error

	// TODO - this key and interface isn't very invertable..
	workerString := share[0].(string)
	workerStringParts := strings.Split(workerString, ".")
	if len(workerStringParts) < 2 {
		return errors.New("Invalid miner address")
	}
	minerAddress := workerStringParts[0]
	rigID := workerStringParts[1]

	primaryBlockHeight := primaryBlockTemplate.Template.Height
	nonce := share[primaryBlockTemplate.NonceSubmissionSlot()].(string)
	extranonce2Slot, _ := primaryBlockTemplate.Extranonce2SubmissionSlot()
	extranonce2 := share[extranonce2Slot].(string)
	nonceTime := share[primaryBlockTemplate.NonceTimeSubmissionSlot()].(string)

	// TODO - validate input

	extranonce := client.extranonce1 + extranonce2

	_, err = primaryBlockTemplate.MakeHeader(extranonce, nonce, nonceTime)

	if err != nil {
		return err
	}

	shareStatus, shareDifficulty := validateAndWeighShare(&primaryBlockTemplate, auxBlock, share, p.config.PoolDifficulty)

	heightMessage := fmt.Sprintf("%v", primaryBlockHeight)
	if shareStatus == dualCandidate {
		heightMessage = fmt.Sprintf("%v,%v", primaryBlockHeight, auxBlock.Height)
	} else if shareStatus == aux1Candidate {
		heightMessage = fmt.Sprintf("%v", auxBlock.Height)
	}

	if shareStatus == shareInvalid {
		m := "❔ Invalid share for block %v from %v"
		m = fmt.Sprintf(m, heightMessage, client.ip)
		return errors.New(m)
	}

	m := "Valid share for block %v from %v"
	m = fmt.Sprintf(m, heightMessage, client.ip)
	log.Println(m)

	blockTarget := bitcoin.Target(primaryBlockTemplate.Template.Target)
	blockDifficulty, _ := blockTarget.ToDifficulty()
	blockDifficulty = blockDifficulty * primaryBlockTemplate.ShareMultiplier()

	p.Lock()
	p.shareBuffer = append(p.shareBuffer, persistence.Share{
		PoolID:            p.config.PoolName,
		BlockHeight:       primaryBlockHeight,
		Miner:             minerAddress,
		Worker:            rigID,
		UserAgent:         client.userAgent,
		Difficulty:        shareDifficulty,
		NetworkDifficulty: blockDifficulty,
		IpAddress:         client.ip,
		Created:           time.Now(),
	})
	p.Unlock()

	if shareStatus == shareValid {
		return nil
	}

	statusReadable := statusMap[shareStatus]
	successStatus := 0

	m = "%v block candidate for block %v from %v"
	m = fmt.Sprintf(m, statusReadable, heightMessage, client.ip)
	log.Println(m)

	found := persistence.Found{
		PoolID:               p.config.PoolName,
		Status:               persistence.StatusPending,
		Type:                 statusReadable,
		ConfirmationProgress: 0,
		// Effort: 0, // Filled by payout processor later down the road
		// TransactionConfirmationData: "", // TODO - Return from submit..
		Miner: minerAddress,
		// Reward: 0, // Filled by payout processor later down the road
		Source: "",
	}

	aux1Name := p.config.GetAux1()
	if aux1Name != "" && shareStatus >= aux1Candidate {
		err = p.submitAuxBlock(primaryBlockTemplate, *auxBlock, aux1Name)
		if err != nil {
			log.Println(err)
		} else {
			// EnrichShare
			aux1Target := bitcoin.Target(reverseHexBytes(auxBlock.Target))
			aux1Difficulty, _ := aux1Target.ToDifficulty()
			aux1Difficulty = aux1Difficulty * bitcoin.GetChain(aux1Name).ShareMultiplier()

			found.Chain = aux1Name
			found.Created = time.Now()
			found.Hash = auxBlock.Hash
			found.NetworkDifficulty = aux1Difficulty
			found.BlockHeight = uint(auxBlock.Height)
			// TODO - I may have to edit dogecoind.exe
			found.TransactionConfirmationData = "" // I'm not sure we can get the coinbase from this..

			err = persistence.Blocks.Insert(found)
			if err != nil {
				log.Println(err)
			}

			successStatus = aux1Candidate
		}
	}

	if shareStatus == dualCandidate || shareStatus == primaryCandidate {
		err = p.submitBlockToChain(primaryBlockTemplate, share, p.config.GetPrimary())
		if err != nil {
			return err
		} else {
			found.Chain = p.config.GetPrimary()
			found.Created = time.Now()
			found.Hash, err = primaryBlockTemplate.HeaderHashed()
			if err != nil {
				log.Println(err)
			}
			found.NetworkDifficulty = blockDifficulty
			found.BlockHeight = primaryBlockHeight
			found.TransactionConfirmationData, err = primaryBlockTemplate.CoinbaseHashed()
			if err != nil {
				log.Println(err)
			}

			err = persistence.Blocks.Insert(found)
			if err != nil {
				log.Println(err)
			}
			found.Chain = ""
			if successStatus == aux1Candidate {
				successStatus = dualCandidate
			} else {
				successStatus = primaryCandidate
			}
		}
	}

	statusReadable = statusMap[successStatus]

	log.Printf("✅  Successful %v submission of block %v from: %v", statusReadable, heightMessage, client.ip)

	return nil
}

func (pool *PoolServer) generateWorkFromCache(refresh bool) (bitcoin.Work, error) {
	work := append(pool.workCache, interface{}(refresh))

	return work, nil
}
