package payouts

import (
	"strings"

	"designs.capital/dogepool/config"
)

func payoutSchemeFactory(schemeName string, config *config.Config) Scheme {
	schemeName = strings.ToUpper(schemeName)
	switch schemeName {
	case "PROP":
		return PROP{}
	case "PPLNS":
		return PPLNS{config}
	case "SOLO":
		return SOLO{}
	default:
		panic("Unknown payout scheme: " + schemeName)
	}
}
