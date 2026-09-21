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
type MarkToolTierChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	EarnedTier agent.AutonomyTier
	Promoted   bool
	At         int64
}

type AgentToolTrustRepository interface {
	Record(ctx context.Context, req RecordToolTrustRequest) (*agent.ToolTrust, error)
	ListByDefinition(ctx context.Context, req ListToolTrustRequest) ([]*agent.ToolTrust, error)
	MarkTierChange(ctx context.Context, req MarkToolTierChangeRequest) (*agent.ToolTrust, error)
}
