package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentExceptionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetAgentExceptionByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type UpdateAgentExceptionResolutionRequest struct {
	ID              pulid.ID              `json:"id"`
	ResolutionState agent.ResolutionState `json:"resolutionState"`
	ResolutionNotes string                `json:"resolutionNotes"`
	TenantInfo      pagination.TenantInfo `json:"-"`
}

type AgentExceptionRepository interface {
	List(
		ctx context.Context,
		req *ListAgentExceptionRequest,
	) (*pagination.ListResult[*agent.AgentException], error)
	GetByID(ctx context.Context, req GetAgentExceptionByIDRequest) (*agent.AgentException, error)
	Create(ctx context.Context, entity *agent.AgentException) (*agent.AgentException, error)
	UpdateResolution(
		ctx context.Context,
		req UpdateAgentExceptionResolutionRequest,
	) (*agent.AgentException, error)
}
