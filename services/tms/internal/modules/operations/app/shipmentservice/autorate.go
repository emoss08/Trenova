package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PreviewContractRate answers what the rate agreements would charge for a
// shipment, without writing anything.
//
// The billing panel asks this while a shipment is still being typed, so what
// gets priced is the payload on screen rather than anything stored. No quote is
// recorded: a figure nobody has accepted yet competing with the shipment's real
// rate is exactly what makes a rating history unreadable.
func (s *service) PreviewContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipmentports.ContractRateApplication, error) {
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")

		return nil, multiErr
	}

	auditActor := actor.AuditActor()

	if err := s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	s.applyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err := s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	rated, err := s.commercial.RateAgainstContract(ctx, entity, auditActor.UserID, false)
	if err != nil {
		return nil, err
	}

	return s.describeContractRate(ctx, entity, rated, entity.FreightChargeAmount), nil
}

// AutoRate prices a saved shipment from its contract again.
//
// This is the deliberate re-rate, and it overwrites: the rating method, the
// base rate and every contract accessorial go back to what the agreement says,
// discarding whatever was there. A shipment already invoiced is refused rather
// than repriced, because its customer has seen the numbers.
func (s *service) AutoRate(
	ctx context.Context,
	req *shipmentports.AutoRateShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, *shipmentports.ContractRateApplication, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "AutoRate"),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	plan, err := s.planAutoRate(ctx, req, actor)
	if err != nil {
		return nil, nil, err
	}

	if !plan.rated.Outcome.Priced() {
		return plan.original, s.describeContractRate(
			ctx, plan.entity, plan.rated, plan.previousLinehaul,
		), nil
	}

	if err = s.commercial.RecordContractQuote(ctx, plan.entity, plan.rated); err != nil {
		return nil, nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, plan.entity)
	if err != nil {
		log.Error("failed to save an auto-rated shipment", zap.Error(err))

		return nil, nil, err
	}

	if err = s.logShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		plan.original,
		updatedEntity,
		auditservice.WithComment("Shipment re-rated from its contract"),
		auditservice.WithDiff(plan.original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"auto-rate",
			updatedEntity,
		),
	)

	return updatedEntity, s.describeContractRate(
		ctx, updatedEntity, plan.rated, plan.previousLinehaul,
	), nil
}

