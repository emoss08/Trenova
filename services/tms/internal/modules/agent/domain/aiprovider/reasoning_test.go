package aiprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Off sends nothing, because the reasoning parameter is refused by models
// without it; the other levels map to what each protocol takes.
func TestReasoningEffort(t *testing.T) {
	t.Parallel()

	assert.False(t, ReasoningOff.Enabled())
	assert.Empty(t, ReasoningOff.Wire())
	assert.Zero(t, ReasoningOff.ThinkingBudget())

	assert.True(t, ReasoningHigh.Enabled())
	assert.Equal(t, "high", ReasoningHigh.Wire())
	assert.Equal(t, 16384, ReasoningHigh.ThinkingBudget())
	assert.Equal(t, 1024, ReasoningLow.ThinkingBudget(), "the API's minimum")

	assert.False(t, ReasoningEffort("Max").IsValid())
	assert.False(t, ReasoningEffort("Max").Enabled(), "an unknown value is never sent")
}

// A model that reasons by default — the GPT-5 family, a thinking Ollama
// model — keeps reasoning at its own default when nothing is sent, so Off
// cannot turn it off. None says so explicitly; Minimal asks for the least
// reasoning the model allows.
func TestReasoningEffort_NoneAndMinimalAreSentExplicitly(t *testing.T) {
	t.Parallel()

	assert.True(t, ReasoningNone.IsValid())
	assert.False(t, ReasoningNone.Enabled())
	assert.Equal(t, "none", ReasoningNone.Wire())
	assert.Zero(t, ReasoningNone.ThinkingBudget())
	assert.True(t, ReasoningNone.Disabled())
	assert.False(t, ReasoningOff.Disabled(), "Off sends nothing, so it disables nothing")

	assert.True(t, ReasoningMinimal.IsValid())
	assert.True(t, ReasoningMinimal.Enabled())
	assert.Equal(t, "minimal", ReasoningMinimal.Wire())
	assert.Equal(t, 1024, ReasoningMinimal.ThinkingBudget(), "the API's minimum")
	assert.False(t, ReasoningMinimal.Disabled())
}
