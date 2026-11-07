package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverStart_APositionedDriverStartsAtTheTargetFromTheirArrival(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	positioned := addPlacedDriver(s, "Ana", 35.4676, -97.5164)
	other := addPlacedDriver(s, "Ben", 35.4676, -97.5164)
	s.PositionedByWorker = map[pulid.ID]*driverpositioning.Positioning{}
	s.PositionedByWorker[positioned.WorkerID] = &driverpositioning.Positioning{
		WorkerID:    positioned.WorkerID,
		ToLatitude:  32.5468,
		ToLongitude: -96.3281,
		ArriveAt:    testNow + 5*3600,
		Status:      driverpositioning.StatusActive,
	}

	start, freeAt, ok := s.DriverStart(positioned.WorkerID)
	require.True(t, ok)
	assert.Equal(t, dispatchmarket.Point{Latitude: 32.5468, Longitude: -96.3281}, start)
	assert.Equal(t, testNow+5*3600, freeAt, "the planner waits for the driver to arrive")

	start, freeAt, ok = s.DriverStart(other.WorkerID)
	require.True(t, ok)
	assert.Equal(t, dispatchmarket.Point{Latitude: 35.4676, Longitude: -97.5164}, start)
	assert.Equal(t, testNow, freeAt)
}
