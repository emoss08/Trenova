package agentactivityports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentActivityTotalsRequest struct {
	TenantInfo     pagination.TenantInfo
	RunsSince      int64
	DecisionsSince int64
}

type AgentActivityTotals struct {
	Runs              int
	RunsFailed        int
	RunsWorking       int
	RunsAwaiting      int
	PendingProposals  int
	OldestPendingAt   *int64
	OpenExceptions    int
	DecisionsAccepted int
	DecisionsModified int
	DecisionsRejected int
}

type AgentActivityRepository interface {
	Totals(ctx context.Context, req AgentActivityTotalsRequest) (*AgentActivityTotals, error)
}
