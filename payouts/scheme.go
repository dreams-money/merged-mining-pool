package payouts

import (
	"time"

	"designs.capital/dogepool/persistence"
)

type Scheme interface {
	UpdateMinerBalances(poolID string, remainingReward float32, confirmed persistence.Found) (time.Time, error)
}
