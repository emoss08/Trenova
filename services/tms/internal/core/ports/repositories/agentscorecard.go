package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ScorecardRequest is one agent's record over one window.
type ScorecardRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	// Since is the start of the window, in epoch seconds.
	Since int64
}

// AgentScorecardRepository counts what an agent has done.
//
// It is a read model of its own rather than a Count method on each of the
// three repositories it spans, because the answer is one screen and splitting
// it would mean a round trip per figure. Everything it returns is counted in
// SQL: a scorecard assembled by listing rows and tallying them in Go is one
// that gets slower every week the agent runs.
type AgentScorecardRepository interface {
	// Aggregate counts the runs, the proposals by tool and outcome, the
	// exceptions, and what the model cost.
	Aggregate(ctx context.Context, req ScorecardRequest) (*agent.ScorecardTotals, error)
}
