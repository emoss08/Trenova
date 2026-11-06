package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	jolietLat = 41.525
	jolietLon = -88.082
)

func TestLastTimeOffEnds_KeepsEachDriversLatestEndUpToNow(t *testing.T) {
	t.Parallel()

	ada, bo := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	ends := lastTimeOffEnds([]*dispatchboardports.WorkerTimeOff{
		{WorkerID: ada, EndDate: 100},
		{WorkerID: ada, EndDate: 300},
		{WorkerID: bo, EndDate: 9_999},
	}, 1_000)

	assert.Equal(t, map[pulid.ID]int64{ada: 300, bo: 1_000}, ends,
		"time off still running counts as home until now")
}

func TestApplyHomeProfile_TakesTheLatestOfRecordedTimeOffAndSeenAtHome(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", jolietLat+0.1, jolietLon)
	w := s.WorkersByID[driver.WorkerID]
	w.HomeLatitude, w.HomeLongitude = new(jolietLat), new(jolietLon)
	w.LastHomeAt = new(int64(1_000))
	s.LastTimeOffEnd = map[pulid.ID]int64{driver.WorkerID: 2_000}
	s.PosByTractor[driver.TractorID].RecordedAt = 3_000

	profile := buildProfile(s, driver)

	require.NotNil(t, profile)
	assert.True(t, profile.homeKnown)
	assert.Equal(t, int64(3_000), profile.lastHomeAt, "the tractor is within the home radius now")
}

func TestApplyHomeProfile_IgnoresAPositionAwayFromHome(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", denverLat, denverLon)
	w := s.WorkersByID[driver.WorkerID]
	w.HomeLatitude, w.HomeLongitude = new(jolietLat), new(jolietLon)
	w.LastHomeAt = new(int64(1_000))
	s.PosByTractor[driver.TractorID] = &telematics.VehiclePosition{
		TractorID: driver.TractorID, Latitude: denverLat, Longitude: denverLon, RecordedAt: 5_000,
	}

	profile := buildProfile(s, driver)

	assert.Equal(t, int64(1_000), profile.lastHomeAt)
}

func TestHomeOverdueDays_CountsDaysPastTheTargetWhenTheMoveFinishes(t *testing.T) {
	t.Parallel()

	profile := &driverProfile{lastHomeAt: 0}
	assert.Zero(t, homeOverdueDays(profile, 20*secondsPerDay, 10), "an unknown last home prices nothing")

	profile.lastHomeAt = 1
	assert.Zero(t, homeOverdueDays(profile, 8*secondsPerDay, 10))
	assert.InDelta(t, 2.0, homeOverdueDays(profile, 12*secondsPerDay+1, 10), 0.001)
}

func TestHomewardShare_IsTheShareOfTheWayHomeAMoveCloses(t *testing.T) {
	t.Parallel()

	profile := &driverProfile{
		home:         geoPoint{lat: jolietLat, lon: jolietLon},
		homeKnown:    true,
		origin:       geoPoint{lat: denverLat, lon: denverLon},
		originSource: dispatchplanports.OriginPosition,
	}
	home := &dispatchboardports.BoardMove{
		DestinationLatitude: new(jolietLat), DestinationLongitude: new(jolietLon),
	}
	away := &dispatchboardports.BoardMove{
		DestinationLatitude: new(denverLat - 5), DestinationLongitude: new(denverLon),
	}

	assert.InDelta(t, 1.0, homewardShare(profile, home), 0.001)
	assert.Zero(t, homewardShare(profile, away), "a move away from home earns no credit")

	profile.homeKnown = false
	assert.Zero(t, homewardShare(profile, home), "an unknown home earns no credit")
}
