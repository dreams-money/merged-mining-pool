package pool

import (
	"designs.capital/dogepool/bitcoin"
)

const (
	shareInvalid = iota
	shareValid
	primaryCandidate
	aux1Candidate
	dualCandidate
)

var statusMap = map[int]string{
	2: "Primary",
	3: "Aux1",
	4: "Dual",
}

func verifyShare(primary *bitcoin.BitcoinBlock, aux1 *bitcoin.AuxBlock, share bitcoin.Work, poolDifficulty float32) int {
	primarySum, err := primary.Sum()
	logOnError(err)

	primaryTarget := bitcoin.Target(primary.Template.Target)
	primaryTargetBig, _ := primaryTarget.ToBig()

	status := shareInvalid

	if primarySum.Cmp(primaryTargetBig) <= 0 {
		status = primaryCandidate
	}

	if aux1 != nil {
		auxTarget := bitcoin.Target(reverseHexBytes(aux1.Target))
		auxTargetBig, _ := auxTarget.ToBig()

		if primarySum.Cmp(auxTargetBig) <= 0 {
			if status == primaryCandidate {
				status = dualCandidate
			} else {
				status = aux1Candidate
			}
		}
	}

	if status > shareInvalid {
		return status
	}

	poolTarget, _ := bitcoin.TargetFromDifficulty(poolDifficulty / float32(primary.ShareMultiplier()))
	poolTargettBig, _ := poolTarget.ToBig()

	if primarySum.Cmp(poolTargettBig) <= 0 {
		return shareValid
	}

	return shareInvalid
}
