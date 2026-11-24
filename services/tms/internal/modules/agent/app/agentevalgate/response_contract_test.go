package agentevalgate_test

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/recordquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryListOrGetReadAnswersInTheEnvelope(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, descriptor := range kit.Queries.Descriptors() {
		parts := strings.SplitN(descriptor.Name, "_", 3)
		if len(parts) < 2 || (parts[1] != "list" && parts[1] != "get") {
			continue
		}
		if descriptor.Name == agentcapability.RecordGet {
			continue
		}
		_, projection := agentcapability.Projection(descriptor.Name)
		assert.True(t, projection,
			"%s reads records outside records_query and record_get; make it an entity's reader "+
				"(agentcapability entities) or declare it a projection with its reason", descriptor.Name)
	}
	for _, name := range []string{agentcapability.RecordsQuery, agentcapability.RecordGet, agentcapability.DescribeEntity} {
		_, ok := kit.Queries.Get(name)
		assert.True(t, ok, "%s is registered", name)
	}
}

func TestTheEnvelopeAlwaysCarriesItsContract(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(&recordquerytools.Envelope{Entity: "shipment", AppliedFilters: []recordquerytools.Applied{}})
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))
	for _, key := range []string{"entity", "items", "hasMore", "appliedFilters"} {
		assert.Contains(t, decoded, key)
	}
}

func TestEveryEntityReaderAndProjectionExists(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, name := range agentcapability.Backings() {
		_, ok := kit.Queries.Get(name)
		assert.True(t, ok, "%s backs an entity but is not registered", name)
	}
}
