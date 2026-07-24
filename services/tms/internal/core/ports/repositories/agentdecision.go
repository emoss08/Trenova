package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentDecisionByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type AgentDecisionRepository interface {
	GetByID(ctx context.Context, req GetAgentDecisionByIDRequest) (*agent.AgentDecision, error)
	Create(ctx context.Context, entity *agent.AgentDecision) (*agent.AgentDecision, error)
}
