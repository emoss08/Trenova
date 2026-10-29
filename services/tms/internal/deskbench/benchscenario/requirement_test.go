package benchscenario

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequirement_DocumentTextIsCheckedWhenTheFileIsAttached(t *testing.T) {
	t.Parallel()

	needs := Requirement{DocumentText: true}
	require.NoError(t, needs.validate())
	statement, args := needs.Probe()
	assert.Equal(t, attachedTextReadySQL, statement, "no row says whether the converter runs")
	assert.Empty(t, args)
	assert.Contains(t, needs.Label(), "documentConversion")
}
