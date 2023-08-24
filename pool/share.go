package pool

import (
	"designs.capital/dogepool/block"
	"designs.capital/dogepool/template"
)

const (
	shareValid = iota
	blockCandidate
	shareInvalid
)

func verifyShare(chain block.BlockchainProcessor, blockTemplate template.Block, share []interface{}, poolDifficulty float32) int {
	blockSum, err := chain.CalculateSum(&blockTemplate, share)
	logOnError(err)

	// TODO - make share multiplier invertable
	poolTarget, _ := block.TargetFromDifficulty(poolDifficulty / float32(chain.ShareMultiplier()))
	poolTargetBig, _ := poolTarget.ToBig()

	chainTarget := block.Target(blockTemplate.RpcBlockTemplate.Target)
	chainTargetBig, _ := chainTarget.ToBig()

	if blockSum.Cmp(chainTargetBig) <= 0 {
		return blockCandidate
	}

	if blockSum.Cmp(poolTargetBig) <= 0 {
		return shareValid
	}

	return shareInvalid
}
