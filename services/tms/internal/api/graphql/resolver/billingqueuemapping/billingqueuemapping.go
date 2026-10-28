package billingqueuemapping

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func RequiredBillingQueueItemToModel(ctx context.Context,
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, errortypes.NewDatabaseError("Billing queue transfer did not return an item").
			WithInternal(errors.New("shipment service returned nil billing queue item"))
	}

	return BillingQueueItemToModel(ctx, item)
}

func BillingQueueItemToModel(ctx context.Context,
	item *billingqueue.BillingQueueItem,
) (*gqlmodel.BillingQueueItem, error) {
	if item == nil {
		return nil, nil
	}

	shipment, err := shipmentmapping.ShipmentToModel(ctx, item.Shipment)
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
		ShipmentID:                gqlid.IDPtr(item.ShipmentID),
		OrderID:                   gqlid.IDPtr(item.OrderID),
		BillToCustomerID:          item.BillToCustomerID.String(),
		AllocatedTotalAmount:      item.AllocatedTotalAmount.StringFixed(2),
		BillToCustomer:            item.BillToCustomer,
		AssignedBillerID:          gqlid.IDPtrFromPulidPtr(item.AssignedBillerID),
		Number:                    item.Number,
		Status:                    item.Status,
		BillType:                  item.BillType,
		ExceptionReasonCode:       item.ExceptionReasonCode,
		ReviewNotes:               item.ReviewNotes,
		ExceptionNotes:            item.ExceptionNotes,
		ReviewStartedAt:           gqlscalar.IntPtr(item.ReviewStartedAt),
		ReviewCompletedAt:         gqlscalar.IntPtr(item.ReviewCompletedAt),
		CanceledByID:              gqlid.IDPtrFromPulidPtr(item.CanceledByID),
		CanceledAt:                gqlscalar.IntPtr(item.CanceledAt),
		CancelReason:              item.CancelReason,
		IsAdjustmentOrigin:        item.IsAdjustmentOrigin,
		SourceInvoiceID:           gqlid.IDPtrFromPulidPtr(item.SourceInvoiceID),
		SourceInvoiceAdjustmentID: gqlid.IDPtrFromPulidPtr(item.SourceInvoiceAdjustmentID),
		SourceCreditMemoInvoiceID: gqlid.IDPtrFromPulidPtr(item.SourceCreditMemoInvoiceID),
		CorrectionGroupID:         gqlid.IDPtrFromPulidPtr(item.CorrectionGroupID),
		RebillStrategy:            gqlscalar.StringPtrFromValue(item.RebillStrategy),
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
