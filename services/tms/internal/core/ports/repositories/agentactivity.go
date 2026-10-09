package repositories

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
	Runs              int    `bun:"runs"`
	RunsFailed        int    `bun:"runs_failed"`
	RunsWorking       int    `bun:"runs_working"`
	RunsAwaiting      int    `bun:"runs_awaiting"`
	PendingProposals  int    `bun:"pending_proposals"`
	OldestPendingAt   *int64 `bun:"oldest_pending_at"`
	OpenExceptions    int    `bun:"open_exceptions"`
	DecisionsAccepted int    `bun:"decisions_accepted"`
	DecisionsModified int    `bun:"decisions_modified"`
	DecisionsRejected int    `bun:"decisions_rejected"`
}

type AgentActivityRepository interface {
	Totals(ctx context.Context, req AgentActivityTotalsRequest) (*AgentActivityTotals, error)
}
