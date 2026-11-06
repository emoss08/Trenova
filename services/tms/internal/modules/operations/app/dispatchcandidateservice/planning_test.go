package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitPlannedMove_DeadheadIsMeasuredFromThePriorDestination(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", denverLat, denverLon)
	next := pickupAt(chicagoLat, chicagoLon, 200)

	before := evaluateFor(s, driver, next, nil)
	require.True(t, before.trip.emptyKnown)
	assert.Greater(t, before.trip.emptyMiles, 800.0,
		"from the live Denver position, Chicago is most of a day of empty miles")

	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: s,
		Move: &dispatchboardports.BoardMove{
			MoveID:               pulid.MustNew("smv_"),
			DestinationLatitude:  new(chicagoLat),
			DestinationLongitude: new(chicagoLon),
		},
		Score: &dispatchplanports.CandidateScore{
			WorkerID:        driver.WorkerID,
			ProjectedFreeAt: testNow + 8*3600,
		},
	})

	after := evaluateFor(s, driver, next, nil)
	require.True(t, after.trip.emptyKnown)
	assert.Less(t, after.trip.emptyMiles, 1.0,
		"the driver already ends up in Chicago, so the next Chicago pickup is not a deadhead")
	assert.Equal(t, dispatchplanports.OriginCommitment, after.profile.originSource)
}

func TestCommitPlannedMove_AdvancesDepartureToTheProjectedFinish(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	completion := testNow + 8*3600

	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: s,
		Move:     &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")},
		Score: &dispatchplanports.CandidateScore{
			WorkerID:        driver.WorkerID,
			ProjectedFreeAt: completion,
		},
	})

	eval := evaluateFor(s, driver, pickupAt(chicagoLat, chicagoLon, 100), nil)

	assert.Equal(t, completion, eval.trip.departure,
		"a driver committed through 8 hours out cannot start the next move before then")
}

func TestCommitPlannedMove_RefreshesAPreparedProfileInPlace(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	s.Prepare()
	require.Equal(t, testNow, s.profile(driver).availableAt)

	completion := testNow + 6*3600
	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: s,
		Move:     &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")},
		Score: &dispatchplanports.CandidateScore{
			WorkerID:        driver.WorkerID,
			ProjectedFreeAt: completion,
		},
	})

	assert.Equal(t, completion, s.profile(driver).availableAt,
		"the next evaluation in the same run sees the new commitment")
}

func TestCommitPlannedMove_CarriesTheTrailerForward(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	workerID := pulid.MustNew("wrk_")
	trailerID := pulid.MustNew("tr_")

	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: s,
		Move:     &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")},
		Score: &dispatchplanports.CandidateScore{
			WorkerID:        workerID,
			TrailerID:       trailerID,
			ProjectedFreeAt: testNow + 3600,
		},
	})

	current, ok := currentTrailer(s.CommitmentsByWorker[workerID])
	require.True(t, ok)
	assert.Equal(t, trailerID, current)
}

func TestCommitPlannedMove_IgnoresIncompleteRequests(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	move := &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")}

	tests := []struct {
		name string
		req  *CommitPlannedMoveRequest
	}{
		{name: "nil request"},
		{
			name: "nil snapshot",
			req: &CommitPlannedMoveRequest{
				Move:  move,
				Score: &dispatchplanports.CandidateScore{WorkerID: workerID},
			},
		},
		{
			name: "nil move",
			req: &CommitPlannedMoveRequest{
				Snapshot: &FleetSnapshot{},
				Score:    &dispatchplanports.CandidateScore{WorkerID: workerID},
			},
		},
		{
			name: "nil score",
			req:  &CommitPlannedMoveRequest{Snapshot: &FleetSnapshot{}, Move: move},
		},
		{
			name: "score without a worker",
			req: &CommitPlannedMoveRequest{
				Snapshot: &FleetSnapshot{},
				Move:     move,
				Score:    &dispatchplanports.CandidateScore{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { CommitPlannedMove(tt.req) })
			if tt.req != nil && tt.req.Snapshot != nil {
				assert.Empty(t, tt.req.Snapshot.CommitmentsByWorker)
			}
		})
	}
}

func TestPlannedCompletion(t *testing.T) {
	t.Parallel()

	end := testNow + 22*3600

	tests := []struct {
		name  string
		move  *dispatchboardports.BoardMove
		score *dispatchplanports.CandidateScore
		want  int64
	}{
		{
			name:  "the projected finish wins",
			move:  &dispatchboardports.BoardMove{DestinationWindowStart: testNow + 20*3600},
			score: &dispatchplanports.CandidateScore{ProjectedFreeAt: testNow + 9*3600},
			want:  testNow + 9*3600,
		},
		{
			name: "without a projection the delivery window closes it",
			move: &dispatchboardports.BoardMove{
				DestinationWindowStart: testNow + 20*3600,
				DestinationWindowEnd:   &end,
			},
			score: &dispatchplanports.CandidateScore{},
			want:  end,
		},
		{name: "nil move", score: &dispatchplanports.CandidateScore{}, want: 0},
		{name: "nil score", move: &dispatchboardports.BoardMove{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, PlannedCompletion(tt.move, tt.score))
		})
	}
}
