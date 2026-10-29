package agentreflectionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentReflectionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ThreadReflectedThroughRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentReflectionConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type AgentReflectionRepository interface {
	Claim(
		ctx context.Context,
		entity *agentreflection.Reflection,
	) (*agentreflection.Reflection, error)
	GetByID(ctx context.Context, req GetAgentReflectionRequest) (*agentreflection.Reflection, error)
	Update(
		ctx context.Context,
		entity *agentreflection.Reflection,
	) (*agentreflection.Reflection, error)
	ThreadReflectedThrough(
		ctx context.Context,
		req ThreadReflectedThroughRequest,
	) (int, bool, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentReflectionConnectionRequest,
	) (*pagination.CursorListResult[*agentreflection.Reflection], error)
}
