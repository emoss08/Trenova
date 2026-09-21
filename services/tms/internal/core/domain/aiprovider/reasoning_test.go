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
