package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type createPreparation struct {
	rating     *shipmentcommercial.ContractRating
	advisories []*errortypes.AdvisoryError
}

func (s *service) PreviewCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) (*services.ShipmentCreatePlan, error) {
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")

		return nil, multiErr
	}

	prepared, err := s.prepareCreate(ctx, entity, actor.AuditActor().UserID)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentCreatePlan{
		Shipment: entity,
		Rating:   ratingOutcome(prepared.rating),
	}, nil
}

func (s *service) prepareCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*createPreparation, error) {
	control, err := s.getShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	entity.ApplyEntryMethodDefault(nil)
	entity.ApplyFreightTermsDefault(nil)
	entity.NormalizeBillTo()

	if multiErr := s.coordinator.PrepareForCreateWithDelayThreshold(
		entity,
		delayThresholdMinutes(control),
	); multiErr != nil {
		return nil, multiErr
	}

	s.dropSystemGeneratedAdditionalChargesForCreate(entity)

	if err = s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	s.applyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err = s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	rating, err := s.commercial.RateAndAdoptContract(ctx, entity, control, userID)
	if err != nil {
		return nil, err
	}

	if err = s.validateExplicitOrder(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validateChargeAllocations(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	multiErr, advisories := s.validator.ValidateCreateWithAdvisories(ctx, entity)
	if multiErr != nil {
		return nil, multiErr
	}

	if err = s.checkDuplicateBOLsWithControl(ctx, control, duplicateBOLCheckRequest(entity)); err != nil {
		return nil, err
	}

	return &createPreparation{rating: rating, advisories: advisories}, nil
}

func ratingOutcome(rating *shipmentcommercial.ContractRating) *services.ShipmentRatingOutcome {
	if rating == nil || rating.Quote == nil || !rating.Quote.Outcome.Priced() {
		return nil
	}

	outcome := &services.ShipmentRatingOutcome{
		Adopted:     rating.Adopted,
		Amount:      rating.ContractAmount,
		Currency:    quoteCurrency(rating.Quote),
		Explanation: rating.Quote.Explanation(),
	}
	if rating.Quote.Trace != nil {
		if winner := rating.Quote.Trace.Winner(); winner != nil {
			outcome.AgreementName = winner.AgreementName
		}
	}

	return outcome
}

func quoteCurrency(quote *ratequote.RateQuote) string {
	return stringutils.FirstNonEmpty(quote.BillingCurrency, quote.Currency, money.DefaultCurrencyCode)
}
