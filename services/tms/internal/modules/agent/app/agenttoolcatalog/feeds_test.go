package agenttoolcatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
)

// A recipe's earlier steps feed the tool it belongs to; the tool itself and
// what follows it feed nothing.
func TestFeedsNamesTheToolsARecipeStepLeadsTo(t *testing.T) {
	t.Parallel()

	catalog := New([]agentports.AgentToolDescriptor{
		{Name: "create_shipment", Recipe: []string{"find_shipment_records", "create_shipment"}},
		{
			Name:   "reschedule_stop",
			Recipe: []string{"search_shipments", "get_shipment", "reschedule_stop"},
		},
		{
			Name:   "record_stop_actual",
			Recipe: []string{"search_shipments", "get_shipment", "record_stop_actual"},
		},
		{Name: "find_shipment_records"},
	})

	assert.Equal(t, []string{"create_shipment"}, catalog.Feeds("find_shipment_records"))
	assert.ElementsMatch(
		t,
		[]string{"reschedule_stop", "record_stop_actual"},
		catalog.Feeds("get_shipment"),
	)
	assert.Empty(t, catalog.Feeds("create_shipment"))
}
