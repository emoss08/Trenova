package deskmemoryresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeskMemoryToModel_CarriesWhyAndWhatItReplaced(t *testing.T) {
	t.Parallel()

	old := pulid.MustNew("amem_")
	newer := pulid.MustNew("amem_")
	model := deskMemoryToModel(&services.DeskMemory{
		Memory: &agent.Memory{
			ID:      pulid.MustNew("amem_"),
			Kind:    agent.MemoryKindProcedure,
			Source:  agent.MemorySourceReflection,
			Status:  agent.MemoryStatusActive,
			Content: "Copy dispatch and billing on rate confirmations.",
			Evidence: &agent.MemoryEvidence{
				Reason: "The person asked for billing as well.",
				Quotes: []string{"Billing needs these too"},
			},
		},
		Replaces: &services.DeskMemoryLink{
			ID:      old,
			Content: "Copy dispatch on rate confirmations.",
			Status:  agent.MemoryStatusRetired,
		},
		ReplacedBy: &services.DeskMemoryLink{
			ID:      newer,
			Content: "Copy dispatch, billing and the rep.",
			Status:  agent.MemoryStatusActive,
		},
	})

	assert.Equal(t, "The person asked for billing as well.", model.Reason)
	assert.Equal(t, []string{"Billing needs these too"}, model.Quotes)
	require.NotNil(t, model.Replaces)
	assert.Equal(t, old.String(), model.Replaces.ID)
	assert.Equal(t, agent.MemoryStatusRetired, model.Replaces.Status)
	require.NotNil(t, model.ReplacedBy)
	assert.Equal(t, newer.String(), model.ReplacedBy.ID)
}

func TestDeskMemoryToModel_AMemoryAPersonWroteHasNoWhy(t *testing.T) {
	t.Parallel()

	model := deskMemoryToModel(&services.DeskMemory{
		Memory: &agent.Memory{
			ID:      pulid.MustNew("amem_"),
			Source:  agent.MemorySourceUser,
			Content: "Group AR by facility.",
		},
	})

	assert.Empty(t, model.Reason)
	assert.NotNil(t, model.Quotes, "the list is empty, never null")
	assert.Empty(t, model.Quotes)
	assert.Nil(t, model.Replaces)
	assert.Nil(t, model.ReplacedBy)
}
