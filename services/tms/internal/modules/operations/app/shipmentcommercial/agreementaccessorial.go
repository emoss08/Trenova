package shipmentcommercial

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const cents = 2

func (c *Calculator) loadAgreementByID(
	ctx context.Context,
	entity *shipment.Shipment,
	agreementID *pulid.ID,
) *rateagreement.RateAgreement {
	if c.agreementRepo == nil || agreementID == nil || agreementID.IsNil() {
		return nil
	}

	agreement, err := c.agreementRepo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: *agreementID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeChildren: true,
		OmitRules:       true,
	})
	if err != nil {
		c.logger.Warn("failed to load rate agreement for shipment",
			zap.String("shipmentId", entity.ID.String()),
			zap.String("agreementId", agreementID.String()),
			zap.Error(err),
		)

		return nil
	}

	return agreement
}

func fuelOverride(agreement *rateagreement.RateAgreement) *fuelsurchargeports.FuelProgramOverride {
	if agreement == nil || agreement.FuelBinding == nil {
		return nil
	}

	binding := agreement.FuelBinding

	return &fuelsurchargeports.FuelProgramOverride{
		AgreementID:   agreement.ID,
		ProgramID:     binding.FuelSurchargeProgramID,
		Waived:        binding.Waived,
		PegPrice:      binding.PegPriceOverride,
		IncrementRate: binding.IncrementRateOverride,
		CapAmount:     binding.CapAmount,
	}
}

func (c *Calculator) syncAgreementAccessorials(
	ctx context.Context,
	entity *shipment.Shipment,
	contract *ratequoteports.RatedShipment,
	pass *ratingPass,
) error {
	if contract.Agreement == nil {
		removeAgreementCharges(entity)
		return nil
	}

	applicable := applicableAccessorials(contract, entity, ratingDate(entity, c.now), pass)
	charges, err := pricedAgreementCharges(ctx, entity, applicable, freightOf(entity))
	if err != nil {
		return err
	}
	reconcileAgreementCharges(entity, charges)
	return nil
}

func applicableAccessorials(
	contract *ratequoteports.RatedShipment,
	entity *shipment.Shipment,
	ratedAt int64,
	pass *ratingPass,
) []*rateagreement.RateAgreementAccessorial {
	scoped := contract.Agreement.AutoApplyAccessorials(rateagreement.AccessorialFacts{
		At:             ratedAt,
		ServiceTypeID:  entity.ServiceTypeID,
		ShipmentTypeID: entity.ShipmentTypeID,
	})

	applicable := make([]*rateagreement.RateAgreementAccessorial, 0, len(scoped))
	for _, accessorial := range scoped {
		if strings.TrimSpace(accessorial.Condition) != "" && !contract.Conditions[accessorial.ID] {
			continue
		}
		if pass.skips(accessorial.AccessorialChargeID, skipSourceAgreement) {
			continue
		}
		applicable = append(applicable, accessorial)
	}
	return applicable
}

type pricedAccessorial struct {
	accessorial *rateagreement.RateAgreementAccessorial
	charge      *shipment.AdditionalCharge
}

func measuresOf(entity *shipment.Shipment) *rateagreement.AccessorialMeasures {
	return &rateagreement.AccessorialMeasures{
		Miles:  entity.RatingMiles(),
		Dwells: entity.DwellHours(),
		Days:   entity.ServiceDays(),
		Stops:  decimal.NewFromInt(int64(entity.RatedStopCount())),
	}
}

