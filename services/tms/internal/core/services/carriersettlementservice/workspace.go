package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
	TotalNetMinor        int64     `json:"totalNetMinor"`
	TotalGrossMinor      int64     `json:"totalGrossMinor"`
	PendingEventCount    int       `json:"pendingEventCount"`
	PendingAmountMinor   int64     `json:"pendingAmountMinor"`
	PendingCarrierCount  int       `json:"pendingCarrierCount"`
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
		&repositories.GetCarrierSettlementWorkspaceCountsRequest{
			TenantInfo:  tenantInfo,
			PeriodStart: bounds.PeriodStart,
			PeriodEnd:   bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}

	pending, err := s.costEventRepo.GetPendingSummary(ctx, tenantInfo)
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
		TotalNetMinor:        counts.TotalNetMinor,
		TotalGrossMinor:      counts.TotalGrossMinor,
		PendingEventCount:    pending.PendingCount,
		PendingAmountMinor:   pending.PendingAmountMinor,
		PendingCarrierCount:  pending.PendingCarrierCount,
	}

	batches, err := s.batchRepo.List(ctx, &repositories.ListCarrierSettlementBatchesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: 1},
		},
		Status: carriersettlement.BatchStatusOpen,
	})
	if err == nil && len(batches.Items) > 0 {
		batchID := batches.Items[0].ID
		summary.OpenBatchID = &batchID
	}

	return summary, nil
}

func (s *Service) GetCurrentPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PeriodBounds, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	bounds := ResolveCurrentPeriod(control, timeutils.NowUnix())
	return &bounds, nil
}
