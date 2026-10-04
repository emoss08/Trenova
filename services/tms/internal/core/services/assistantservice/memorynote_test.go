package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryNote_CarriesWhyAndWhatItReplaced(t *testing.T) {
	t.Parallel()

	old := pulid.MustNew("amem_")
	note := memoryNote(&serviceports.DeskMemory{
		Memory: &agent.Memory{
			ID:       pulid.MustNew("amem_"),
			Kind:     agent.MemoryKindProcedure,
			Source:   agent.MemorySourceReflection,
			Status:   agent.MemoryStatusActive,
			Scope:    agent.MemoryScopeAgent,
			Content:  "Read the move before assigning it.",
			Evidence: &agent.MemoryEvidence{Reason: "Assigning failed until the move was read."},
		},
		Replaces: &serviceports.DeskMemoryLink{
			ID:      old,
			Content: "Assign the move straight away.",
			Status:  agent.MemoryStatusRetired,
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

	note := memoryNote(&serviceports.DeskMemory{
		Memory: &agent.Memory{ID: pulid.MustNew("amem_"), Source: agent.MemorySourceUser},
	})

	assert.Empty(t, note.Reason)
	assert.Nil(t, note.Replaces)
	assert.Nil(t, note.ReplacedBy)
}