func pricedAgreementCharges(
	ctx context.Context,
	entity *shipment.Shipment,
	applicable []*rateagreement.RateAgreementAccessorial,
	freight decimal.Decimal,
) ([]pricedAccessorial, error) {
	measures := measuresOf(entity)
	out := make([]pricedAccessorial, 0, len(applicable))
	percentages := make([]int, 0, len(applicable))

	for _, accessorial := range applicable {
		charge := &shipment.AdditionalCharge{Unit: 1}
		switch accessorial.Method {
		case accessorialcharge.MethodPercentage:
			charge.Method = accessorialcharge.MethodPercentage
			charge.Amount = accessorial.PricedAmount()
			charge.Rate = decimal.NewNullDecimal(accessorial.Amount)
			percentages = append(percentages, len(out))
		case accessorialcharge.MethodPerUnit:
			quantity := accessorial.BillableQuantity(measures)
			amount, _ := accessorial.Cap(accessorial.PricedAmount().Mul(quantity).Round(cents))
			charge.Method = accessorialcharge.MethodFlat
			charge.Amount = amount
			charge.Quantity = decimal.NewNullDecimal(quantity)
			charge.Rate = decimal.NewNullDecimal(accessorial.Amount)
			charge.RateUnit = accessorial.TariffRateUnit()
		default:
			amount, _ := accessorial.Cap(accessorial.PricedAmount())
			charge.Method = accessorialcharge.MethodFlat
			charge.Amount = amount
			charge.Quantity = decimal.NewNullDecimal(decimal.NewFromInt(1))
		}
		out = append(out, pricedAccessorial{accessorial: accessorial, charge: charge})
	}

	if err := capPercentages(ctx, out, percentages, freight); err != nil {
		return nil, err
	}

	kept := out[:0]
	for _, priced := range out {
		if priced.charge.Amount.IsPositive() {
			kept = append(kept, priced)
		}
	}
	return kept, nil
}

func capPercentages(
	ctx context.Context,
	priced []pricedAccessorial,
	percentages []int,
	freight decimal.Decimal,
) error {
	if len(percentages) == 0 {
		return nil
	}
	charges := make([]*shipment.AdditionalCharge, 0, len(percentages))
	for _, index := range percentages {
		charges = append(charges, priced[index].charge)
	}
	totals, err := shipment.AdditionalChargeTotals(ctx, charges, freight)
	if err != nil {
		return err
	}
	for position, index := range percentages {
		entry := priced[index]
		capped, applied := entry.accessorial.Cap(totals.Each[position])
		if !applied {
			continue
		}
		entry.charge.Method = accessorialcharge.MethodFlat
		entry.charge.Amount = capped
	}
	return nil
}

func reconcileAgreementCharges(entity *shipment.Shipment, priced []pricedAccessorial) {
	existing := make(map[pulid.ID]*shipment.AdditionalCharge, len(priced))
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges)+len(priced))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if charge.Owner() != shipment.SystemOwnerAgreement {
			filtered = append(filtered, charge)
			continue
		}

		if _, seen := existing[*charge.RateAgreementAccessorialID]; !seen {
			existing[*charge.RateAgreementAccessorialID] = charge
		}
	}

	for _, entry := range priced {
		filtered = append(filtered, agreementCharge(entity, entry, existing[entry.accessorial.ID]))
	}

	entity.AdditionalCharges = filtered
}

func agreementCharge(
	entity *shipment.Shipment,
	entry pricedAccessorial,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	accessorialID := entry.accessorial.ID

	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.AccessorialChargeID = entry.accessorial.AccessorialChargeID
	charge.Method = entry.charge.Method
	charge.Amount = entry.charge.Amount
	charge.Unit = 1
	charge.Quantity = entry.charge.Quantity
	charge.Rate = entry.charge.Rate
	charge.RateUnit = entry.charge.RateUnit
	charge.Label = ""
	charge.RateAgreementAccessorialID = &accessorialID
	charge.RateQuoteID = entity.RateQuoteID

	charge.FuelSurchargeProgramID = nil
	charge.IsDetention = false
	charge.IsTariff = false

	return charge
}

func removeAgreementCharges(entity *shipment.Shipment) {
	filtered := entity.AdditionalCharges[:0]

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.Owner() == shipment.SystemOwnerAgreement {
			continue
		}
		filtered = append(filtered, charge)
	}

	entity.AdditionalCharges = filtered
}
