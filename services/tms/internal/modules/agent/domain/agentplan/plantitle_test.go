package agentplan_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedPlanTitle(t *testing.T) {
	t.Parallel()

	title, err := agentplan.NamedPlanTitle("  Recover S-240201\n after the\tTyson delay ")
	require.NoError(t, err)
	assert.Equal(t, "Recover S-240201 after the Tyson delay", title)

	title, err = agentplan.NamedPlanTitle(strings.Repeat("ü", agentplan.MaxNamedPlanTitleLength))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("ü", 80), title)

	title, err = agentplan.NamedPlanTitle("Bill\x00 the\x1b wait\u0085 now\x7f")
	require.NoError(t, err)
	assert.Equal(t, "Bill the wait now", title)

	_, err = agentplan.NamedPlanTitle("\x00\x07")
	require.ErrorIs(t, err, agentplan.ErrPlanTitleEmpty)

	_, err = agentplan.NamedPlanTitle(" \n\t ")
	require.ErrorIs(t, err, agentplan.ErrPlanTitleEmpty)

	_, err = agentplan.NamedPlanTitle(strings.Repeat("a", 81))
	require.ErrorIs(t, err, agentplan.ErrPlanTitleTooLong)
}
