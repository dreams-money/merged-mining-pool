package block

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"designs.capital/dogepool/blockchain"
	"designs.capital/dogepool/rpc"
	"designs.capital/dogepool/template"
)

const requestedWorkCacheLength = 5 // Relates to length of iota consts below

const (
	coinbaseInitialCacheID = iota
	coinbaseFinalCacheID
	merkleStepsCacheID
	Extranonce1CacheID
	coinbaseCacheID
)

type bitcoin struct {
	poolPubScriptKey string
	chain            blockchain.Blockchain
}

func (b *bitcoin) GenerateWork(blockTemplate *template.Block, arbitrary string) ([]interface{}, error) {
	reversePrevBlockHash, err := reverseHex4Bytes(blockTemplate.RpcBlockTemplate.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	arbitraryWithHeader := bytesWithLengthHeader([]byte(arbitrary))

	extranonceLength := 8 // TODO This poses an issue..  aux headers don't need this.
	coinbaseInitial, coinbaseFinal, err := makeCoinbaseParts(blockTemplate.RpcBlockTemplate, arbitraryWithHeader, b.poolPubScriptKey, extranonceLength)
	coinbaseInitialHex := hex.EncodeToString(coinbaseInitial)
	coinbaseFinalHex := hex.EncodeToString(coinbaseFinal)

	transactionIDs := make([]string, len(blockTemplate.RpcBlockTemplate.Transactions))
	for i, transaction := range blockTemplate.RpcBlockTemplate.Transactions {
		transactionIDs[i] = transaction.ID
	}

	merkleSteps, err := makeRequestMerkleBranchSteps(transactionIDs)
	if err != nil {
		return nil, err
	}

	// We cache coinbase and merkle steps to block template
	blockTemplate.RequestedWork = make([]interface{}, requestedWorkCacheLength)
	blockTemplate.RequestedWork[coinbaseInitialCacheID] = interface{}(coinbaseInitialHex)
	blockTemplate.RequestedWork[coinbaseFinalCacheID] = interface{}(coinbaseFinalHex)
	blockTemplate.RequestedWork[merkleStepsCacheID] = interface{}(merkleSteps)

	work := make([]interface{}, 8)
	work[0] = randString(6) // Job ID
	work[1] = reversePrevBlockHash
	work[2] = coinbaseInitialHex
	work[3] = coinbaseFinalHex
	work[4] = merkleSteps
	work[5] = fmt.Sprintf("%08x", blockTemplate.RpcBlockTemplate.Version)
	work[6] = blockTemplate.RpcBlockTemplate.Bits
	work[7] = fmt.Sprintf("%x", blockTemplate.RpcBlockTemplate.CurrentTime)

	return work, nil
}

func (b *bitcoin) CalculateSum(blockTemplate *template.Block, arbitrary []interface{}) (*big.Int, error) {

	blockHeader, err := createBlockHeader(blockTemplate, arbitrary)
	if err != nil {
		return new(big.Int), err
	}

	sumBytes, err := b.chain.BlockDigest(blockHeader)
	if err != nil {
		return new(big.Int), err
	}

	sumBytes = reverse(sumBytes) // Little endian write

	return new(big.Int).SetBytes(sumBytes), nil
}

func (b *bitcoin) GenerateAuxHeader(t *template.Block, signature string) (string, error) {
	arbitrary := []interface{}{
		interface{}(signature),
	}
	auxHeader, err := createBlockHeader(t, arbitrary)
	if err != nil {
		return "", err
	}

	auxHeaderString := hex.EncodeToString(auxHeader)
	// l := len(auxHeaderString)
	// auxHeaderString = auxHeaderString[:l-4] // remove nonce

	return auxHeaderString, nil
}

func (b *bitcoin) PrepareWorkForSubmission(blockTemplate template.Block, work []interface{}) ([]interface{}, error) {
	header, err := createBlockHeader(&blockTemplate, work)
	if err != nil {
		return nil, err
	}

	coinbase := blockTemplate.RequestedWork[coinbaseCacheID]

	transactionPool := make([]string, len(blockTemplate.RpcBlockTemplate.Transactions))
	for i, transaction := range blockTemplate.RpcBlockTemplate.Transactions {
		transactionPool[i] = transaction.Data
	}

	submissionHex, err := createSubmissionHex(header, coinbase.([]byte), transactionPool)
	if err != nil {
		return nil, err
	}

	submissionInterface := []interface{}{
		interface{}(submissionHex),
	}

	return submissionInterface, nil
}

func (b *bitcoin) ShareMultiplier() float32 {
	return b.chain.ShareMultiplier()
}

func (b *bitcoin) ValidMainnetAddress(address string) bool {
	return b.chain.ValidMainnetAddress(address)
}

func (b *bitcoin) ValidTestnetAddress(address string) bool {
	return b.chain.ValidTestnetAddress(address)
}

func (b *bitcoin) init(rpc *rpc.RPCClient, rewardAddress string, chain blockchain.Blockchain) error {
	if chain == nil {
		return errors.New("Chain cannot be nil")
	}

	if rewardAddress != "" {
		address, err := rpc.ValidateAddress(rewardAddress)
		if err != nil {
			panic(err)
		}
		b.poolPubScriptKey = address.ScriptPubKey
	}

	b.chain = chain
	return nil
}

func createBlockHeader(blockTemplate *template.Block, submittedWork []interface{}) ([]byte, error) {
	emptyBytes := []byte{}

	version := uint(blockTemplate.RpcBlockTemplate.Version)
	prevBlockHash := blockTemplate.RpcBlockTemplate.PrevBlockHash

	coinbaseInitialHex := blockTemplate.RequestedWork[coinbaseInitialCacheID].(string)
	coinbaseInitial, err := hex.DecodeString(coinbaseInitialHex)
	if err != nil {
		return emptyBytes, err
	}

	coinbaseFinalHex := blockTemplate.RequestedWork[coinbaseFinalCacheID].(string)
	coinbaseFinal, err := hex.DecodeString(coinbaseFinalHex)
	if err != nil {
		return emptyBytes, err
	}

	merkleBranchSteps := blockTemplate.RequestedWork[merkleStepsCacheID].([]string)

	extraNonce1Hex := blockTemplate.RequestedWork[Extranonce1CacheID].(string)
	extraNonce1, err := hex.DecodeString(extraNonce1Hex)
	if err != nil {
		return emptyBytes, err
	}

	extraNonce2Hex := submittedWork[2].(string)
	extraNonce2, err := hex.DecodeString(extraNonce2Hex)
	if err != nil {
		return emptyBytes, err
	}

	arbitrary := append(extraNonce1, extraNonce2...)

	coinbase := finalizeCoinbase(coinbaseInitial, arbitrary, coinbaseFinal, blockTemplate)
	merkleRoot, err := makeHeaderMerkleRoot(coinbase, merkleBranchSteps)
	if err != nil {
		return nil, err
	}

	nonceTimeHex := submittedWork[3].(string)
	bits := blockTemplate.RpcBlockTemplate.Bits
	nonceHex := submittedWork[4].(string)

	return blockHeader(version, prevBlockHash, hex.EncodeToString(merkleRoot), nonceTimeHex, bits, nonceHex)
}
