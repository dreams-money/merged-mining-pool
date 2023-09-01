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

func verifyShare(primary, aux1 bitcoin.BitcoinBlock, share bitcoin.Work, poolDifficulty float32) int {

	primarySum, err := primary.Sum()
	logOnError(err)

	primaryTarget := bitcoin.Target(primary.Template.Target)
	primaryTargetBig, _ := primaryTarget.ToBig()

	// fmt.Println("     Primary Sum", primarySum.Text(16))
	// fmt.Println("  Primary Target", primaryTarget)

	auxStatus := int(0)
	if aux1.Template != nil {
		aux1Sum, err := aux1.Sum()
		logOnError(err)

		aux1Target := bitcoin.Target(aux1.Template.Target)
		aux1TargetBig, _ := aux1Target.ToBig()

		// fmt.Println("         Aux sum", aux1Sum.Text(16))
		// fmt.Println("      Aux Target", aux1Target)

		if aux1Sum.Cmp(aux1TargetBig) <= 0 {
			auxStatus = aux1Candidate
		}
	}

	if primarySum.Cmp(primaryTargetBig) <= 0 {
		if auxStatus == aux1Candidate {
			return dualCandidate
		}
		return primaryCandidate
	}

	if auxStatus == aux1Candidate {
		return aux1Candidate
	}

	// Not sure if auxCoin.ShareMultiplier() every varies..
	poolTarget, _ := bitcoin.TargetFromDifficulty(poolDifficulty / float32(primary.ShareMultiplier()))
	poolTargettBig, _ := poolTarget.ToBig()

	// fmt.Println("      Pool Target", poolTarget)

	if primarySum.Cmp(poolTargettBig) <= 0 {
		return shareValid
	}

	return shareInvalid
}
