package pool

import (
	"designs.capital/dogepool/bitcoin"
)

const (
	shareValid = iota
	blockCandidate
	shareInvalid
)

func verifyShare(block bitcoin.BitcoinBlock, share bitcoin.Work, poolDifficulty float32) int {

	blockSum, err := block.Sum()
	logOnError(err)

	// TODO - make share multiplier invertable
	poolTarget, _ := bitcoin.TargetFromDifficulty(poolDifficulty / float32(block.ShareMultiplier()))
	poolTargetBig, _ := poolTarget.ToBig()

	chainTarget := bitcoin.Target(block.Template.Target)
	chainTargetBig, _ := chainTarget.ToBig()

	if blockSum.Cmp(chainTargetBig) <= 0 {
		return blockCandidate
	}

	if blockSum.Cmp(poolTargetBig) <= 0 {
		return shareValid
	}

	return shareInvalid
}
