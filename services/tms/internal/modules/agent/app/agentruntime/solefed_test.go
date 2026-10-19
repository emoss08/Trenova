package agentruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
shipment_match_records was loaded on its own, its answer said to call
shipment_create next, and the model went looking for shipment_create with
find_tools first. A lookup that feeds one held write brings it; one that many
held recipes start from brings none.
*/
func TestSoleFed(t *testing.T) {
	t.Parallel()

	allowed := []string{"shipment_match_records", "shipment_create", "shipment_get",
		"shipment_reschedule_stop", "dispatch_record_stop_actual"}

	assert.Equal(t, "shipment_create", soleFed(allowed, []string{"shipment_create"}))
	assert.Empty(t, soleFed(allowed, []string{"shipment_reschedule_stop", "dispatch_record_stop_actual"}))
	assert.Equal(
		t,
		"shipment_reschedule_stop",
		soleFed(allowed, []string{"shipment_reschedule_stop", "invoice_post"}),
		"a write the agent does not hold does not count",
	)
	assert.Empty(t, soleFed(allowed, nil))
}
