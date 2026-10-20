package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitEffort(t *testing.T) {
	t.Parallel()

	name, effort, err := splitEffort("Haiku 5.5@low")
	require.NoError(t, err)
	assert.Equal(t, "Haiku 5.5", name)
	assert.Equal(t, aiprovider.ReasoningLow, effort)

	name, effort, err = splitEffort("claude-opus-5-5")
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5-5", name)
	assert.Empty(t, effort)

	_, _, err = splitEffort("Haiku 5.5@turbo")
	require.Error(t, err)
}

func TestCellLabel_NamesTheEffort(t *testing.T) {
	t.Parallel()

	provider := &aiprovider.Provider{Name: "Haiku 5.5", Model: "claude-haiku-5-5"}
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) @medium",
		Cell{Provider: provider, Effort: aiprovider.ReasoningMedium}.Label())
	assert.Equal(t, "default", Cell{}.Label())
}

func TestCellLabel_NamesTheAdvisor(t *testing.T) {
	t.Parallel()

	provider := &aiprovider.Provider{Name: "Haiku 5.5", Model: "claude-haiku-5-5"}
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) @low +advisor", Cell{
		Provider: provider, Effort: aiprovider.ReasoningLow,
		Advisor: agentports.DefaultAdvisor(""),
	}.Label())
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) +advisor=claude-sonnet-5-5", Cell{
		Provider: provider, Advisor: agentports.DefaultAdvisor("claude-sonnet-5-5"),
	}.Label())
}

func TestParseCellRef_ReadsTheAdvisorSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref     string
		name    string
		effort  aiprovider.ReasoningEffort
		advisor *agentports.AdvisorSpec
	}{
		{
			ref:     "claude-haiku-5-5@low+advisor",
			name:    "claude-haiku-5-5",
			effort:  aiprovider.ReasoningLow,
			advisor: &agentports.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2},
		},
		{
			ref:     "claude-haiku-5-5+advisor",
			name:    "claude-haiku-5-5",
			advisor: &agentports.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2},
		},
		{
			ref:     " Haiku 5.5 @ medium + Advisor = claude-fable-5-1 ",
			name:    "Haiku 5.5",
			effort:  aiprovider.ReasoningMedium,
			advisor: &agentports.AdvisorSpec{Model: "claude-fable-5-1", MaxUses: 2},
		},
		{
			ref:    "claude-haiku-5-5@low",
			name:   "claude-haiku-5-5",
			effort: aiprovider.ReasoningLow,
		},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			t.Parallel()

			parsed, err := parseCellRef(tc.ref)
			require.NoError(t, err)
			assert.Equal(t, tc.name, parsed.name)
			assert.Equal(t, tc.effort, parsed.effort)
			assert.Equal(t, tc.advisor, parsed.advisor)
		})
	}
}

func TestParseCellRef_RefusesWhatItDoesNotKnow(t *testing.T) {
	t.Parallel()

	for _, ref := range []string{
		"claude-haiku-5-5@low+code",
		"claude-haiku-5-5@low+advisor+advisor",
		"claude-haiku-5-5@low+advisor=",
		"claude-haiku-5-5@low+",
		"claude-haiku-5-5@turbo+advisor",
	} {
		_, err := parseCellRef(ref)
		require.Error(t, err, ref)
	}
}

func TestCheckAdvisor_NeedsAnAnthropicProvider(t *testing.T) {
	t.Parallel()

	advisor := agentports.DefaultAdvisor("")
	anthropic := &aiprovider.Provider{Kind: aiprovider.KindAnthropicMessages}
	openai := &aiprovider.Provider{Kind: aiprovider.KindOpenAIResponses}

	require.NoError(t, checkAdvisor(anthropic, advisor))
	require.NoError(t, checkAdvisor(openai, nil))
	require.ErrorIs(t, checkAdvisor(openai, advisor), errAdvisorNeedsAnthropic)
	require.ErrorIs(t, checkAdvisor(nil, advisor), errAdvisorNeedsAnthropic,
		"the default provider cannot be checked, so it is refused")
}
