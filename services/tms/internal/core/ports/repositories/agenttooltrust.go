package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RecordToolTrustRequest applies one outcome to an agent's ledger row for a
// tool, creating the row when it is the first outcome for that pair.
type RecordToolTrustRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	ToolName          string
	Outcome           agent.TrustOutcome
	At                int64
}

type ListToolTrustRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
}

// MarkToolTierChangeRequest records that the ledger moved the tool to a
// tier, or took one back. A promotion starts a fresh streak; a demotion
// clears the earned tier so anything further has to be re-earned.
//
// Version is the row's version the change was decided from. The change is
// made only while the row still stands there, so of two decisions that both
// read a streak past the threshold only the later one moves the tier, and a
// setback recorded in between stops a promotion outright. A row that has
// moved on is a version mismatch.
type MarkToolTierChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	EarnedTier agent.AutonomyTier
	Promoted   bool
	At         int64
}

type AgentToolTrustRepository interface {
	Record(ctx context.Context, req RecordToolTrustRequest) (*agent.ToolTrust, error)
	ListByDefinition(ctx context.Context, req ListToolTrustRequest) ([]*agent.ToolTrust, error)
	MarkTierChange(ctx context.Context, req MarkToolTierChangeRequest) (*agent.ToolTrust, error)
}
