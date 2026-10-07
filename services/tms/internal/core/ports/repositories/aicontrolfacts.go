package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentroster"
	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentCountsRequest struct {
	TenantInfo pagination.TenantInfo
	// ShadowSince is the start of the window shadow recordings are counted in.
	ShadowSince int64
}

type AgentRosterRequest struct {
	TenantInfo     pagination.TenantInfo
	RunsSince      int64
	DecisionsSince int64
}

type ProviderFailuresRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

// ProviderFailureCount is one provider whose most recent call in a window
// failed: how many of its calls failed, and when the last one did.
type ProviderFailureCount struct {
	ProviderID    pulid.ID
	FailedCalls   int
	LastFailureAt int64
}

// AIControlFactsRepository reads the counts AI control's sentences are made
// of, each a count rather than rows.
type AIControlFactsRepository interface {
	AgentCounts(ctx context.Context, req *AgentCountsRequest) (aicontrolsummary.AgentCounts, error)
	ProviderFailures(ctx context.Context, req *ProviderFailuresRequest) ([]ProviderFailureCount, error)
	AgentRoster(ctx context.Context, req *AgentRosterRequest) ([]*agentroster.Stat, error)
}
