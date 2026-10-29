package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakeToolRules struct {
	rules map[string]*agenttooltrust.ToolRuleOverride
	err   error
	asked []pagination.TenantInfo
}

func (f *fakeToolRules) For(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]*agenttooltrust.ToolRuleOverride, error) {
	f.asked = append(f.asked, tenantInfo)

	return f.rules, f.err
}

func ruledPolicy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          "shipment_update",
		Kind:          agenttooltrust.ToolKindAction,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agenttooltrust.ExternalReadNever,
	}
}

func ruledRequest() *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Actor: &permissionports.RequestActor{OrganizationID: "org_1", BusinessUnitID: "bu_1"},
	}
}

func TestRuled_AppliesTheOrganizationsRule(t *testing.T) {
	t.Parallel()

	rules := &fakeToolRules{rules: map[string]*agenttooltrust.ToolRuleOverride{
		"shipment_update": {
			MaxTier:       agent.TierActWithApproval,
			ReadsExternal: agenttooltrust.ExternalReadAlways,
		},
	}}
	s := buildRuntime(Params{
		Rules:  rules,
		Logger: zap.NewNop(),
	})

	got := s.dispatcher.Ruled(t.Context(), ruledRequest(), new(ruledPolicy()))

	assert.Equal(t, agent.TierActWithApproval, got.MaxTier)
	assert.Equal(t, agent.TierActWithApproval, got.DefaultTier)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, got.ReadsExternal)
	assert.Equal(t, []pagination.TenantInfo{{OrgID: "org_1", BuID: "bu_1"}}, rules.asked)
}

func TestRuled_HoldsTheToolToAProposalWhenRulesCannotBeRead(t *testing.T) {
	t.Parallel()

	s := buildRuntime(Params{
		Rules:  &fakeToolRules{err: errors.New("db down")},
		Logger: zap.NewNop(),
	})

	got := s.dispatcher.Ruled(t.Context(), ruledRequest(), new(ruledPolicy()))

	assert.Equal(t, agent.TierPropose, got.MaxTier)
	assert.Equal(t, agent.TierPropose, got.DefaultTier)
}

func TestRuled_KeepsTheDeclaredRuleWithoutRules(t *testing.T) {
	t.Parallel()

	s := buildRuntime(Params{
		Logger: zap.NewNop(),
	})
	assert.Equal(t, ruledPolicy(), s.dispatcher.Ruled(t.Context(), ruledRequest(), new(ruledPolicy())))

	rewire(
		s,
		func(p *Params) { p.Rules = &fakeToolRules{rules: map[string]*agenttooltrust.ToolRuleOverride{}} },
	)
	assert.Equal(t, ruledPolicy(), s.dispatcher.Ruled(t.Context(), ruledRequest(), new(ruledPolicy())))
}
