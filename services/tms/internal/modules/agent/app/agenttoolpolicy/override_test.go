package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
)

func declaredRule() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          "shipment_update",
		Kind:          agenttooltrust.ToolKindAction,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agenttooltrust.ExternalReadMarked,
	}
}

func TestApplyOverride_HoldsTheToolLower(t *testing.T) {
	t.Parallel()

	got := agenttoolpolicy.ApplyOverride(new(declaredRule()), &agenttooltrust.ToolRuleOverride{
		MaxTier:       agent.TierPropose,
		ReadsExternal: agenttooltrust.ExternalReadAlways,
	})

	assert.Equal(t, agent.TierPropose, got.MaxTier)
	assert.Equal(
		t,
		agent.TierPropose,
		got.DefaultTier,
		"the default never sits above the most freedom",
	)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, got.ReadsExternal)
}

func TestApplyOverride_NeverLoosensTheDeclaredRule(t *testing.T) {
	t.Parallel()

	declared := declaredRule()
	declared.MaxTier = agent.TierActWithApproval

	got := agenttoolpolicy.ApplyOverride(&declared, &agenttooltrust.ToolRuleOverride{
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agenttooltrust.ExternalReadNever,
	})

	assert.Equal(t, agent.TierActWithApproval, got.MaxTier)
	assert.Equal(t, agenttooltrust.ExternalReadMarked, got.ReadsExternal)
}

func TestApplyOverride_LeavesTheDeclaredRuleWithoutAnOverride(t *testing.T) {
	t.Parallel()

	declared := declaredRule()

	assert.Equal(t, declared, agenttoolpolicy.ApplyOverride(&declared, nil))
	assert.Equal(
		t,
		declared,
		agenttoolpolicy.ApplyOverride(&declared, &agenttooltrust.ToolRuleOverride{Reason: "reset"}),
	)
	assert.Equal(
		t,
		declared,
		agenttoolpolicy.ApplyOverrides(&declared, map[string]*agenttooltrust.ToolRuleOverride{
			"other_tool": {MaxTier: agent.TierPropose},
		}),
	)
}
