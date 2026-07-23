package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type WorkspaceSummary struct {
	PeriodStart          int64     `json:"periodStart"`
	PeriodEnd            int64     `json:"periodEnd"`
	PayDate              int64     `json:"payDate"`
	DraftCount           int       `json:"draftCount"`
	PendingApprovalCount int       `json:"pendingApprovalCount"`
	ApprovedCount        int       `json:"approvedCount"`
	PostedCount          int       `json:"postedCount"`
	PaidCount            int       `json:"paidCount"`
	ExceptionCount       int       `json:"exceptionCount"`
	TotalNetMinor        int64     `json:"totalNetMinor"`
	TotalGrossMinor      int64     `json:"totalGrossMinor"`
	UnsettledEventCount  int       `json:"unsettledEventCount"`
	UnsettledGrossMinor  int64     `json:"unsettledGrossMinor"`
	HeldEventCount       int       `json:"heldEventCount"`
	HeldGrossMinor       int64     `json:"heldGrossMinor"`
	UnsettledWorkerCount int       `json:"unsettledWorkerCount"`
	OpenBatchID          *pulid.ID `json:"openBatchId"`
}

func (s *Service) GetWorkspaceSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	periodStart, periodEnd int64,
) (*WorkspaceSummary, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	bounds := PeriodBounds{PeriodStart: periodStart, PeriodEnd: periodEnd}
	if bounds.PeriodStart == 0 || bounds.PeriodEnd == 0 {
		bounds = ResolveCurrentPeriod(control, timeutils.NowUnix())
	}

	counts, err := s.settlementRepo.GetWorkspaceCounts(
		ctx,
		&repositories.GetWorkspaceCountsRequest{
			TenantInfo:  tenantInfo,
			PeriodStart: bounds.PeriodStart,
			PeriodEnd:   bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}

	unsettled, err := s.payEventRepo.GetUnsettledSummary(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	summary := &WorkspaceSummary{
		PeriodStart:          bounds.PeriodStart,
		PeriodEnd:            bounds.PeriodEnd,
		PayDate:              bounds.PayDate,
		DraftCount:           counts.DraftCount,
		PendingApprovalCount: counts.PendingApprovalCount,
		ApprovedCount:        counts.ApprovedCount,
		PostedCount:          counts.PostedCount,
		PaidCount:            counts.PaidCount,
		ExceptionCount:       counts.ExceptionCount,
		TotalNetMinor:        counts.TotalNetMinor,
		TotalGrossMinor:      counts.TotalGrossMinor,
		UnsettledEventCount:  unsettled.AccruedCount,
		UnsettledGrossMinor:  unsettled.AccruedGrossMinor,
		HeldEventCount:       unsettled.HeldCount,
		HeldGrossMinor:       unsettled.HeldGrossMinor,
		UnsettledWorkerCount: unsettled.WorkerCount,
	}

	batches, err := s.batchRepo.List(ctx, &repositories.ListSettlementBatchesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: 1},
		},
		Status: driversettlement.BatchStatusOpen,
	})
	if err == nil && len(batches.Items) > 0 {
		batchID := batches.Items[0].ID
		summary.OpenBatchID = &batchID
	}

	return summary, nil
}

func (s *Service) ListUnsettledWorkers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	periodStart, periodEnd int64,
) ([]*repositories.UnsettledWorkerSummary, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	bounds := PeriodBounds{PeriodStart: periodStart, PeriodEnd: periodEnd}
	if bounds.PeriodStart == 0 || bounds.PeriodEnd == 0 {
		bounds = ResolveCurrentPeriod(control, timeutils.NowUnix())
	}

	return s.payEventRepo.ListUnsettledWorkerSummaries(
		ctx,
		&repositories.ListUnsettledWorkerSummariesRequest{
			TenantInfo:  tenantInfo,
			PeriodStart: bounds.PeriodStart,
			PeriodEnd:   bounds.PeriodEnd,
		},
	)
}

type BulkActionType string

const (
	BulkActionSubmit   = BulkActionType("Submit")
	BulkActionApprove  = BulkActionType("Approve")
	BulkActionPost     = BulkActionType("Post")
	BulkActionMarkPaid = BulkActionType("MarkPaid")
)

func (b BulkActionType) IsValid() bool {
	switch b {
	case BulkActionSubmit, BulkActionApprove, BulkActionPost, BulkActionMarkPaid:
		return true
	default:
		return false
	}
}

type BulkActionRequest struct {
	TenantInfo       pagination.TenantInfo
	SettlementIDs    []pulid.ID
	Action           BulkActionType
	PaymentMethod    string
	PaymentReference string
}

type BulkActionResult struct {
	SettlementID pulid.ID `json:"settlementId"`
	Success      bool     `json:"success"`
	Error        string   `json:"error"`
}

type BulkActionPayload struct {
	Results      []*BulkActionResult `json:"results"`
	SuccessCount int                 `json:"successCount"`
	FailureCount int                 `json:"failureCount"`
}

func (s *Service) BulkAction(
	ctx context.Context,
	req *BulkActionRequest,
	actor *serviceports.RequestActor,
) (*BulkActionPayload, error) {
	if err := requireActor(actor, "Bulk settlement action"); err != nil {
		return nil, err
	}
	if len(req.SettlementIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"settlementIds",
			errortypes.ErrRequired,
			"Select at least one settlement",
		)
	}
	if !req.Action.IsValid() {
		return nil, errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Bulk action is invalid",
		)
	}
	if req.Action == BulkActionMarkPaid && req.PaymentMethod == "" {
		return nil, errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required to mark settlements paid",
		)
	}

	payload := &BulkActionPayload{
		Results: make([]*BulkActionResult, 0, len(req.SettlementIDs)),
	}
	for _, settlementID := range req.SettlementIDs {
		var err error
		switch req.Action {
		case BulkActionSubmit:
			_, err = s.SubmitForApproval(ctx, req.TenantInfo, settlementID, actor)
		case BulkActionApprove:
			_, err = s.approveInternal(ctx, req.TenantInfo, settlementID, actor, true)
		case BulkActionPost:
			_, err = s.Post(ctx, req.TenantInfo, settlementID, actor)
		case BulkActionMarkPaid:
			_, err = s.MarkPaid(
				ctx,
				req.TenantInfo,
				settlementID,
				req.PaymentMethod,
				req.PaymentReference,
				actor,
			)
		}

		result := &BulkActionResult{SettlementID: settlementID, Success: err == nil}
		if err != nil {
			result.Error = err.Error()
			payload.FailureCount++
		} else {
			payload.SuccessCount++
		}
		payload.Results = append(payload.Results, result)
	}
	return payload, nil
}
