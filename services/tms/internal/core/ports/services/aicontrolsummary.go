package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AIControlSummaryRequest struct {
	TenantInfo pagination.TenantInfo
	Tab        aicontrolsummary.Tab
}

type VisibleFailuresRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Failing    []aicontrolsummary.ProviderFailure
}

type ProviderFailureDismissal struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ProviderID pulid.ID
	// FailureAt is the last failure the person saw.
	FailureAt int64
}

// AIControlSummaryService writes the sentence heading each tab of AI control,
// and keeps which failing providers each person has put away.
type AIControlSummaryService interface {
	Summary(ctx context.Context, req *AIControlSummaryRequest) (*aicontrolsummary.Summary, error)
	VisibleFailures(
		ctx context.Context,
		req *VisibleFailuresRequest,
	) ([]aicontrolsummary.ProviderFailure, error)
	DismissFailure(ctx context.Context, req *ProviderFailureDismissal) error
	RestoreFailure(ctx context.Context, req *ProviderFailureDismissal) error
}