func (s *service) PreviewAutoRate(
	ctx context.Context,
	req *shipmentports.AutoRateShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipmentports.ShipmentAutoRatePreview, error) {
	plan, err := s.planAutoRate(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return &shipmentports.ShipmentAutoRatePreview{
		Before: plan.original,
		After:  plan.entity,
		Application: s.describeContractRate(
			ctx, plan.entity, plan.rated, plan.previousLinehaul,
		),
	}, nil
}

type autoRatePlan struct {
	original         *shipment.Shipment
	entity           *shipment.Shipment
	rated            *ratequoteports.RatedShipment
	previousLinehaul decimal.NullDecimal
}

func (s *service) planAutoRate(
	ctx context.Context,
	req *shipmentports.AutoRateShipmentRequest,
	actor *permissionports.RequestActor,
) (*autoRatePlan, error) {
	auditActor := actor.AuditActor()
	original, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if multiErr := validateShipmentNotLockedForBilling(original); multiErr != nil {
		return nil, multiErr
	}

	if multiErr := validateShipmentRateNotLocked(original); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	plan := &autoRatePlan{
		original:         original,
		entity:           entity,
		previousLinehaul: entity.FreightChargeAmount,
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	plan.rated, err = s.commercial.RateAgainstContract(ctx, entity, auditActor.UserID, false)
	if err != nil {
		return nil, err
	}

	if !plan.rated.Outcome.Priced() {
		plan.entity = original

		return plan, nil
	}

	if err = s.commercial.AdoptAndRecalculateContractRate(
		ctx, entity, plan.rated, control, auditActor.UserID,
	); err != nil {
		return nil, err
	}

	return plan, nil
}

// validateShipmentRateNotLocked refuses to reprice a shipment whose numbers the
// customer has already been invoiced against.
func validateShipmentRateNotLocked(entity *shipment.Shipment) *errortypes.MultiError {
	if entity == nil || !entity.RateLocked {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"rateLocked",
		errortypes.ErrInvalidOperation,
		"This shipment's rate is locked and cannot be re-rated",
	)

	return multiErr
}

// describeContractRate turns a rating decision into the account of it the
// dialog reads out.
//
// It reports the contract's own accessorials and not the shipment's other
// charges: a rater accepting a re-rate is agreeing to what the contract adds,
// and listing a fuel surcharge or a detention charge they had already accepted
// would bury that.
func (s *service) describeContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
	previousLinehaul decimal.NullDecimal,
) *shipmentports.ContractRateApplication {
	application := &shipmentports.ContractRateApplication{
		Applied:           rated.Outcome.Priced(),
		Outcome:           rated.Outcome,
		AgreementID:       rated.AgreementID,
		RuleID:            rated.RuleID,
		FormulaTemplateID: rated.FormulaTemplateID,
		BaseRate:          rated.BaseRate,
		LinehaulAmount:    rated.Amount,
	}

	if previousLinehaul.Valid {
		application.PreviousLinehaulAmount = previousLinehaul.Decimal
	}

	if quote := rated.Quote; quote != nil {
		application.Explanation = quote.Explanation()

		if trace := quote.Trace; trace != nil {
			if winner := trace.Winner(); winner != nil {
				application.AgreementName = winner.AgreementName
				application.RuleLabel = winner.RuleLabel
			}
		}
	}

	application.FormulaTemplateName = linehaulTemplateName(rated)
	application.Accessorials = s.describeContractAccessorials(ctx, entity, rated)

	// The totals are the contract's own: its linehaul plus the charges its
	// schedule applies. Fuel and detention are priced by their own engines and
	// are not what somebody accepting a contract rate is agreeing to.
	other := decimal.Zero
	for _, accessorial := range application.Accessorials {
		other = other.Add(accessorial.Amount)
	}

	application.OtherChargeAmount = other
	application.TotalChargeAmount = application.LinehaulAmount.Add(other)

	return application
}

// linehaulTemplateName reads the formula that produced the linehaul off the
// trace, which is where it is recorded whether the rule named the template
// itself or reached it through a rate matrix.
func linehaulTemplateName(rated *ratequoteports.RatedShipment) string {
	if rated.Quote == nil || rated.Quote.Trace == nil {
		return ""
	}

	for i := range rated.Quote.Trace.Components {
		component := &rated.Quote.Trace.Components[i]
		if component.Kind == ratetypes.ComponentKindLinehaul {
			return component.SourceName
		}
	}

	return ""
}

func (s *service) describeContractAccessorials(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
) []shipmentports.ContractRateAccessorial {
	accessorials := s.commercial.ContractAccessorials(ctx, entity, rated)
	applied := make([]shipmentports.ContractRateAccessorial, 0, len(accessorials))

	for _, accessorial := range accessorials {
		if accessorial == nil {
			continue
		}

		applied = append(applied, shipmentports.ContractRateAccessorial{
			AccessorialChargeID: accessorial.AccessorialChargeID,
			Description: s.accessorialDescription(
				ctx, entity, accessorial.AccessorialChargeID, accessorial.AccessorialCharge,
			),
			Method: accessorial.Method,
			Amount: accessorial.PricedAmount(),
			Unit:   1,
		})
	}

	return applied
}

// accessorialDescription names a charge for the dialog, reading it from the
// relation when the charge was loaded with one and looking it up when it was
// not, since a charge the contract has just placed carries no relation yet.
func (s *service) accessorialDescription(
	ctx context.Context,
	entity *shipment.Shipment,
	chargeID pulid.ID,
	loaded *accessorialcharge.AccessorialCharge,
) string {
	if loaded != nil && loaded.Description != "" {
		return loaded.Description
	}

	if s.accessorialRepo == nil || chargeID.IsNil() {
		return ""
	}

	charge, err := s.accessorialRepo.GetByID(ctx, accessorialchargeports.GetAccessorialChargeByIDRequest{
		ID: chargeID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil || charge == nil {
		return ""
	}

	return charge.Description
}
