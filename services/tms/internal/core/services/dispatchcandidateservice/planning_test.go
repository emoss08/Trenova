package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	denverLat, denverLon   = 39.7392, -104.9903
	chicagoLat, chicagoLon = 41.8781, -87.6298
)

func TestCommitPlannedMove_DeadheadIsMeasuredFromThePriorDestination(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trac_")

	snapshot := &FleetSnapshot{
		Now: now,
		PosByTractor: map[pulid.ID]*telematics.VehiclePosition{
			tractorID: {TractorID: tractorID, Latitude: denverLat, Longitude: denverLon},
		},
	}
	driver := &repositories.BoardDriver{WorkerID: workerID, TractorID: tractorID}

	nextMove := &repositories.BoardMove{
		Distance:        ptr(200),
		OriginLatitude:  ptr(chicagoLat),
		OriginLongitude: ptr(chicagoLon),
	}

	before := svc.computeTrip(driver, nextMove, snapshot)
	require.NotNil(t, before.deadheadMiles)
	assert.Greater(t, *before.deadheadMiles, 800.0,
		"from the live Denver position, Chicago is most of a day of empty miles")

	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: snapshot,
		Move: &repositories.BoardMove{
			MoveID:               pulid.MustNew("mov_"),
			DestinationLatitude:  ptr(chicagoLat),
			DestinationLongitude: ptr(chicagoLon),
			DestinationWindowEnd: ptrInt64(now + 8*3600),
		},
		Score: &CandidateScore{
			WorkerID:           workerID,
			ProjectedAvailable: now,
			EstimatedDriveMs:   4 * 3600 * 1000,
		},
	})

	after := svc.computeTrip(driver, nextMove, snapshot)
	require.NotNil(t, after.deadheadMiles)
	assert.Less(t, *after.deadheadMiles, 1.0,
		"the driver already ends up in Chicago, so the next Chicago pickup is not a deadhead")
}

func TestCommitPlannedMove_AdvancesDeparture(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")

	snapshot := &FleetSnapshot{Now: now}
	driver := &repositories.BoardDriver{WorkerID: workerID}

	completion := now + 8*3600
	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: snapshot,
		Move: &repositories.BoardMove{
			MoveID:               pulid.MustNew("mov_"),
			DestinationWindowEnd: &completion,
		},
		Score: &CandidateScore{WorkerID: workerID, ProjectedAvailable: now},
	})

	trip := svc.computeTrip(
		driver,
		&repositories.BoardMove{Distance: ptr(100)},
		snapshot,
	)

	assert.Equal(t, completion, trip.departure,
		"a driver committed through 8 hours out cannot start the next move before then")
}

func TestCommitPlannedMove_CarriesTheTrailerForward(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")
	trailerID := pulid.MustNew("trl_")

	snapshot := &FleetSnapshot{Now: now}

	CommitPlannedMove(&CommitPlannedMoveRequest{
		Snapshot: snapshot,
		Move:     &repositories.BoardMove{MoveID: pulid.MustNew("mov_")},
		Score: &CandidateScore{
			WorkerID:           workerID,
			TrailerID:          trailerID,
			ProjectedAvailable: now,
			EstimatedDriveMs:   3600 * 1000,
		},
	})

	current, ok := currentTrailer(snapshot.CommitmentsByWorker[workerID])
	require.True(t, ok)
	assert.Equal(t, trailerID, current)
}

func TestCommitPlannedMove_IgnoresIncompleteRequests(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	move := &repositories.BoardMove{MoveID: pulid.MustNew("mov_")}

	tests := []struct {
		name string
		req  *CommitPlannedMoveRequest
	}{
		{name: "nil request"},
		{name: "nil snapshot", req: &CommitPlannedMoveRequest{Move: move, Score: &CandidateScore{WorkerID: workerID}}},
		{name: "nil move", req: &CommitPlannedMoveRequest{Snapshot: &FleetSnapshot{}, Score: &CandidateScore{WorkerID: workerID}}},
		{name: "nil score", req: &CommitPlannedMoveRequest{Snapshot: &FleetSnapshot{}, Move: move}},
		{
			name: "score without a worker",
			req: &CommitPlannedMoveRequest{
				Snapshot: &FleetSnapshot{},
				Move:     move,
				Score:    &CandidateScore{},
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

	now := int64(1_000_000)

	tests := []struct {
		name  string
		move  *repositories.BoardMove
		score *CandidateScore
		want  int64
	}{
		{
			name: "no destination window falls back to finishing the drive",
			move: &repositories.BoardMove{},
			score: &CandidateScore{
				ProjectedAvailable: now,
				EstimatedDriveMs:   6 * 3600 * 1000,
			},
			want: now + 6*3600,
		},
		{
			name: "a later appointment wins over the drive time",
			move: &repositories.BoardMove{DestinationWindowStart: now + 20*3600},
			score: &CandidateScore{
				ProjectedAvailable: now,
				EstimatedDriveMs:   6 * 3600 * 1000,
			},
			want: now + 20*3600,
		},
		{
			name: "a drive running past the appointment wins",
			move: &repositories.BoardMove{DestinationWindowStart: now + 2*3600},
			score: &CandidateScore{
				ProjectedAvailable: now,
				EstimatedDriveMs:   9 * 3600 * 1000,
			},
			want: now + 9*3600,
		},
		{name: "nil move", move: nil, score: &CandidateScore{}, want: 0},
		{name: "nil score", move: &repositories.BoardMove{}, score: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, PlannedCompletion(tt.move, tt.score))
		})
	}
}

func TestScoreCandidate_RejectsIncompleteRequests(t *testing.T) {
	t.Parallel()

	svc := &Service{}

	assert.Nil(t, svc.ScoreCandidate(nil))
	assert.Nil(t, svc.ScoreCandidate(&ScoreCandidateRequest{}))
	assert.Nil(t, svc.ScoreCandidate(&ScoreCandidateRequest{
		Driver:   &repositories.BoardDriver{},
		Move:     &repositories.BoardMove{},
		Snapshot: &FleetSnapshot{},
	}), "a snapshot without a control has no scoring weights to resolve")
}

func ptrInt64(v int64) *int64 { return &v }
