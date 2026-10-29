package detentionsync

import (
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// contractPrices is what a shipment's own contract charges for the accessorials
// detention bills against, keyed by accessorial.
//
// It is resolved once per shipment rather than once per stop: a load with six
// stops would otherwise read the same contract six times to reach the same
// answer.
type contractPrices struct {
	byAccessorial map[pulid.ID]*rateagreement.RateAgreementAccessorial
}

// priceFor returns the contract's rate for an accessorial, and whether the
// contract priced it at all.
//
// A waived row still counts as priced: the contract giving detention away is a
// stated term, and falling through to the organization default would bill the
// customer for something they were promised they would not see.
func (c *contractPrices) priceFor(
	charge *accessorialcharge.AccessorialCharge,
) (decimal.Decimal, accessorialcharge.RateUnit, bool) {
	if c == nil || charge == nil {
		return decimal.Zero, "", false
	}

	row, ok := c.byAccessorial[charge.ID]
	if !ok {
		return decimal.Zero, "", false
	}

	unit := row.RateUnit
	if unit == "" {
		unit = charge.RateUnit
	}

	return row.PricedAmount(), unit, true
}
