package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLReads_EveryEntityHasItsTable(t *testing.T) {
	t.Parallel()

	for name, read := range graphQLReads {
		if read.Entity == nil {
			continue
		}
		table, ok := entityTable(read.Entity())
		require.True(t, ok, "%s reads %T, which has no keel table", name, read.Entity())
		assert.NotEmpty(t, entityFields(read.Entity()).FilterableFields,
			"%s reads %s, which filters by nothing", name, table.Name)
	}
}

func TestEntityTable_FindsTheModelsTable(t *testing.T) {
	t.Parallel()

	table, ok := entityTable(new(worker.Worker))
	require.True(t, ok)
	assert.Equal(t, "workers", table.Name)

	_, ok = entityTable(nil)
	assert.False(t, ok)
}
