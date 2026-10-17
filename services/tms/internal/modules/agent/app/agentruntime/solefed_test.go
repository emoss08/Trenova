package agentruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
find_shipment_records was loaded on its own, its answer said to call
create_shipment next, and the model went looking for create_shipment with
find_tools first. A lookup that feeds one held write brings it; one that many
held recipes start from brings none.
*/
func TestSoleFed(t *testing.T) {
	t.Parallel()

	allowed := []string{"find_shipment_records", "create_shipment", "get_shipment",
		"reschedule_stop", "record_stop_actual"}

	assert.Equal(t, "create_shipment", soleFed(allowed, []string{"create_shipment"}))
	assert.Empty(t, soleFed(allowed, []string{"reschedule_stop", "record_stop_actual"}))
	assert.Equal(
		t,
		"reschedule_stop",
		soleFed(allowed, []string{"reschedule_stop", "post_invoices"}),
		"a write the agent does not hold does not count",
	)
	assert.Empty(t, soleFed(allowed, nil))
}
