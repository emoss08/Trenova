package agentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReflectionSignals_ListsEachSignalTheLookBackRead(t *testing.T) {
	t.Parallel()

	reflection := &agentreflection.Reflection{Signals: agentreflection.ReflectionSignals{
		{
			Kind:   agentreflection.ReflectionSignalToolRecovered,
			Count:  2,
			Detail: "dispatch_assign_move",
		},
		{Kind: agentreflection.ReflectionSignalLongTask, Count: 7},
	}}

	signals := reflectionSignals(reflection)

	require.Len(t, signals, 2)
	assert.Equal(t, agentreflection.ReflectionSignalToolRecovered, signals[0].Kind)
	assert.Equal(t, 2, signals[0].Count)
	assert.Equal(t, "dispatch_assign_move", signals[0].Detail)
	assert.Equal(t, agentreflection.ReflectionSignalLongTask, signals[1].Kind)
	assert.Empty(t, reflectionSignals(&agentreflection.Reflection{}))
	assert.NotNil(t, reflectionSignals(nil))
}

func TestMemoryLinks_AreEmptyWithoutARequestLoader(t *testing.T) {
	t.Parallel()

	old := pulid.MustNew("amem_")
	memory := &agentmemory.Memory{ID: pulid.MustNew("amem_"), SupersedesID: &old}

	superseded, err := supersededMemory(t.Context(), memory)
	require.NoError(t, err)
	assert.Nil(t, superseded)

	replacement, err := replacingMemory(t.Context(), memory)
	require.NoError(t, err)
	assert.Nil(t, replacement)
}
