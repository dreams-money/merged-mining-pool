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

func validateAndWeighShare(primary *bitcoin.BitcoinBlock, aux1 *bitcoin.AuxBlock, share bitcoin.Work, poolDifficulty float64) (int, float64) {
	primarySum, err := primary.Sum()
	logOnError(err)

	primaryTarget := bitcoin.Target(primary.Template.Target)
	primaryTargetBig, _ := primaryTarget.ToBig()

	// TODO - We should probably abstract this to the bitcoin/chain package. I.e. chain.getDifficulty()
	primaryHash := primarySum.Text(16)
	primaryHashHit := bitcoin.Target(primaryHash)
	shareDifficulty, _ := primaryHashHit.ToDifficulty()
	shareDifficulty = shareDifficulty * primary.ShareMultiplier()

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
		return status, shareDifficulty
	}

	poolTarget, _ := bitcoin.TargetFromDifficulty(poolDifficulty / primary.ShareMultiplier())
	poolTargettBig, _ := poolTarget.ToBig()

	if primarySum.Cmp(poolTargettBig) <= 0 {
		return shareValid, shareDifficulty
	}

	return shareInvalid, shareDifficulty
}
