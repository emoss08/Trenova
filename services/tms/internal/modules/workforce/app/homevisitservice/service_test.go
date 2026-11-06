package homevisitservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/app/homevisitservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestHomeVisitsOf_StampsOnlyDriversSeenWithinTheHomeRadius(t *testing.T) {
	t.Parallel()

	near, far, homeless := pulid.MustNew("trac_"), pulid.MustNew("trac_"), pulid.MustNew("trac_")
	nearDriver, farDriver := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")

	visits := homevisitservice.VisitsOf(
		[]*telematics.VehiclePosition{
			{TractorID: near, Latitude: 41.60, Longitude: -88.08, RecordedAt: 100},
			{TractorID: far, Latitude: 39.74, Longitude: -104.99, RecordedAt: 200},
			{TractorID: homeless, Latitude: 41.52, Longitude: -88.08, RecordedAt: 300},
		},
		[]telematicsports.TractorHome{
			{TractorID: near, WorkerID: nearDriver, HomeLatitude: 41.52, HomeLongitude: -88.08},
			{TractorID: far, WorkerID: farDriver, HomeLatitude: 41.52, HomeLongitude: -88.08},
		},
	)

	assert.Equal(t, []telematicsports.HomeVisit{{WorkerID: nearDriver, At: 100}}, visits)
}
