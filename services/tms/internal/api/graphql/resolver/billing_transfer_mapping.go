package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
)

func billingTransferRunToModel(
	run *billingtransfer.BillingTransferRun,
) *gqlmodel.BillingTransferRun {
	if run == nil {
		return nil
	}

	model := &gqlmodel.BillingTransferRun{
		ID:                          run.ID.String(),
		Status:                      run.Status,
		Scope:                       run.Scope,
		BillType:                    run.BillType,
		MarkCompletedReadyToInvoice: run.MarkCompletedReadyToInvoice,
		RequestedByID:               run.RequestedByID.String(),
		TotalCount:                  run.TotalCount,
		ProcessedCount:              run.ProcessedCount,
		TransferredCount:            run.TransferredCount,
		NotTransferredCount:         run.NotTransferredCount,
		SkippedCount:                run.SkippedCount,
		MarkedReadyToInvoiceCount:   run.MarkedReadyToInvoiceCount,
		RetryableCount:              run.RetryableCount,
		UnmatchedCount:              run.UnmatchedCount,
		QueuedAt:                    int(run.QueuedAt),
		StartedAt:                   unixPtr(run.StartedAt),
		CompletedAt:                 unixPtr(run.CompletedAt),
		CreatedAt:                   int(run.CreatedAt),
		UpdatedAt:                   int(run.UpdatedAt),
		CancelRequestedAt:           unixPtr(run.CancelRequestedAt),
	}

	if run.SearchQuery != "" {
		model.SearchQuery = &run.SearchQuery
	}
	if run.ShipmentStatus != "" {
		status := gqlmodel.ShipmentStatus(run.ShipmentStatus)
		model.ShipmentStatus = &status
	}
	if !run.SourceRunID.IsNil() {
		sourceRunID := run.SourceRunID.String()
		model.SourceRunID = &sourceRunID
	}
	if run.FailureMessage != "" {
		model.FailureMessage = &run.FailureMessage
	}
	if !run.CancelRequestedByID.IsNil() {
		cancelRequestedByID := run.CancelRequestedByID.String()
		model.CancelRequestedByID = &cancelRequestedByID
	}

	return model
}

func billingTransferRunItemToModel(
	item *billingtransfer.BillingTransferRunItem,
) *gqlmodel.BillingTransferRunItem {
	if item == nil {
		return nil
	}

	model := &gqlmodel.BillingTransferRunItem{
		ID:                   item.ID.String(),
		RunID:                item.RunID.String(),
		ShipmentID:           item.ShipmentID.String(),
		Sequence:             item.Sequence,
		Status:               item.Status,
		MarkedReadyToInvoice: item.MarkedReadyToInvoice,
		MissingRequirements:  make([]*billingtransfer.MissingRequirement, 0, len(item.MissingRequirements)),
		ValidationFailures:   make([]*billingtransfer.ValidationFailure, 0, len(item.ValidationFailures)),
		ProcessedAt:          unixPtr(item.ProcessedAt),
		CreatedAt:            int(item.CreatedAt),
	}

	if item.ProNumber != "" {
		model.ProNumber = &item.ProNumber
	}
	if item.FailureCode != "" {
		failureCode := item.FailureCode
		model.FailureCode = &failureCode
	}
	if item.ErrorMessage != "" {
		model.ErrorMessage = &item.ErrorMessage
	}
	if !item.BillingQueueItemID.IsNil() {
		billingQueueItemID := item.BillingQueueItemID.String()
		model.BillingQueueItemID = &billingQueueItemID
	}
	if item.BillingQueueNumber != "" {
		model.BillingQueueNumber = &item.BillingQueueNumber
	}
	if item.BillingQueueStatus != "" {
		billingQueueStatus := item.BillingQueueStatus
		model.BillingQueueStatus = &billingQueueStatus
	}

	for i := range item.MissingRequirements {
		model.MissingRequirements = append(model.MissingRequirements, &item.MissingRequirements[i])
	}
	for i := range item.ValidationFailures {
		model.ValidationFailures = append(model.ValidationFailures, &item.ValidationFailures[i])
	}

	return model
}

func billingTransferRunItemConnectionToModel(
	result *pagination.CursorListResult[*billingtransfer.BillingTransferRunItem],
) (*gqlmodel.BillingTransferRunItemConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *billingtransfer.BillingTransferRunItem, cursor string) *gqlmodel.BillingTransferRunItemEdge {
			return &gqlmodel.BillingTransferRunItemEdge{
				Node:   billingTransferRunItemToModel(node),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.BillingTransferRunItemEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.BillingTransferRunItemConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: derefInt(page.TotalCount),
	}, nil
}

// unixPtr converts an optional epoch second to the pointer shape the Timestamp
// scalar maps to. An unset instant stays unset rather than becoming 1970.
func unixPtr(v *int64) *int {
	if v == nil || *v == 0 {
		return nil
	}

	out := int(*v)

	return &out
}

// billingTransferShipmentStatus narrows the optional status filter a run was
// started with. An absent status means "either of the two a shipment can be
// transferred from", which the candidate query already enforces.
func billingTransferShipmentStatus(status *gqlmodel.ShipmentStatus) shipment.Status {
	if status == nil {
		return ""
	}

	return shipment.Status(*status)
}

func billingTransferBillType(billType *billingqueue.BillType) billingqueue.BillType {
	if billType == nil {
		return billingqueue.BillTypeInvoice
	}

	return *billType
}
