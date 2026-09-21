package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ReplayAgentRunRequest asks for a recorded run to be replayed against its
// agent as it is now.
type ReplayAgentRunRequest struct {
	RunID      pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentEvaluationService interface {
	Replay(
		ctx context.Context,
		req *ReplayAgentRunRequest,
		actor *RequestActor,
	) (*agent.Evaluation, error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentEvaluationByIDRequest,
	) (*agent.Evaluation, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentEvaluationConnectionRequest,
	) (*pagination.CursorListResult[*agent.Evaluation], error)
}
