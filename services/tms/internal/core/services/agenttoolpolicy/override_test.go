package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
)

func declaredRule() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          "update_shipment",
		Kind:          agent.ToolKindAction,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agent.ExternalReadMarked,
	}
}

func TestApplyOverride_HoldsTheToolLower(t *testing.T) {
	t.Parallel()

	got := agenttoolpolicy.ApplyOverride(declaredRule(), &agent.ToolRuleOverride{
		MaxTier:       agent.TierPropose,
		ReadsExternal: agent.ExternalReadAlways,
	})

	assert.Equal(t, agent.TierPropose, got.MaxTier)
	assert.Equal(t, agent.TierPropose, got.DefaultTier, "the default never sits above the most freedom")
	assert.Equal(t, agent.ExternalReadAlways, got.ReadsExternal)
}

func TestApplyOverride_NeverLoosensTheDeclaredRule(t *testing.T) {
	t.Parallel()

	declared := declaredRule()
	declared.MaxTier = agent.TierActWithApproval

	got := agenttoolpolicy.ApplyOverride(declared, &agent.ToolRuleOverride{
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agent.ExternalReadNever,
	})

	assert.Equal(t, agent.TierActWithApproval, got.MaxTier)
	assert.Equal(t, agent.ExternalReadMarked, got.ReadsExternal)
}

func TestApplyOverride_LeavesTheDeclaredRuleWithoutAnOverride(t *testing.T) {
	t.Parallel()

	declared := declaredRule()

	assert.Equal(t, declared, agenttoolpolicy.ApplyOverride(declared, nil))
	assert.Equal(t, declared, agenttoolpolicy.ApplyOverride(declared, &agent.ToolRuleOverride{Reason: "reset"}))
	assert.Equal(t, declared, agenttoolpolicy.ApplyOverrides(declared, map[string]*agent.ToolRuleOverride{
		"other_tool": {MaxTier: agent.TierPropose},
	}))
}
