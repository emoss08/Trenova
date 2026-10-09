package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ToolRuleOverrides interface {
	For(ctx context.Context, tenantInfo pagination.TenantInfo) (map[string]*agent.ToolRuleOverride, error)
}

type ToolRuleChange struct {
	MaxTier       agent.AutonomyTier
	ReadsExternal agent.ExternalRead
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
	Before    agent.AutonomyAnswer
	After     agent.AutonomyAnswer
}

type SaveToolRuleResult struct {
	Override *agent.ToolRuleOverride
	Affected []AgentToolRuleImpact
}

type AgentToolRuleService interface {
	ToolRuleOverrides
	Get(ctx context.Context, tenantInfo pagination.TenantInfo, toolName string) (*agent.ToolRuleOverride, error)
	Impact(ctx context.Context, req *ToolRuleImpactRequest) ([]AgentToolRuleImpact, error)
	Save(ctx context.Context, req *SaveToolRuleRequest, actor *RequestActor) (*SaveToolRuleResult, error)
}
