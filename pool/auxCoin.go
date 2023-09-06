package pool

import (
	"encoding/hex"
	"fmt"

	"designs.capital/dogepool/bitcoin"
)

func (p *PoolServer) generateAuxHeader(auxTemplate bitcoin.Template, signature string) (*bitcoin.BitcoinBlock, string, error) {
	aux1Name := p.config.BlockChainOrder.GetAux1()
	rewardPubScriptKey := p.activeNodes[aux1Name].RewardPubScriptKey

	block, _, err := bitcoin.GenerateWork(&auxTemplate, aux1Name, signature, rewardPubScriptKey, 0)
	if err != nil {
		return nil, "", err
	}

	header, err := block.Header("", "", fmt.Sprintf("%x", auxTemplate.CurrentTime))
	if err != nil {
		return nil, "", err
	}

	return block, hexToStringByte(header), nil
}

func hexToStringByte(header string) string {
	hexBytes, err := hex.DecodeString(header)
	if err != nil {
		panic(err)
	}
	return string(hexBytes)
}
