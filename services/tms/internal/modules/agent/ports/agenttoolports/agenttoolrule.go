package agenttoolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetToolRuleOverrideRequest struct {
	TenantInfo pagination.TenantInfo
	ToolName   string
}

type AgentToolRuleOverrideRepository interface {
	List(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*agenttooltrust.ToolRuleOverride, error)
	Get(
		ctx context.Context,
		req GetToolRuleOverrideRequest,
	) (*agenttooltrust.ToolRuleOverride, error)
	Create(
		ctx context.Context,
		override *agenttooltrust.ToolRuleOverride,
	) (*agenttooltrust.ToolRuleOverride, error)
	Update(
		ctx context.Context,
		override *agenttooltrust.ToolRuleOverride,
	) (*agenttooltrust.ToolRuleOverride, error)
}
