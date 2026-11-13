package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/shared/pulid"
)

func TestHandTo_StartsTheExceptionAndRefusesAResolvedOne(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.NoError(t, item.Snooze(testNow+600, testNow))
	require.NoError(
		t,
		item.HandTo(pulid.MustNew("adef_"), pulid.MustNew("ar_"), pulid.Nil, testNow),
	)
	assert.Equal(t, StatusInProgress, item.Status)
	assert.Nil(t, item.SnoozedUntil)
	require.NotNil(t, item.AcknowledgedAt)

	resolved := openItem()
	require.NoError(t, resolved.Resolve(ResolutionFixed, "", pulid.Nil, testNow))
	require.Error(
		t,
		resolved.HandTo(pulid.MustNew("adef_"), pulid.MustNew("ar_"), pulid.Nil, testNow),
	)
}

func TestHasAgent_NeedsBothTheAgentAndItsRun(t *testing.T) {
	t.Parallel()

	item := openItem()
	assert.False(t, item.HasAgent())

	item.AgentDefinitionID = pulid.MustNew("adef_")
	assert.False(t, item.HasAgent(), "an agent with no run has not been handed it")

	item.AgentRunID = pulid.MustNew("ar_")
	assert.True(t, item.HasAgent())
}
