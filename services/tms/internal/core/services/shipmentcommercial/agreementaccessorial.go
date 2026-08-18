package shipmentcommercial

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// shipmentSchemaID names the formula environment an accessorial condition is
// written against — the same one the rating formulas use, so a pricing analyst
// writes conditions in the vocabulary they already know.
const shipmentSchemaID = "shipment"

// loadAgreement reads the contract the rate engine just resolved, with its
// accessorial schedule and fuel binding attached.
//
// A contract that cannot be read is logged and treated as absent rather than
// failing the save. The linehaul is already priced by this point, and a
// shipment missing an automatic accessorial is a far smaller problem than a
// shipment that cannot be saved at all.
func (c *Calculator) loadAgreement(
	ctx context.Context,
	entity *shipment.Shipment,
) *rateagreement.RateAgreement {
	if c.agreementRepo == nil || entity.RateAgreementID == nil ||
		entity.RateAgreementID.IsNil() {
		return nil
	}

	agreement, err := c.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: *entity.RateAgreementID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeChildren: true,
	})
	if err != nil {
		c.logger.Warn("failed to load rate agreement for shipment",
			zap.String("shipmentId", entity.ID.String()),
			zap.String("agreementId", entity.RateAgreementID.String()),
			zap.Error(err),
		)

		return nil
	}

	return agreement
}

// fuelOverride translates the contract's fuel binding into the neutral form the
// fuel service takes, so the fuel service never learns about rate agreements.
func fuelOverride(agreement *rateagreement.RateAgreement) *services.FuelProgramOverride {
	if agreement == nil || agreement.FuelBinding == nil {
		return nil
	}

	binding := agreement.FuelBinding

	return &services.FuelProgramOverride{
		AgreementID:   agreement.ID,
		ProgramID:     binding.FuelSurchargeProgramID,
		Waived:        binding.Waived,
		PegPrice:      binding.PegPriceOverride,
		IncrementRate: binding.IncrementRateOverride,
		CapAmount:     binding.CapAmount,
	}
}

// syncAgreementAccessorials rebuilds the charges a contract's accessorial
// schedule applies automatically.
//
// This is what makes the rate confirmation and the invoice agree: both read the
// accessorial's price from the same contract, rather than one reading the
// organization default and the other whatever a clerk typed.
//
// It runs before the fuel surcharge, because a fuel program that takes a
// percentage of linehaul plus accessorials has to see them first.
func (c *Calculator) syncAgreementAccessorials(
	ctx context.Context,
	entity *shipment.Shipment,
	agreement *rateagreement.RateAgreement,
) {
	if agreement == nil {
		removeAgreementCharges(entity)
		return
	}

	reconcileAgreementCharges(
		entity,
		c.applicableAccessorials(ctx, agreement, entity, ratingDate(entity, c.now)),
	)
}

// applicableAccessorials picks the schedule rows that apply to this shipment.
//
// Only rows marked to apply automatically are considered; the rest are prices
// waiting for somebody to add the charge. An empty service or shipment type set
// means the row does not care, which is the same convention every other scoped
// record in the system uses.
//
// The cheap set checks run before the expression, because a condition is the
// only test here that costs anything to answer.
func (c *Calculator) applicableAccessorials(
	ctx context.Context,
	agreement *rateagreement.RateAgreement,
	entity *shipment.Shipment,
	ratedAt int64,
) []*rateagreement.RateAgreementAccessorial {
	applicable := make([]*rateagreement.RateAgreementAccessorial, 0, len(agreement.Accessorials))

	for _, accessorial := range agreement.Accessorials {
		if accessorial == nil || !accessorial.AutoApply || accessorial.Waived {
			continue
		}

		if !accessorial.IsEffectiveAt(ratedAt) {
			continue
		}

		if !matchesIDSet(accessorial.ServiceTypeIDs, entity.ServiceTypeID) ||
			!matchesIDSet(accessorial.ShipmentTypeIDs, entity.ShipmentTypeID) {
			continue
		}

		if !c.conditionHolds(ctx, accessorial, entity) {
			continue
		}

		applicable = append(applicable, accessorial)
	}

	return applicable
}

// conditionHolds evaluates the accessorial's own applicability expression.
//
// A condition that will not evaluate withholds the charge rather than applying
// it. Billing a customer for something the contract may not entitle us to is
// the more expensive mistake of the two, and the warning names the row so the
// contract can be corrected.
func (c *Calculator) conditionHolds(
	ctx context.Context,
	accessorial *rateagreement.RateAgreementAccessorial,
	entity *shipment.Shipment,
) bool {
	if accessorial.ApplyCondition == "" {
		return true
	}

	if c.predicate == nil {
		return false
	}

	holds, err := c.predicate.EvaluatePredicate(ctx, &services.EvaluatePredicateRequest{
		Expression: accessorial.ApplyCondition,
		SchemaID:   shipmentSchemaID,
		Entity:     entity,
	})
	if err != nil {
		c.logger.Warn("rate agreement accessorial condition could not be evaluated",
			zap.String("shipmentId", entity.ID.String()),
			zap.String("accessorialId", accessorial.ID.String()),
			zap.String("condition", accessorial.ApplyCondition),
			zap.Error(err),
		)

		return false
	}

	return holds
}

func matchesIDSet(allowed []pulid.ID, value pulid.ID) bool {
	if len(allowed) == 0 {
		return true
	}

	for _, candidate := range allowed {
		if candidate == value {
			return true
		}
	}

	return false
}

// reconcileAgreementCharges rewrites the shipment's contract accessorials so
// exactly one charge survives per applicable schedule row.
//
// Existing rows are reused rather than replaced, for the same reason the
// detention reconciliation does it: a charge that keeps its identity keeps its
// audit trail, and anything pointing at it stays resolvable after the save.
func reconcileAgreementCharges(
	entity *shipment.Shipment,
	applicable []*rateagreement.RateAgreementAccessorial,
) {
	existing := make(map[pulid.ID]*shipment.AdditionalCharge, len(applicable))
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

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

	for _, accessorial := range applicable {
		filtered = append(filtered, agreementCharge(
			entity,
			accessorial,
			existing[accessorial.ID],
		))
	}

	entity.AdditionalCharges = filtered
}

func agreementCharge(
	entity *shipment.Shipment,
	accessorial *rateagreement.RateAgreementAccessorial,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	accessorialID := accessorial.ID

	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.AccessorialChargeID = accessorial.AccessorialChargeID
	charge.Method = accessorial.Method
	charge.Amount = accessorial.PricedAmount()
	charge.Unit = 1
	charge.RateAgreementAccessorialID = &accessorialID
	charge.RateQuoteID = entity.RateQuoteID

	// The owner columns are mutually exclusive, and the database enforces it.
	// Clearing the others here means a charge can never be claimed by two
	// engines even if it was previously produced by one of them.
	charge.FuelSurchargeProgramID = nil
	charge.DetentionOccurrenceID = nil

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
