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
	Filter     *pagination.QueryOptions `json:"filter"`
	Cursor     pagination.CursorInfo    `json:"-"`
	Columns    []string                 `json:"-"`
	SuiteRunID pulid.ID                 `json:"-"`
}

type ListSuiteEvaluationsRequest struct {
	TenantInfo   pagination.TenantInfo
	SuiteRunID   pulid.ID
	AfterOrdinal int
	Limit        int
}

type SkipPendingSuiteEvaluationsRequest struct {
	TenantInfo pagination.TenantInfo
	SuiteRunID pulid.ID
	Reason     string
	At         int64
}

type AgentEvaluationRepository interface {
	Create(ctx context.Context, entity *agent.Evaluation) (*agent.Evaluation, error)
	Update(ctx context.Context, entity *agent.Evaluation) (*agent.Evaluation, error)
	GetByID(ctx context.Context, req GetAgentEvaluationByIDRequest) (*agent.Evaluation, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentEvaluationConnectionRequest,
	) (*pagination.CursorListResult[*agent.Evaluation], error)
	CreateMany(ctx context.Context, entities []*agent.Evaluation) error
	ListBySuiteRun(
		ctx context.Context,
		req ListSuiteEvaluationsRequest,
	) ([]*agent.Evaluation, error)
	SkipPendingBySuiteRun(ctx context.Context, req SkipPendingSuiteEvaluationsRequest) (int, error)
}
