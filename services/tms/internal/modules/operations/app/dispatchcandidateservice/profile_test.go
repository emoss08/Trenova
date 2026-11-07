package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProfile_OriginFallsBackFromCommitmentToGPSToLastKnownLocation(t *testing.T) {
	t.Parallel()

	const (
		commitLat, commitLon = 36.0, -91.0
		gpsLat, gpsLon       = 40.0, -100.0
		lastLat, lastLon     = 33.0, -84.0
	)

	activeCommitment := func(workerID pulid.ID) *dispatchboardports.WorkerCommitment {
		return &dispatchboardports.WorkerCommitment{
			WorkerID:      workerID,
			WindowStart:   testNow - 3600,
			WindowEnd:     testNow + 2*3600,
			DestLatitude:  new(commitLat),
			DestLongitude: new(commitLon),
		}
	}

	tests := []struct {
		name   string
		setup  func(*FleetSnapshot, *dispatchboardports.BoardDriver)
		source dispatchplanports.OriginSource
		lat    float64
		lon    float64
	}{
		{
			name: "a driver still on a run departs from where it ends",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				s.CommitmentsByWorker[d.WorkerID] = []*dispatchboardports.WorkerCommitment{
					activeCommitment(d.WorkerID),
				}
			},
			source: dispatchplanports.OriginCommitment,
			lat:    commitLat,
			lon:    commitLon,
		},
		{
			name: "the latest of several runs is the one that counts",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				later := activeCommitment(d.WorkerID)
				later.WindowEnd = testNow + 5*3600
				later.DestLatitude = new(35.0)
				later.DestLongitude = new(-89.0)
				s.CommitmentsByWorker[d.WorkerID] = []*dispatchboardports.WorkerCommitment{
					later,
					activeCommitment(d.WorkerID),
				}
			},
			source: dispatchplanports.OriginCommitment,
			lat:    35.0,
			lon:    -89.0,
		},
		{
			name: "a finished run falls back to the live position",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				done := activeCommitment(d.WorkerID)
				done.WindowEnd = testNow - 60
				s.CommitmentsByWorker[d.WorkerID] = []*dispatchboardports.WorkerCommitment{done}
			},
			source: dispatchplanports.OriginPosition,
			lat:    gpsLat,
			lon:    gpsLon,
		},
		{
			name: "a run with no geocoded destination falls back to the live position",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				unknown := activeCommitment(d.WorkerID)
				unknown.DestLatitude = nil
				unknown.DestLongitude = nil
				s.CommitmentsByWorker[d.WorkerID] = []*dispatchboardports.WorkerCommitment{unknown}
			},
			source: dispatchplanports.OriginPosition,
			lat:    gpsLat,
			lon:    gpsLon,
		},
		{
			name: "without a position the tractor's last known location is used",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				delete(s.PosByTractor, d.TractorID)
			},
			source: dispatchplanports.OriginLastDrop,
			lat:    lastLat,
			lon:    lastLon,
		},
		{
			name: "nothing known leaves the origin unknown",
			setup: func(s *FleetSnapshot, d *dispatchboardports.BoardDriver) {
				delete(s.PosByTractor, d.TractorID)
				s.TractorByID[d.TractorID].LastKnownLocationLatitude = nil
				s.TractorByID[d.TractorID].LastKnownLocationLongitude = nil
			},
			source: dispatchplanports.OriginUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestSnapshot()
			driver := addPlacedDriver(s, "Ada", gpsLat, gpsLon)
			s.TractorByID[driver.TractorID].LastKnownLocationLatitude = new(lastLat)
			s.TractorByID[driver.TractorID].LastKnownLocationLongitude = new(lastLon)
			tt.setup(s, driver)

			profile := buildProfile(s, driver)

			require.NotNil(t, profile)
			assert.Equal(t, tt.source, profile.originSource)
			assert.InDelta(t, tt.lat, profile.origin.lat, 0.0001)
			assert.InDelta(t, tt.lon, profile.origin.lon, 0.0001)
		})
	}
}

func TestBuildProfile_DriverWithoutAWorkerRecordHasNoProfile(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	delete(s.WorkersByID, driver.WorkerID)

	assert.Nil(t, buildProfile(s, driver))
}

