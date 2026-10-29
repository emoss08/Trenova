package workforceselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestWorkerSelectOptionMapper(t *testing.T) {
	t.Parallel()

	workerOption := workerSelectOption(&worker.Worker{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Ada",
		LastName:  "Lovelace",
		WholeName: "Ada Lovelace",
		FleetCode: &fleetcode.FleetCode{Code: "OTR"},
	})
	assert.Equal(t, "Ada Lovelace", workerOption.Label)
	assert.Equal(t, "Ada", workerOption.Meta["firstName"])
	assert.Equal(t, "OTR", workerOption.Meta["fleetCode"])
}
