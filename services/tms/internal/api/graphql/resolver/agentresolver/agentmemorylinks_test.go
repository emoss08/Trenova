package agentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReflectionSignals_ListsEachSignalTheLookBackRead(t *testing.T) {
	t.Parallel()

	reflection := &agent.Reflection{Signals: agent.ReflectionSignals{
		{Kind: agent.ReflectionSignalToolRecovered, Count: 2, Detail: "assign_move"},
		{Kind: agent.ReflectionSignalLongTask, Count: 7},
	}}

	signals := reflectionSignals(reflection)

	require.Len(t, signals, 2)
	assert.Equal(t, agent.ReflectionSignalToolRecovered, signals[0].Kind)
	assert.Equal(t, 2, signals[0].Count)
	assert.Equal(t, "assign_move", signals[0].Detail)
	assert.Equal(t, agent.ReflectionSignalLongTask, signals[1].Kind)
	assert.Empty(t, reflectionSignals(&agent.Reflection{}))
	assert.NotNil(t, reflectionSignals(nil))
}

func TestMemoryLinks_AreEmptyWithoutARequestLoader(t *testing.T) {
	t.Parallel()

	old := pulid.MustNew("amem_")
	memory := &agent.Memory{ID: pulid.MustNew("amem_"), SupersedesID: &old}

	superseded, err := supersededMemory(t.Context(), memory)
	require.NoError(t, err)
	assert.Nil(t, superseded)

	replacement, err := replacingMemory(t.Context(), memory)
	require.NoError(t, err)
	assert.Nil(t, replacement)
}
