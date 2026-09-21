package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentEvaluationByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentEvaluationConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type AgentEvaluationRepository interface {
	Create(ctx context.Context, entity *agent.Evaluation) (*agent.Evaluation, error)
	Update(ctx context.Context, entity *agent.Evaluation) (*agent.Evaluation, error)
	GetByID(ctx context.Context, req GetAgentEvaluationByIDRequest) (*agent.Evaluation, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentEvaluationConnectionRequest,
	) (*pagination.CursorListResult[*agent.Evaluation], error)
}
