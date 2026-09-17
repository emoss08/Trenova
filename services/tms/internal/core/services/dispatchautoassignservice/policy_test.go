package dispatchautoassignservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPolicyFor(t *testing.T) {
	t.Parallel()

	missing := policyFor(nil, true)
	assert.Equal(t, agent.TierPropose, missing.Tier)
	assert.False(t, missing.Enabled)
	assert.True(t, missing.ShadowMode, "without a dispatch agent the organization's shadow setting applies")

	disabled := policyFor(&agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Enabled:         false,
		AutonomyCeiling: agent.TierAutoExecute,
		ToolTiers:       map[string]agent.AutonomyTier{toolNameAssignMove: agent.TierAutoExecute},
	}, false)
	assert.Equal(t, agent.TierPropose, disabled.Tier,
		"a disabled dispatch agent can never act, whatever tier is stored")
	assert.False(t, disabled.Enabled)
	assert.False(t, disabled.DefinitionID.IsNil(), "runs still link to the definition that governs them")

	capped := policyFor(&agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Enabled:         true,
		AutonomyCeiling: agent.TierActWithApproval,
		ToolTiers:       map[string]agent.AutonomyTier{toolNameAssignMove: agent.TierAutoExecute},
	}, false)
	assert.Equal(t, agent.TierActWithApproval, capped.Tier,
		"the definition's ceiling caps the tool tier")

	autoExecute := policyFor(&agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Enabled:         true,
		AutonomyCeiling: agent.TierAutoExecute,
		ToolTiers:       map[string]agent.AutonomyTier{toolNameAssignMove: agent.TierAutoExecute},
	}, false)
	assert.Equal(t, agent.TierAutoExecute, autoExecute.Tier)
	assert.True(t, autoExecute.Enabled)
	assert.False(t, autoExecute.ShadowMode)

	shadowed := policyFor(&agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Enabled:         true,
		ShadowMode:      true,
		AutonomyCeiling: agent.TierAutoExecute,
	}, false)
	assert.True(t, shadowed.ShadowMode, "a definition in shadow mode stays in shadow even when the organization is live")

	orgShadow := policyFor(&agentdefinition.Definition{
		ID:      pulid.MustNew("agd_"),
		Enabled: true,
	}, true)
	assert.True(t, orgShadow.ShadowMode, "organization shadow overrides a live definition")
}
