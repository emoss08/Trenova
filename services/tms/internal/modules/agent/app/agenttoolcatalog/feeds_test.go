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
		{Name: "shipment_create", Recipe: []string{"shipment_match_records", "shipment_create"}},
		{
			Name:   "shipment_reschedule_stop",
			Recipe: []string{"shipment_list", "shipment_get", "shipment_reschedule_stop"},
		},
		{
			Name:   "dispatch_record_stop_actual",
			Recipe: []string{"shipment_list", "shipment_get", "dispatch_record_stop_actual"},
		},
		{Name: "shipment_match_records"},
	})

	assert.Equal(t, []string{"shipment_create"}, catalog.Feeds("shipment_match_records"))
	assert.ElementsMatch(
		t,
		[]string{"shipment_reschedule_stop", "dispatch_record_stop_actual"},
		catalog.Feeds("shipment_get"),
	)
	assert.Empty(t, catalog.Feeds("shipment_create"))
}
