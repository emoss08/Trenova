package agent_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedPlanTitle(t *testing.T) {
	t.Parallel()

	title, err := agent.NamedPlanTitle("  Recover S-240201\n after the\tTyson delay ")
	require.NoError(t, err)
	assert.Equal(t, "Recover S-240201 after the Tyson delay", title)

	title, err = agent.NamedPlanTitle(strings.Repeat("ü", agent.MaxNamedPlanTitleLength))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("ü", 80), title)

	title, err = agent.NamedPlanTitle("Bill\x00 the\x1b wait\u0085 now\x7f")
	require.NoError(t, err)
	assert.Equal(t, "Bill the wait now", title)

	_, err = agent.NamedPlanTitle("\x00\x07")
	require.ErrorIs(t, err, agent.ErrPlanTitleEmpty)

	_, err = agent.NamedPlanTitle(" \n\t ")
	require.ErrorIs(t, err, agent.ErrPlanTitleEmpty)

	_, err = agent.NamedPlanTitle(strings.Repeat("a", 81))
	require.ErrorIs(t, err, agent.ErrPlanTitleTooLong)
}
