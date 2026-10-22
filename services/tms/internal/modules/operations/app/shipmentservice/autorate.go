package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
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

	return s.describeContractRate(ctx, entity, rated, entity.FreightChargeAmount)
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
		application, describeErr := s.describeContractRate(
			ctx, plan.entity, plan.rated, plan.previousFreight,
		)
		if describeErr != nil {
			return nil, nil, describeErr
		}
		return plan.original, application, nil
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

	application, err := s.describeContractRate(
		ctx, updatedEntity, plan.rated, plan.previousFreight,
	)
	if err != nil {
		return nil, nil, err
	}
	return updatedEntity, application, nil
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

	application, err := s.describeContractRate(
		ctx, plan.entity, plan.rated, plan.previousFreight,
	)
	if err != nil {
		return nil, err
	}

	return &shipmentports.ShipmentAutoRatePreview{
		Before:      plan.original,
		After:       plan.entity,
		Application: application,
	}, nil
}

type autoRatePlan struct {
	original         *shipment.Shipment
	entity           *shipment.Shipment
	rated            *ratequoteports.RatedShipment
	previousFreight decimal.NullDecimal
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
		previousFreight: entity.FreightChargeAmount,
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

func (s *service) describeContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
	previousFreight decimal.NullDecimal,
) (*shipmentports.ContractRateApplication, error) {
	application := &shipmentports.ContractRateApplication{
		Applied:             rated.Outcome.Priced(),
		Outcome:             rated.Outcome,
		AgreementID:         rated.AgreementID,
		RuleID:              rated.RuleID,
		FormulaTemplateID:   rated.FormulaTemplateID,
		FormulaTemplateName: rated.FormulaTemplateName,
		TemplateVersion:     rated.TemplateVersion,
		Inputs:              rated.TariffInputs,
		FreightAmount:       rated.Amount,
		FreightLines:        []shipment.FreightLine{},
		Charges:             []shipmentports.ContractRateCharge{},
	}
	if rated.Refusal != nil {
		application.Refusal = rated.Refusal.Localize(i18n.FromContext(ctx))
	}
	if previousFreight.Valid {
		application.PreviousFreightAmount = previousFreight.Decimal
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

	charges, err := s.commercial.ContractCharges(ctx, entity, rated)
	if err != nil {
		return nil, err
	}
	application.FreightLines = charges.FreightLines

	other := decimal.Zero
	for _, charge := range charges.Charges {
		application.Charges = append(application.Charges, contractRateCharge(charge))
		other = other.Add(charge.Amount)
	}
	application.OtherChargeAmount = other
	application.TotalChargeAmount = application.FreightAmount.Add(other)

	return application, nil
}

func contractRateCharge(charge *shipment.AdditionalCharge) shipmentports.ContractRateCharge {
	out := shipmentports.ContractRateCharge{
		AccessorialChargeID: charge.AccessorialChargeID,
		Owner:               charge.Owner(),
		Label:               charge.Label,
		Amount:              charge.Amount,
		Quantity:            charge.Quantity,
		Rate:                charge.Rate,
		RateUnit:            charge.RateUnit,
	}
	if definition := charge.AccessorialCharge; definition != nil {
		out.Code = definition.Code
		out.Description = definition.Description
	}
	return out
}
