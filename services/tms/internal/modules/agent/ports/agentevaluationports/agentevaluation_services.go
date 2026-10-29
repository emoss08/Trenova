package agentevaluationports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ReplayAgentRunRequest asks for a recorded run to be replayed against its
// agent as it is now.
type ReplayAgentRunRequest struct {
	RunID      pulid.ID
	TenantInfo pagination.TenantInfo
}

type ReplayAgentEvalCaseRequest struct {
	CaseID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentEvaluationService interface {
	ReplayCase(
		ctx context.Context,
		req *ReplayAgentEvalCaseRequest,
		actor *permissionports.RequestActor,
	) (*agentevaluation.Evaluation, error)
	Replay(
		ctx context.Context,
		req *ReplayAgentRunRequest,
		actor *permissionports.RequestActor,
	) (*agentevaluation.Evaluation, error)
	GetByID(
		ctx context.Context,
		req GetAgentEvaluationByIDRequest,
	) (*agentevaluation.Evaluation, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentEvaluationConnectionRequest,
	) (*pagination.CursorListResult[*agentevaluation.Evaluation], error)
}
