package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An agent asked to remember what it already knew got a "saved" card with
// an Undo, and undoing it retired the memory the person had kept for months.
// A refreshed memory is now named as used, never offered for undo, and the
// model is told nothing new was recorded.
func TestSavedOrRefreshed_ARefreshedMemoryIsUsedNotSaved(t *testing.T) {
	t.Parallel()

	kept := &agentmemory.Memory{ID: pulid.MustNew("amem_"), Status: agentmemory.MemoryStatusActive}
	saved, refreshed := savedOrRefreshed(kept, "call_1")
	require.NotNil(t, saved)
	assert.Nil(t, refreshed)
	assert.Equal(t, kept.ID, saved.ID)
	assert.Equal(t, "call_1", saved.CallID)

	again := &agentmemory.Memory{ID: pulid.MustNew("amem_"), Refreshed: true}
	saved, refreshed = savedOrRefreshed(again, "call_2")
	assert.Nil(t, saved)
	assert.Same(t, again, refreshed)

	content := refreshedContent("agent_create_memory", again)
	assert.Contains(t, content, "already remembered as memory "+again.ID.String())
	assert.Contains(t, content, "instead of saving a duplicate")
}
