package agentdefinition_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validDefinition() *agentdefinition.Definition {
	return &agentdefinition.Definition{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Name:            "Night dispatch helper",
		Kind:            agentdefinition.KindDispatchAssistant,
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       []string{"flag_for_manual_review"},
		Enabled:         true,
	}
}

func fieldErrors(t *testing.T, d *agentdefinition.Definition) map[string]bool {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	d.Validate(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestValidate_AcceptsAWellFormedDefinition(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validDefinition().Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "unexpected errors: %v", multiErr.Errors)
}

func TestValidate_RejectsAnOverlongFocusNote(t *testing.T) {
	t.Parallel()

	// A very long focus note is usually an attempt to write a system prompt in
	// disguise, and it is sent on every turn either way.
	d := validDefinition()
	d.Focus = strings.Repeat("a", 2001)

	assert.True(t, fieldErrors(t, d)["focus"])
}

func TestValidate_RejectsToolsOnAReadOnlyKind(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.Kind = agentdefinition.KindGeneralAssistant
	d.ToolNames = []string{"correct_charge_code"}

	assert.True(t, fieldErrors(t, d)["toolNames"])
}

func TestValidate_RejectsDuplicateAndEmptyTools(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.ToolNames = []string{"a", "a", "  "}

	errs := fieldErrors(t, d)
	assert.True(t, errs["toolNames[1]"], "duplicate tool should be reported")
	assert.True(t, errs["toolNames[2]"], "empty tool name should be reported")
}

func TestValidate_RejectsTooManyTools(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.ToolNames = make([]string, 0, 33)
	for i := range 33 {
		d.ToolNames = append(d.ToolNames, string(rune('a'+i%26))+string(rune('0'+i/26)))
	}

	assert.True(t, fieldErrors(t, d)["toolNames"])
}

// The ceiling exists to restrict. A configuration that could raise a tool's tier
// would let an administrator promote a propose-only tool into one that writes on
// its own, which is precisely what the tier is there to prevent.
func TestEffectiveTier_OnlyEverRestricts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ceiling  agent.AutonomyTier
		toolTier agent.AutonomyTier
		want     agent.AutonomyTier
	}{
		{"ceiling lowers auto-execute", agent.TierPropose, agent.TierAutoExecute, agent.TierPropose},
		{
			"ceiling lowers approval tier",
			agent.TierPropose,
			agent.TierActWithApproval,
			agent.TierPropose,
		},
		{
			"ceiling cannot raise propose",
			agent.TierAutoExecute,
			agent.TierPropose,
			agent.TierPropose,
		},
		{
			"ceiling cannot raise approval tier",
			agent.TierAutoExecute,
			agent.TierActWithApproval,
			agent.TierActWithApproval,
		},
		{
			"equal tiers are unchanged",
			agent.TierActWithApproval,
			agent.TierActWithApproval,
			agent.TierActWithApproval,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := validDefinition()
			d.AutonomyCeiling = tc.ceiling

			assert.Equal(t, tc.want, d.EffectiveTier(tc.toolTier))
		})
	}
}

func TestAllowsTool(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	assert.True(t, d.AllowsTool("flag_for_manual_review"))
	assert.False(t, d.AllowsTool("correct_charge_code"))
}

func TestKindAllowedResources_NarrowsByRole(t *testing.T) {
	t.Parallel()

	// A customer assistant must not be able to hold billing tools, whatever the
	// administrator writes in the focus note.
	customer := agentdefinition.KindCustomerAssistant.AllowedResources()
	for _, resource := range customer {
		assert.NotEqual(t, "billing_queue", resource.String())
	}

	assert.True(t, agentdefinition.KindDispatchAssistant.MutatingAllowed())
	assert.False(t, agentdefinition.KindGeneralAssistant.MutatingAllowed())
	assert.Empty(t, agentdefinition.KindGeneralAssistant.AllowedResources())

	for _, kind := range agentdefinition.AllKinds() {
		assert.True(t, kind.IsValid())
		assert.NotEmpty(t, kind.Label())
		assert.NotEmpty(t, kind.Description())
	}
}
