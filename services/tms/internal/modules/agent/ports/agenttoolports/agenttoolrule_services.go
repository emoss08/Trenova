package agenttoolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ToolRuleOverrides interface {
	For(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (map[string]*agenttooltrust.ToolRuleOverride, error)
}

type ToolRuleChange struct {
	MaxTier       agent.AutonomyTier
	ReadsExternal agenttooltrust.ExternalRead
}

type SaveToolRuleRequest struct {
	TenantInfo pagination.TenantInfo
	ToolName   string
	Change     ToolRuleChange
	Reason     string
	Version    int64
}

type ToolRuleImpactRequest struct {
	TenantInfo pagination.TenantInfo
	ToolName   string
	Change     ToolRuleChange
}

type AgentToolRuleImpact struct {
	AgentID   pulid.ID
	AgentName string
	Before    agenttooltrust.AutonomyAnswer
	After     agenttooltrust.AutonomyAnswer
}

type SaveToolRuleResult struct {
	Override *agenttooltrust.ToolRuleOverride
	Affected []AgentToolRuleImpact
}

type AgentToolRuleService interface {
	ToolRuleOverrides
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		toolName string,
	) (*agenttooltrust.ToolRuleOverride, error)
	Impact(ctx context.Context, req *ToolRuleImpactRequest) ([]AgentToolRuleImpact, error)
	Save(
		ctx context.Context,
		req *SaveToolRuleRequest,
		actor *permissionports.RequestActor,
	) (*SaveToolRuleResult, error)
}
