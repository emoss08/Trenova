package memorynotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryNote_CarriesWhyAndWhatItReplaced(t *testing.T) {
	t.Parallel()

	old := pulid.MustNew("amem_")
	note := memoryNote(&agentmemoryports.DeskMemory{
		Memory: &agentmemory.Memory{
			ID:      pulid.MustNew("amem_"),
			Kind:    agentmemory.MemoryKindProcedure,
			Source:  agentmemory.MemorySourceReflection,
			Status:  agentmemory.MemoryStatusActive,
			Scope:   agentmemory.MemoryScopeAgent,
			Content: "Read the move before assigning it.",
			Evidence: &agentmemory.MemoryEvidence{
				Reason: "Assigning failed until the move was read.",
			},
		},
		Replaces: &agentmemoryports.DeskMemoryLink{
			ID:      old,
			Content: "Assign the move straight away.",
			Status:  agentmemory.MemoryStatusRetired,
		},
	})

	assert.Equal(t, "Assigning failed until the move was read.", note.Reason)
	require.NotNil(t, note.Replaces)
	assert.Equal(t, old, note.Replaces.ID)
	assert.Equal(t, "Assign the move straight away.", note.Replaces.Content)
	assert.Equal(t, "Retired", note.Replaces.Status)
	assert.Nil(t, note.ReplacedBy)
}

func TestMemoryNote_AMemoryWithoutEvidenceHasNoWhy(t *testing.T) {
	t.Parallel()

	note := memoryNote(&agentmemoryports.DeskMemory{
		Memory: &agentmemory.Memory{
			ID:     pulid.MustNew("amem_"),
			Source: agentmemory.MemorySourceUser,
		},
	})

	assert.Empty(t, note.Reason)
	assert.Nil(t, note.Replaces)
	assert.Nil(t, note.ReplacedBy)
}
