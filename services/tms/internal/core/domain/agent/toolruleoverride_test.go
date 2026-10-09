package agent_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func validate(override *agent.ToolRuleOverride, bounds agent.ToolRuleBounds) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	override.Validate(bounds, multiErr)

	return multiErr
}

func TestToolRuleOverride_Validate(t *testing.T) {
	t.Parallel()

	changing := agent.ToolRuleBounds{
		MaxTier:       agent.TierActWithApproval,
		ReadsExternal: agent.ExternalReadMarked,
		Changes:       true,
	}

	tests := []struct {
		name     string
		override agent.ToolRuleOverride
		bounds   agent.ToolRuleBounds
		field    string
	}{
		{
			name:     "holding a tool lower is allowed",
			override: agent.ToolRuleOverride{MaxTier: agent.TierPropose},
			bounds:   changing,
		},
		{
			name:     "the declared tier itself is allowed",
			override: agent.ToolRuleOverride{MaxTier: agent.TierActWithApproval},
			bounds:   changing,
		},
		{
			name:     "a looser tier than declared is refused",
			override: agent.ToolRuleOverride{MaxTier: agent.TierAutoExecute},
			bounds:   changing,
			field:    "maxTier",
		},
		{
			name:     "an unknown tier is refused",
			override: agent.ToolRuleOverride{MaxTier: "Whenever"},
			bounds:   changing,
			field:    "maxTier",
		},
		{
			name:     "a tier on a read is refused",
			override: agent.ToolRuleOverride{MaxTier: agent.TierPropose},
			bounds:   agent.ToolRuleBounds{MaxTier: agent.TierAutoExecute},
			field:    "maxTier",
		},
		{
			name:     "treating more as outside text is allowed",
			override: agent.ToolRuleOverride{ReadsExternal: agent.ExternalReadAlways},
			bounds:   changing,
		},
		{
			name:     "treating less as outside text is refused",
			override: agent.ToolRuleOverride{ReadsExternal: agent.ExternalReadNever},
			bounds:   changing,
			field:    "readsExternal",
		},
		{
			name:     "an unknown outside text setting is refused",
			override: agent.ToolRuleOverride{ReadsExternal: "sometimes"},
			bounds:   changing,
			field:    "readsExternal",
		},
		{
			name: "a reason past the limit is refused",
			override: agent.ToolRuleOverride{
				MaxTier: agent.TierPropose,
				Reason:  strings.Repeat("a", agent.MaxToolRuleReasonLength+1),
			},
			bounds: changing,
			field:  "reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := validate(&tt.override, tt.bounds)
			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), multiErr.Error())
				return
			}
			assert.True(t, multiErr.HasErrors())
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
		})
	}
}

func TestToolRuleOverride_Empty(t *testing.T) {
	t.Parallel()

	var missing *agent.ToolRuleOverride
	assert.True(t, missing.Empty())
	assert.True(t, (&agent.ToolRuleOverride{Reason: "returned to the declared rule"}).Empty())
	assert.False(t, (&agent.ToolRuleOverride{MaxTier: agent.TierPropose}).Empty())
	assert.False(t, (&agent.ToolRuleOverride{ReadsExternal: agent.ExternalReadAlways}).Empty())
}
