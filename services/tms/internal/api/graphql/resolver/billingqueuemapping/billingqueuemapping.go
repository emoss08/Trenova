package billingqueuemapping

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
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
		return nil, nil //nolint:nilnil // a missing item maps to a null GraphQL field
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
		Lane:                      item.Lane,
		HoldReasonCode:            item.HoldReasonCode,
		PoNumber:                  item.PONumber,
		AgentVerdict:              item.AgentVerdict,
		AgentSummary:              gqlscalar.StringPtrFromValue(item.AgentSummary),
		AgentReviewedAt:           gqlscalar.IntPtr(item.AgentReviewedAt),
		Headline:                  headlineToModel(item.Headline),
	}, nil
}

func headlineToModel(headline *billingqueue.Headline) *gqlmodel.BillingQueueHeadline {
	if headline == nil {
		return nil
	}

	return &gqlmodel.BillingQueueHeadline{
		Code:     string(headline.Code),
		CheckKey: string(headline.CheckKey),
		Summary:  headline.Summary,
		Open:     headline.Open,
		Waiting:  headline.Waiting,
	}
}

// ListRequestFromFilter reads the queue filter a client sent into the list request,
// rejecting a lane or amount the queue does not know.
func ListRequestFromFilter(
	filter *gqlmodel.BillingQueueItemsFilterInput,
) (*billingqueueports.ListBillingQueueItemsRequest, error) {
	req := &billingqueueports.ListBillingQueueItemsRequest{}
	if filter == nil {
		return req, nil
	}

	for _, lane := range filter.Lanes {
		if !lane.IsValid() {
			return nil, errortypes.NewValidationError(
				"filter.lanes",
				errortypes.ErrInvalid,
				"Unknown billing queue lane {0}",
				string(lane),
			)
		}
		req.Lanes = append(req.Lanes, lane)
	}
	req.IncludePosted = filter.IncludePosted != nil && *filter.IncludePosted

	var err error
	if req.MinAmount, err = amountFilter("filter.minAmount", filter.MinAmount); err != nil {
		return nil, err
	}
	if req.MaxAmount, err = amountFilter("filter.maxAmount", filter.MaxAmount); err != nil {
		return nil, err
	}
	if filter.CreatedBefore != nil && *filter.CreatedBefore > 0 {
		before := int64(*filter.CreatedBefore)
		req.CreatedBefore = &before
	}

	return req, nil
}

func amountFilter(field string, value *string) (decimal.NullDecimal, error) {
	if value == nil || *value == "" {
		return decimal.NullDecimal{}, nil
	}

	amount, err := decimal.NewFromString(*value)
	if err != nil || amount.IsNegative() {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Enter an amount of zero or more",
		)
	}

	return decimal.NewNullDecimal(amount), nil
}

func BillingQueueItemConnectionToModel(
	ctx context.Context,
	result *pagination.CursorListResult[*billingqueue.BillingQueueItem],
) (*gqlmodel.BillingQueueItemConnection, error) {
	var mapErr error
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *billingqueue.BillingQueueItem, cursor string) *gqlmodel.BillingQueueItemEdge {
			model, nodeErr := BillingQueueItemToModel(ctx, node)
			if nodeErr != nil && mapErr == nil {
				mapErr = nodeErr
			}
			return &gqlmodel.BillingQueueItemEdge{Node: model, Cursor: cursor}
		},
		func(edge *gqlmodel.BillingQueueItemEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	if mapErr != nil {
		return nil, mapErr
	}

	return &gqlmodel.BillingQueueItemConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}
