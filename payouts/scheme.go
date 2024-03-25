package payouts

import (
	"time"

	"designs.capital/dogepool/persistence"
)

type Scheme interface {
	UpdateMinerBalances(poolID string, remainingReward float64, confirmed persistence.Found) (time.Time, error)
}
