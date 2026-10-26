package agenttoolruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type LoaderParams struct {
	fx.In

	Repo agentports.AgentToolRuleOverrideRepository
}

type Loader struct {
	repo agentports.AgentToolRuleOverrideRepository
}

var _ agentports.ToolRuleOverrides = (*Loader)(nil)

func NewLoader(p LoaderParams) *Loader {
	return newLoader(p.Repo)
}

func newLoader(repo agentports.AgentToolRuleOverrideRepository) *Loader {
	return &Loader{repo: repo}
}

func (l *Loader) For(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]*agent.ToolRuleOverride, error) {
	overrides, err := l.repo.List(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	rules := make(map[string]*agent.ToolRuleOverride, len(overrides))
	for _, override := range overrides {
		rules[override.ToolName] = override
	}

	return rules, nil
}
