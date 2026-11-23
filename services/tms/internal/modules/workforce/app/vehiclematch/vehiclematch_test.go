package vehiclematch_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/app/vehiclematch"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestMatch_ByVINThenCodeOncePerUnitAndVehicle(t *testing.T) {
	t.Parallel()
	byVIN, byCode, mapped := pulid.MustNew("tr_"), pulid.MustNew("tr_"), pulid.MustNew("tr_")
	unmatched := 0

	got := vehiclematch.Match(
		[]vehiclematch.Candidate{
			{UnitID: byVIN, VIN: " 1hgcm82633a004352 "},
			{UnitID: byCode, Code: "T-101"},
			{UnitID: mapped, ExternalID: "veh-9"},
		},
		[]telematicsports.ProviderVehicle{
			{ID: "veh-1", VIN: "1HGCM82633A004352"},
			{ID: "veh-2", Name: "t-101"},
			{ID: "veh-9", Name: "T-900"},
			{ID: "veh-3", Name: "unknown"},
		},
		&unmatched,
	)

	assert.Equal(t, []vehiclematch.Assignment{
		{UnitID: byVIN, ExternalID: "veh-1"},
		{UnitID: byCode, ExternalID: "veh-2"},
	}, got)
	assert.Equal(t, 1, unmatched, "the vehicle nobody claims is counted; the mapped one is skipped")
}
