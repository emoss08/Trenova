package memorytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ agenttoolports.ToolResultReporter = (*RememberTool)(nil)
	_ agenttoolports.ToolResultReporter = (*EditMemoryTool)(nil)
)

/*
An approved agent_create_memory reported only "it ran". The follow-up told the
person the memory still needed an administrator's approval, which a shared
memory saved by someone who may not manage shared memory does, and one saved by
someone who may does not. The result now says which.
*/
func TestMemoryResult_SaysWhetherTheMemoryTookEffect(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("amem_")
	kept := memoryResult(&agentmemory.Memory{
		ID:      id,
		Content: "Plan Delayed shipments in batches of 50.",
		Scope:   agentmemory.MemoryScopeRole,
		Status:  agentmemory.MemoryStatusActive,
	})
	require.NotNil(t, kept)
	assert.Equal(t, map[string]string{"memoryId": id.String()}, kept.IDs)
	assert.Equal(t,
		`It saved the memory "Plan Delayed shipments in batches of 50.". Now: kept and in `+
			`effect, read in the later conversations of everyone in this person's role.`,
		kept.Describe())

	offered := memoryResult(&agentmemory.Memory{
		ID:      id,
		Content: "Plan Delayed shipments in batches of 50.",
		Scope:   agentmemory.MemoryScopeOrganization,
		Status:  agentmemory.MemoryStatusSuggested,
	})
	assert.Equal(t, "offered", offered.Action)
	assert.Contains(t, offered.Describe(), "offered for an administrator to accept")

	mine := memoryResult(&agentmemory.Memory{
		ID:      id,
		Content: "I prefer loads under 500 miles.",
		Scope:   agentmemory.MemoryScopeUser,
		Status:  agentmemory.MemoryStatusSuggested,
	})
	assert.Contains(t, mine.Describe(), "offered for this person to accept")

	refreshed := memoryResult(&agentmemory.Memory{
		ID:        id,
		Content:   "I prefer loads under 500 miles.",
		Scope:     agentmemory.MemoryScopeUser,
		Status:    agentmemory.MemoryStatusActive,
		Refreshed: true,
	})
	assert.Equal(t, "refreshed", refreshed.Action)
	assert.Contains(t, refreshed.Describe(), "nothing new was recorded")

	assert.Nil(t, memoryResult(nil))
}
