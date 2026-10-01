package base

import (
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func RequiredBillingQueueItemToModel(
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, errortypes.NewDatabaseError("Billing queue transfer did not return an item").
			WithInternal(errors.New("shipment service returned nil billing queue item"))
	}

	return BillingQueueItemToModel(item)
}

func BillingQueueItemToModel(
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, nil
	}

	shipment, err := ShipmentToModel(item.Shipment)
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
		ShipmentID:                IDPtr(item.ShipmentID),
		OrderID:                   IDPtr(item.OrderID),
		BillToCustomerID:          item.BillToCustomerID.String(),
		AllocatedTotalAmount:      item.AllocatedTotalAmount.StringFixed(2),
		BillToCustomer:            item.BillToCustomer,
		AssignedBillerID:          IDPtrFromPulidPtr(item.AssignedBillerID),
		Number:                    item.Number,
		Status:                    item.Status,
		BillType:                  item.BillType,
		ExceptionReasonCode:       item.ExceptionReasonCode,
		ReviewNotes:               item.ReviewNotes,
		ExceptionNotes:            item.ExceptionNotes,
		ReviewStartedAt:           IntPtr(item.ReviewStartedAt),
		ReviewCompletedAt:         IntPtr(item.ReviewCompletedAt),
		CanceledByID:              IDPtrFromPulidPtr(item.CanceledByID),
		CanceledAt:                IntPtr(item.CanceledAt),
		CancelReason:              item.CancelReason,
		IsAdjustmentOrigin:        item.IsAdjustmentOrigin,
		SourceInvoiceID:           IDPtrFromPulidPtr(item.SourceInvoiceID),
		SourceInvoiceAdjustmentID: IDPtrFromPulidPtr(item.SourceInvoiceAdjustmentID),
		SourceCreditMemoInvoiceID: IDPtrFromPulidPtr(item.SourceCreditMemoInvoiceID),
		CorrectionGroupID:         IDPtrFromPulidPtr(item.CorrectionGroupID),
		RebillStrategy:            StringPtrFromValue(item.RebillStrategy),
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
