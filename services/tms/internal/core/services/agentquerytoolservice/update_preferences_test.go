package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/stretchr/testify/assert"
)

/*
Asked to "send the arrival notice to the customer on SEED-SHP-010 when it gets
in", Haiku read FreshHaul's "never send one on your own", whose only examples
of what the person may still ask for were a delay and a new ETA, and set no
wait. An arrival or departure notice the person asks for is theirs, not
routine; the warning says so.
*/
func TestUpdatePreferenceWarning_ANoticeThePersonAsksForStillGoesOut(t *testing.T) {
	t.Parallel()

	warning := updatePreferenceWarning(customer.StatusUpdateNone)

	assert.Contains(t, warning, "never send one on your own")
	assert.Contains(t, warning, "An arrival or departure notice the person asks for is not routine")
	assert.Empty(t, updatePreferenceWarning(customer.StatusUpdateArrivalsAndDepartures))
}
