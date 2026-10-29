package shipmentcommercial

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreementmatch"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

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
	charges, err := shipmentrating.PriceAgreementAccessorials(
		ctx,
		entity.AccessorialMeasures(),
		applicable,
		freightOf(entity),
	)
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
	scoped := rateagreementmatch.AutoApplyAccessorials(
		contract.Agreement,
		rateagreementmatch.AccessorialFacts{
			At:             ratedAt,
			ServiceTypeID:  entity.ServiceTypeID,
			ShipmentTypeID: entity.ShipmentTypeID,
		},
	)

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

func reconcileAgreementCharges(
	entity *shipment.Shipment,
	priced []shipmentrating.PricedAgreementAccessorial,
) {
	existing := make(map[pulid.ID]*shipment.AdditionalCharge, len(priced))
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges)+len(priced))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if charge.Owner() != shipmentvocab.SystemOwnerAgreement {
			filtered = append(filtered, charge)
			continue
		}

		if _, seen := existing[*charge.RateAgreementAccessorialID]; !seen {
			existing[*charge.RateAgreementAccessorialID] = charge
		}
	}

	for _, entry := range priced {
		filtered = append(filtered, agreementCharge(entity, entry, existing[entry.Accessorial.ID]))
	}

	entity.AdditionalCharges = filtered
}

func agreementCharge(
	entity *shipment.Shipment,
	entry shipmentrating.PricedAgreementAccessorial,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	accessorialID := entry.Accessorial.ID

	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.AccessorialChargeID = entry.Accessorial.AccessorialChargeID
	charge.Method = entry.Charge.Method
	charge.Amount = entry.Charge.Amount
	charge.Unit = 1
	charge.Quantity = entry.Charge.Quantity
	charge.Rate = entry.Charge.Rate
	charge.RateUnit = entry.Charge.RateUnit
	charge.Label = ""
	charge.RateAgreementAccessorialID = &accessorialID
	charge.RateQuoteID = entity.RateQuoteID

	charge.FuelSurchargeProgramID = nil
	charge.IsDetention = false
	charge.IsTariff = false

	return charge
}

func (p *ratingPass) dropBilledAgreementCharges(entity *shipment.Shipment) {
	if p == nil || len(p.billed) == 0 {
		return
	}
	kept := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.Owner() == shipmentvocab.SystemOwnerAgreement &&
			p.skips(charge.AccessorialChargeID, skipSourceAgreement) {
			continue
		}
		kept = append(kept, charge)
	}
	entity.AdditionalCharges = kept
}

func removeAgreementCharges(entity *shipment.Shipment) {
	filtered := entity.AdditionalCharges[:0]

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.Owner() == shipmentvocab.SystemOwnerAgreement {
			continue
		}
		filtered = append(filtered, charge)
	}

	entity.AdditionalCharges = filtered
}