func TestBuildProfile_CommittedDriveCountsRemainingMinutesOrOpenDistance(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	s.CommitmentsByWorker[driver.WorkerID] = []*dispatchboardports.WorkerCommitment{
		{WorkerID: driver.WorkerID, WindowEnd: testNow + 3600, RemainingDriveMinutes: 90},
		{WorkerID: driver.WorkerID, WindowEnd: testNow + 7200, Distance: new(100.0)},
		{WorkerID: driver.WorkerID, WindowEnd: testNow - 3600, Distance: new(500.0)},
	}

	profile := buildProfile(s, driver)

	require.NotNil(t, profile)
	assert.Equal(t, 90*millisPerMinute+2*testHourMs, profile.committedDriveMs,
		"remaining minutes win; open work is timed from its distance; finished work adds nothing")
	assert.Equal(t, testNow+7200, profile.availableAt)
}

func TestProjectedTimeAvailable(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)

	assert.Equal(t, now, ProjectedTimeAvailable(nil, now),
		"a driver holding nothing is available now")

	assert.Equal(t, int64(1_020_000), ProjectedTimeAvailable(
		[]*dispatchboardports.WorkerCommitment{
			{WindowEnd: 1_010_000},
			{WindowEnd: 1_020_000},
			{WindowEnd: 1_005_000},
		},
		now,
	), "PTA is the end of the last committed move")

	assert.Equal(t, int64(1_030_000), ProjectedTimeAvailable(
		[]*dispatchboardports.WorkerCommitment{{WindowEnd: 1_010_000, ExpectedEnd: 1_030_000}},
		now,
	), "an ETA past the window end pushes PTA out")

	assert.Equal(t, now, ProjectedTimeAvailable(
		[]*dispatchboardports.WorkerCommitment{{WindowEnd: now - 5_000}},
		now,
	), "commitments already in the past do not push PTA backwards")
}

func TestCurrentTrailer(t *testing.T) {
	t.Parallel()

	older := pulid.MustNew("tr_")
	newer := pulid.MustNew("tr_")

	trailerID, found := currentTrailer([]*dispatchboardports.WorkerCommitment{
		{WindowEnd: 100, TrailerID: older},
		{WindowEnd: 500, TrailerID: newer},
	})
	require.True(t, found)
	assert.Equal(t, newer, trailerID, "the latest commitment's trailer is the current one")

	_, found = currentTrailer([]*dispatchboardports.WorkerCommitment{{WindowEnd: 100}})
	assert.False(t, found, "a commitment without a trailer yields no continuity signal")

	_, found = currentTrailer(nil)
	assert.False(t, found)
}

func TestCurrentTrailer_ZeroWindowsNeverBeatRealOnes(t *testing.T) {
	t.Parallel()

	active := pulid.MustNew("tr_")
	zero := pulid.MustNew("tr_")

	trailerID, found := currentTrailer([]*dispatchboardports.WorkerCommitment{
		{WindowEnd: 500, TrailerID: active},
		{WindowEnd: 0, TrailerID: zero},
	})
	require.True(t, found)
	assert.Equal(t, active, trailerID, "a zero window must not win the latest commitment")

	trailerID, found = currentTrailer([]*dispatchboardports.WorkerCommitment{
		{WindowEnd: 0, TrailerID: zero},
		{WindowEnd: 500, TrailerID: active},
	})
	require.True(t, found)
	assert.Equal(t, active, trailerID, "ordering must not change which commitment is latest")
}

func TestNextApprovedPTOStart_SkipsActiveAndDistantWindows(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)
	timeOff := []*dispatchboardports.WorkerTimeOff{
		{StartDate: now - 3600, EndDate: now + 3600},
		{StartDate: now + 5*86400, EndDate: now + 6*86400},
		{StartDate: now + 2*86400, EndDate: now + 3*86400},
		{StartDate: now + 30*86400, EndDate: now + 31*86400},
	}

	assert.Equal(t, now+2*86400, nextApprovedPTOStart(timeOff, now),
		"the nearest upcoming window within the lookahead wins")
	assert.Zero(t, nextApprovedPTOStart(nil, now))
}

func TestMoveWindowEnd_IsTheLatestDestinationTime(t *testing.T) {
	t.Parallel()

	end := int64(900)
	assert.Equal(t, int64(900), MoveWindowEnd(&dispatchboardports.BoardMove{
		DestinationWindowStart: 800,
		DestinationWindowEnd:   &end,
	}))
	assert.Equal(t, int64(800), MoveWindowEnd(&dispatchboardports.BoardMove{
		DestinationWindowStart: 800,
	}))
}
