package agentsuggestionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordAgentSuggestionRequest struct {
	TenantInfo       pagination.TenantInfo
	SuggestedAgentID pulid.ID
	CurrentAgentID   pulid.ID
	Surface          agentsuggestion.Surface
	Outcome          agentsuggestion.Outcome
}

type SettleAgentSuggestionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Outcome    agentsuggestion.Outcome
}

type SettleAgentSuggestionRow struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	UserID     pulid.ID
	Outcome    agentsuggestion.Outcome
}

type AgentSuggestionRepository interface {
	Insert(
		ctx context.Context,
		entity *agentsuggestion.Suggestion,
	) (*agentsuggestion.Suggestion, error)
	Settle(ctx context.Context, req *SettleAgentSuggestionRow) (*agentsuggestion.Suggestion, error)
}

type AgentSuggestionService interface {
	RecordMine(
		ctx context.Context,
		req *RecordAgentSuggestionRequest,
		actor *permissionports.RequestActor,
	) (*agentsuggestion.Suggestion, error)
	SettleMine(
		ctx context.Context,
		req *SettleAgentSuggestionRequest,
		actor *permissionports.RequestActor,
	) (*agentsuggestion.Suggestion, error)
}
