package agentplanports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// DecideAgentPlanRequest decides every pending step of a plan at once.
type DecideAgentPlanRequest struct {
	PlanID     pulid.ID
	Decision   agentdecision.DecisionType
	ReasonCode string
	Note       string
	TenantInfo pagination.TenantInfo
	// PreviewDigest is the digest of the plan preview the decider was shown.
	PreviewDigest string
}

type AgentPlanService interface {
	Decide(
		ctx context.Context,
		req *DecideAgentPlanRequest,
		actor *permissionports.RequestActor,
	) (*agentplan.AgentPlan, error)
	// DecideOwn decides a plan raised in one of the actor's own
	// conversations, by an agent they may use. Someone else's is not found.
	DecideOwn(
		ctx context.Context,
		req *DecideAgentPlanRequest,
		actor *permissionports.RequestActor,
	) (*agentplan.AgentPlan, error)
	GetByID(ctx context.Context, req GetAgentPlanByIDRequest) (*agentplan.AgentPlan, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentPlanConnectionRequest,
	) (*pagination.CursorListResult[*agentplan.AgentPlan], error)
	PlanHolds(
		ctx context.Context,
		tenant pagination.TenantInfo,
		plans []*agentplan.AgentPlan,
	) (map[pulid.ID]*assistantproposalports.ProposalHold, error)
	// AssertOwnPlan refuses a plan not raised in one of the actor's own
	// conversations, or by an agent they may no longer use, as not found.
	AssertOwnPlan(
		ctx context.Context,
		planID pulid.ID,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
	) error
	agentapprovalports.ApprovalCommitter
}
