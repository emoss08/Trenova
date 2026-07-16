package resolver

import (
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func requiredBillingQueueItemToModel(
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, errortypes.NewDatabaseError("Billing queue transfer did not return an item").
			WithInternal(errors.New("shipment service returned nil billing queue item"))
	}

	return billingQueueItemToModel(item)
}

func billingQueueItemToModel(
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, nil
	}

	shipment, err := shipmentToModel(item.Shipment)
	if err != nil {
		return nil, err
	}

	adjustmentContext := item.AdjustmentContext
	if adjustmentContext == nil {
		adjustmentContext = map[string]any{}
	}

	return &gqlmodel.BillingQueueItem{
		ID:                        item.ID.String(),
		OrganizationID:            item.OrganizationID.String(),
		BusinessUnitID:            item.BusinessUnitID.String(),
		ShipmentID:                idPtr(item.ShipmentID),
		OrderID:                   idPtr(item.OrderID),
		AssignedBillerID:          idPtrFromPulidPtr(item.AssignedBillerID),
		Number:                    item.Number,
		Status:                    item.Status,
		BillType:                  item.BillType,
		ExceptionReasonCode:       item.ExceptionReasonCode,
		ReviewNotes:               item.ReviewNotes,
		ExceptionNotes:            item.ExceptionNotes,
		ReviewStartedAt:           intPtr(item.ReviewStartedAt),
		ReviewCompletedAt:         intPtr(item.ReviewCompletedAt),
		CanceledByID:              idPtrFromPulidPtr(item.CanceledByID),
		CanceledAt:                intPtr(item.CanceledAt),
		CancelReason:              item.CancelReason,
		IsAdjustmentOrigin:        item.IsAdjustmentOrigin,
		SourceInvoiceID:           idPtrFromPulidPtr(item.SourceInvoiceID),
		SourceInvoiceAdjustmentID: idPtrFromPulidPtr(item.SourceInvoiceAdjustmentID),
		SourceCreditMemoInvoiceID: idPtrFromPulidPtr(item.SourceCreditMemoInvoiceID),
		CorrectionGroupID:         idPtrFromPulidPtr(item.CorrectionGroupID),
		RebillStrategy:            stringPtrFromValue(item.RebillStrategy),
		RequiresReplacementReview: item.RequiresReplacementReview,
		RerateVariancePercent:     item.RerateVariancePercent.String(),
		AdjustmentContext:         adjustmentContext,
		Version:                   int(item.Version),
		CreatedAt:                 int(item.CreatedAt),
		UpdatedAt:                 int(item.UpdatedAt),
		Shipment:                  shipment,
		AssignedBiller:            item.AssignedBiller,
		CanceledBy:                item.CanceledBy,
	}, nil
}
