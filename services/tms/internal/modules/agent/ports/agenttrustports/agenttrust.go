package agenttrustports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AgentTrustService keeps the earned-autonomy ledger. Every decision on a
// proposal and every failed execution is an outcome for the agent and tool
// concerned; a long enough streak of clean approvals moves the tool up one
// tier on the agent when the organization allows it, and a setback takes an
// earned tier back.
// ToolPromotion is a tool whose clean streak has earned its agent's next
// tier: where it stands and where it goes.
type ToolPromotion struct {
	AgentDefinitionID pulid.ID           `json:"agentDefinitionId"`
	AgentName         string             `json:"agentName"`
	ToolName          string             `json:"toolName"`
	Streak            int                `json:"streak"`
	From              agent.AutonomyTier `json:"from"`
	To                agent.AutonomyTier `json:"to"`
	Approvals         int                `json:"approvals"`
	Rejections        int                `json:"rejections"`
	TrackedSince      int64              `json:"trackedSince"`
}

type PromoteReadyRequest struct {
	TenantInfo pagination.TenantInfo
	// Threshold is the streak a tool must have reached.
	Threshold int
	// DecidedBy is who turned earned autonomy on, or lowered its threshold.
	DecidedBy pulid.ID
}

type PromoteToolRequest struct {
	PromoteReadyRequest
	AgentDefinitionID pulid.ID
	ToolName          string
}

type AgentTrustService interface {
	// PromotionCandidates are the tools that would move up a tier with
	// earned autonomy on at the threshold. It changes nothing.
	PromotionCandidates(ctx context.Context, req *PromoteReadyRequest) ([]ToolPromotion, error)
	// PromoteReady moves up every tool whose streak has already reached the
	// threshold, for when earned autonomy is turned on or the threshold is
	// lowered: they earned it before there was a clean approval to promote
	// them on.
	PromoteReady(ctx context.Context, req *PromoteReadyRequest) ([]ToolPromotion, error)
	PromoteTool(ctx context.Context, req *PromoteToolRequest) (*ToolPromotion, error)
	RecordDecision(
		ctx context.Context,
		proposal *agent.AgentProposal,
		decision *agentdecision.AgentDecision,
	) error
	RecordExecutionFailure(ctx context.Context, proposal *agent.AgentProposal) error
	ListForDefinition(
		ctx context.Context,
		req agenttoolports.ListToolTrustRequest,
	) ([]*agenttooltrust.ToolTrust, error)
}
