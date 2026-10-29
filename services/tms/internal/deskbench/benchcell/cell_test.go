package benchcell

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitEffort(t *testing.T) {
	t.Parallel()

	name, effort, err := SplitEffort("Haiku 5.5@low")
	require.NoError(t, err)
	assert.Equal(t, "Haiku 5.5", name)
	assert.Equal(t, aireasoning.ReasoningLow, effort)

	name, effort, err = SplitEffort("claude-opus-5-5")
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5-5", name)
	assert.Empty(t, effort)

	_, _, err = SplitEffort("Haiku 5.5@turbo")
	require.Error(t, err)
}

func TestCellLabel_NamesTheEffort(t *testing.T) {
	t.Parallel()

	provider := &aiprovider.Provider{Name: "Haiku 5.5", Model: "claude-haiku-5-5"}
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) @medium",
		Cell{Provider: provider, Effort: aireasoning.ReasoningMedium}.Label())
	assert.Equal(t, "default", Cell{}.Label())
}

func TestCellLabel_NamesTheAdvisor(t *testing.T) {
	t.Parallel()

	provider := &aiprovider.Provider{Name: "Haiku 5.5", Model: "claude-haiku-5-5"}
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) @low +advisor", Cell{
		Provider: provider, Effort: aireasoning.ReasoningLow,
		Advisor: llmports.DefaultAdvisor(""),
	}.Label())
	assert.Equal(t, "Haiku 5.5 (claude-haiku-5-5) +advisor=claude-sonnet-5-5", Cell{
		Provider: provider, Advisor: llmports.DefaultAdvisor("claude-sonnet-5-5"),
	}.Label())
}

func TestParseCellRef_ReadsTheAdvisorSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref     string
		name    string
		effort  aireasoning.ReasoningEffort
		advisor *llmports.AdvisorSpec
	}{
		{
			ref:     "claude-haiku-5-5@low+advisor",
			name:    "claude-haiku-5-5",
			effort:  aireasoning.ReasoningLow,
			advisor: &llmports.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2},
		},
		{
			ref:     "claude-haiku-5-5+advisor",
			name:    "claude-haiku-5-5",
			advisor: &llmports.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2},
		},
		{
			ref:     " Haiku 5.5 @ medium + Advisor = claude-fable-5-1 ",
			name:    "Haiku 5.5",
			effort:  aireasoning.ReasoningMedium,
			advisor: &llmports.AdvisorSpec{Model: "claude-fable-5-1", MaxUses: 2},
		},
		{
			ref:    "claude-haiku-5-5@low",
			name:   "claude-haiku-5-5",
			effort: aireasoning.ReasoningLow,
		},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			t.Parallel()

			parsed, err := ParseRef(tc.ref)
			require.NoError(t, err)
			assert.Equal(t, tc.name, parsed.Name)
			assert.Equal(t, tc.effort, parsed.Effort)
			assert.Equal(t, tc.advisor, parsed.Advisor)
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
		_, err := ParseRef(ref)
		require.Error(t, err, ref)
	}
}

func TestCheckAdvisor_NeedsAnAnthropicProvider(t *testing.T) {
	t.Parallel()

	advisor := llmports.DefaultAdvisor("")
	anthropic := &aiprovider.Provider{Kind: aiprovider.KindAnthropicMessages}
	openai := &aiprovider.Provider{Kind: aiprovider.KindOpenAIResponses}

	require.NoError(t, CheckAdvisor(anthropic, advisor))
	require.NoError(t, CheckAdvisor(openai, nil))
	require.ErrorIs(t, CheckAdvisor(openai, advisor), errAdvisorNeedsAnthropic)
	require.ErrorIs(t, CheckAdvisor(nil, advisor), errAdvisorNeedsAnthropic,
		"the default provider cannot be checked, so it is refused")
}

func TestCheckAdvisor_NeedsAPlatformThatServesIt(t *testing.T) {
	t.Parallel()

	advisor := llmports.DefaultAdvisor("")
	for _, platform := range aiplatform.AllPlatforms() {
		provider := &aiprovider.Provider{
			Kind:     aiprovider.KindAnthropicMessages,
			Platform: platform,
		}
		err := CheckAdvisor(provider, advisor)
		if platform.ServesAdvisor() {
			require.NoError(t, err, platform)
			continue
		}
		require.ErrorContains(t, err, "does not serve the advisor tool", platform)
	}
}
