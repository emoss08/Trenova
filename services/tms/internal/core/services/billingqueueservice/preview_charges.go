package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func (s *service) PreviewUpdateCharges(
	ctx context.Context,
	req *services.UpdateChargesRequest,
	actor *services.RequestActor,
) (*services.ChargeUpdatePreview, error) {
	return s.planChargeUpdate(ctx, req, actor, false)
}

// planChargeUpdate is everything UpdateCharges decides before it saves: the
// item must be in review with no sibling payer invoiced, the charges are
// replaced with the system-owned fields restored, the totals are derived,
// and a split by amount the edit leaves stale is refused or converted. A
// rate or formula change reprices through the rating engine, which writes,
// so only the write may make one.
func (s *service) planChargeUpdate(
	ctx context.Context,
	req *services.UpdateChargesRequest,
	actor *services.RequestActor,
	rerate bool,
) (*services.ChargeUpdatePreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Update charges request is required",
		)
	}

	repricing := (req.FormulaTemplateID != nil && !req.FormulaTemplateID.IsNil()) ||
		req.BaseRate != nil
	if repricing && !rerate {
		return nil, errortypes.NewBusinessError(
			"A rate or formula change reprices the shipment and cannot be previewed",
		)
	}

	item, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if item.Status != billingqueue.StatusInReview {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Charges can only be edited when the item is in InReview status",
		)
	}

	if err = s.guardSiblingPayersUnposted(ctx, item, req.TenantInfo); err != nil {
		return nil, err
	}

	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}
	before := *shp

	if err = s.applyRateChange(ctx, shp, req, actor); err != nil {
		return nil, err
	}

	chargeNames := accessorialNames(shp.AdditionalCharges)
	if req.AdditionalCharges != nil {
		shipment.RestoreSystemOwnedCharges(shp.AdditionalCharges, req.AdditionalCharges)
		shp.AdditionalCharges = req.AdditionalCharges
	}

	freight := shp.FreightChargeAmount.Decimal
	otherTotal := shipment.AdditionalChargesTotal(shp.AdditionalCharges, freight)
	shp.OtherChargeAmount = decimal.NewNullDecimal(otherTotal)
	shp.TotalChargeAmount = decimal.NewNullDecimal(freight.Add(otherTotal))

	convertedSplits := 0
	if stale := shipment.FindStaleAmountSplits(shp, shp.ChargeAllocations); len(stale) > 0 {
		if !req.ConvertAmountSplitsToPercent {
			return nil, staleAmountSplitError(stale, chargeNames)
		}
		convertedSplits = shipment.ConvertAmountSplitsToPercent(shp, shp.ChargeAllocations)
	}

	return &services.ChargeUpdatePreview{
		Item:            item,
		Before:          &before,
		After:           shp,
		ConvertedSplits: convertedSplits,
	}, nil
}

func (s *service) applyRateChange(
	ctx context.Context,
	shp *shipment.Shipment,
	req *services.UpdateChargesRequest,
	actor *services.RequestActor,
) error {
	switch {
	case req.FormulaTemplateID != nil && !req.FormulaTemplateID.IsNil():
		shp.FormulaTemplateID = *req.FormulaTemplateID

		if err := s.rerateShipment(ctx, shp, req.TenantInfo, actor.UserID); err != nil {
			return errortypes.NewValidationError(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Failed to recalculate charges with the selected formula template",
			)
		}
	case req.BaseRate != nil:
		shp.BaseRate = decimal.NewNullDecimal(*req.BaseRate)

		if err := s.rerateShipment(ctx, shp, req.TenantInfo, actor.UserID); err != nil {
			return errortypes.NewValidationError(
				"baseRate",
				errortypes.ErrInvalid,
				"Failed to recalculate charges with updated base rate",
			)
		}
	}

	return nil
}
