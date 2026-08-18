package detentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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

// resolveContractPrices reads the accessorial schedule of the contract that
// priced this shipment.
//
// This is what makes the rate confirmation and the invoice agree on detention:
// both read the price from the contract rather than one reading the
// organization default and the other whatever was negotiated.
//
// A contract that cannot be read is logged and treated as absent. Detention at
// the organization's default rate is a smaller problem than a shipment whose
// stops cannot be evaluated at all.
func (s *Service) resolveContractPrices(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	ratedAt int64,
) *contractPrices {
	if s.agreementRepo == nil || entity.RateAgreementID == nil ||
		entity.RateAgreementID.IsNil() {
		return nil
	}

	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: *entity.RateAgreementID,
		TenantInfo:      tenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		s.l.Warn("failed to load rate agreement for detention pricing",
			zap.String("shipmentId", entity.ID.String()),
			zap.String("agreementId", entity.RateAgreementID.String()),
			zap.Error(err),
		)

		return nil
	}

	prices := &contractPrices{
		byAccessorial: make(
			map[pulid.ID]*rateagreement.RateAgreementAccessorial,
			len(agreement.Accessorials),
		),
	}

	for _, row := range agreement.Accessorials {
		if row == nil || !row.IsEffectiveAt(ratedAt) {
			continue
		}

		prices.byAccessorial[row.AccessorialChargeID] = row
	}

	if len(prices.byAccessorial) == 0 {
		return nil
	}

	return prices
}
