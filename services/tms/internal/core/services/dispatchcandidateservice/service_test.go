package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectedTimeAvailable(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)

	assert.Equal(t, now, projectedTimeAvailable(nil, now),
		"a driver holding nothing is available now")

	assert.Equal(t, int64(1_020_000), projectedTimeAvailable(
		[]*repositories.WorkerCommitment{
			{WindowEnd: 1_010_000},
			{WindowEnd: 1_020_000},
			{WindowEnd: 1_005_000},
		},
		now,
	), "PTA is the end of the last committed move")

	assert.Equal(t, now, projectedTimeAvailable(
		[]*repositories.WorkerCommitment{{WindowEnd: now - 5_000}},
		now,
	), "commitments already in the past do not push PTA backwards")
}

func TestCurrentTrailer(t *testing.T) {
	t.Parallel()

	older := pulid.MustNew("tr_")
	newer := pulid.MustNew("tr_")

	trailerID, found := currentTrailer([]*repositories.WorkerCommitment{
		{WindowEnd: 100, TrailerID: older},
		{WindowEnd: 500, TrailerID: newer},
	})
	require.True(t, found)
	assert.Equal(t, newer, trailerID, "the latest commitment's trailer is the current one")

	_, found = currentTrailer([]*repositories.WorkerCommitment{{WindowEnd: 100}})
	assert.False(t, found, "a commitment without a trailer yields no continuity signal")

	_, found = currentTrailer(nil)
	assert.False(t, found)
}

func TestResolveTrailer_PrefersTheExistingAssignment(t *testing.T) {
	t.Parallel()

	assigned := pulid.MustNew("tr_")
	previous := pulid.MustNew("tr_")

	assert.Equal(t, assigned, resolveTrailer(&repositories.BoardMove{
		AssignedTrailerID:     assigned,
		PreviousMoveTrailerID: previous,
	}))
	assert.Equal(t, previous, resolveTrailer(&repositories.BoardMove{
		PreviousMoveTrailerID: previous,
	}))
	assert.True(t, resolveTrailer(&repositories.BoardMove{}).IsNil())
}

func TestMoveWindow_SpansOriginToLatestDestinationTime(t *testing.T) {
	t.Parallel()

	end := int64(900)
	window := moveWindow(&repositories.BoardMove{
		OriginWindowStart:      100,
		DestinationWindowStart: 800,
		DestinationWindowEnd:   &end,
	})

	assert.Equal(t, int64(100), window.Start)
	assert.Equal(t, int64(900), window.End)

	withoutEnd := moveWindow(&repositories.BoardMove{
		OriginWindowStart:      100,
		DestinationWindowStart: 800,
	})
	assert.Equal(t, int64(800), withoutEnd.End)
}

// Blocked candidates must always sink below eligible ones, even when their soft factors
// score higher — a dispatcher should never see an illegal option ranked first.
func TestSortCandidates_BlockedAlwaysSinkBelowEligible(t *testing.T) {
	t.Parallel()

	blocked := &CandidateScore{
		WorkerName: "Blocked High Scorer",
		Score:      99,
		Findings:   blockingFinding(),
	}
	eligible := &CandidateScore{WorkerName: "Eligible Low Scorer", Score: 10}
	better := &CandidateScore{WorkerName: "Eligible High Scorer", Score: 80}

	results := []*CandidateScore{blocked, eligible, better}
	sortCandidates(results)

	assert.Equal(t, "Eligible High Scorer", results[0].WorkerName)
	assert.Equal(t, "Eligible Low Scorer", results[1].WorkerName)
	assert.Equal(t, "Blocked High Scorer", results[2].WorkerName)
}

func TestSortCandidates_TiesBreakOnNameForStableOrdering(t *testing.T) {
	t.Parallel()

	results := []*CandidateScore{
		{WorkerName: "Zoe Adams", Score: 50},
		{WorkerName: "Alex Ruiz", Score: 50},
	}
	sortCandidates(results)

	assert.Equal(t, "Alex Ruiz", results[0].WorkerName)
}

func TestComputeTrip_DeadheadUnknownWithoutAPosition(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := int64(1_000_000)
	lat, lon := 41.0, -87.0

	trip := svc.computeTrip(
		&repositories.BoardDriver{TractorID: pulid.MustNew("trac_")},
		&repositories.BoardMove{
			Distance:          ptr(500),
			OriginLatitude:    &lat,
			OriginLongitude:   &lon,
			OriginWindowStart: now + 7200,
		},
		&FleetSnapshot{Now: now, PosByTractor: nil},
	)

	assert.Nil(t, trip.deadheadMiles, "no GPS fix means deadhead is unknown, not zero")
	assert.InDelta(t, 500.0, trip.totalMiles, 0.001)
	assert.Positive(t, trip.driveMs)
	assert.InDelta(t, 120.0, trip.slackMinutes, 0.001,
		"with no deadhead leg the driver can start immediately")
}

// A truck hundreds of miles from the pickup cannot make an appointment an hour out; the
// deadhead leg has to show up as negative slack, not be silently ignored.
func TestComputeTrip_DeadheadDrivesTheArrivalProjection(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := int64(1_000_000)
	tractorID := pulid.MustNew("trac_")
	originLat, originLon := 41.8781, -87.6298

	snapshot := &FleetSnapshot{
		Now: now,
		PosByTractor: map[pulid.ID]*telematics.VehiclePosition{
			tractorID: {TractorID: tractorID, Latitude: 39.7392, Longitude: -104.9903},
		},
	}

	trip := svc.computeTrip(
		&repositories.BoardDriver{TractorID: tractorID},
		&repositories.BoardMove{
			Distance:          ptr(300),
			OriginLatitude:    &originLat,
			OriginLongitude:   &originLon,
			OriginWindowStart: now + 3600,
		},
		snapshot,
	)

	require.NotNil(t, trip.deadheadMiles)
	assert.InDelta(t, 900.0, *trip.deadheadMiles, 50.0, "Denver to Chicago is roughly 900 miles")
	assert.Greater(t, trip.totalMiles, 1000.0, "deadhead is added to the loaded miles")
	assert.Negative(t, trip.slackMinutes, "the truck cannot cover 900 empty miles in an hour")
}

func blockingFinding() []dispatcheligibility.Finding {
	return []dispatcheligibility.Finding{{
		Code:     dispatcheligibility.CodeHOSNoDriveTime,
		Severity: dispatcheligibility.SeverityBlock,
	}}
}
