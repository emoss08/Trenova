package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
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
