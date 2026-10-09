package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetToolRuleOverrideRequest struct {
	TenantInfo pagination.TenantInfo
	ToolName   string
}

type AgentToolRuleOverrideRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*agent.ToolRuleOverride, error)
	Get(ctx context.Context, req GetToolRuleOverrideRequest) (*agent.ToolRuleOverride, error)
	Create(ctx context.Context, override *agent.ToolRuleOverride) (*agent.ToolRuleOverride, error)
	Update(ctx context.Context, override *agent.ToolRuleOverride) (*agent.ToolRuleOverride, error)
}
