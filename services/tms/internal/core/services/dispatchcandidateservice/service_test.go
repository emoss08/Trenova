package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectedTimeAvailable(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)

	assert.Equal(t, now, ProjectedTimeAvailable(nil, now),
		"a driver holding nothing is available now")

	assert.Equal(t, int64(1_020_000), ProjectedTimeAvailable(
		[]*repositories.WorkerCommitment{
			{WindowEnd: 1_010_000},
			{WindowEnd: 1_020_000},
			{WindowEnd: 1_005_000},
		},
		now,
	), "PTA is the end of the last committed move")

	assert.Equal(t, now, ProjectedTimeAvailable(
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

func TestCurrentTrailer_ZeroWindowsNeverBeatRealOnes(t *testing.T) {
	t.Parallel()

	real := pulid.MustNew("tr_")
	zero := pulid.MustNew("tr_")

	trailerID, found := currentTrailer([]*repositories.WorkerCommitment{
		{WindowEnd: 500, TrailerID: real},
		{WindowEnd: 0, TrailerID: zero},
	})
	require.True(t, found)
	assert.Equal(t, real, trailerID, "a zero window must not win the latest commitment")

	trailerID, found = currentTrailer([]*repositories.WorkerCommitment{
		{WindowEnd: 0, TrailerID: zero},
		{WindowEnd: 500, TrailerID: real},
	})
	require.True(t, found)
	assert.Equal(t, real, trailerID, "ordering must not change which commitment is latest")

	trailerID, found = currentTrailer([]*repositories.WorkerCommitment{
		{WindowEnd: 0, TrailerID: zero},
	})
	require.True(t, found)
	assert.Equal(t, zero, trailerID, "an all-zero set still yields a continuity signal")
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
	assert.InDelta(t, 918.0*roadCircuityFactor, *trip.deadheadMiles, 60.0,
		"Denver to Chicago is roughly 918 great-circle miles, graded up by road circuity")
	assert.Greater(t, trip.totalMiles, 1000.0, "deadhead is added to the loaded miles")
	assert.Negative(t, trip.slackMinutes, "the truck cannot cover 900 empty miles in an hour")
}

func TestComputeTrip_BusyDriverDepartsAtPTA(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")
	pta := now + 4*3600

	snapshot := &FleetSnapshot{
		Now: now,
		CommitmentsByWorker: map[pulid.ID][]*repositories.WorkerCommitment{
			workerID: {{WindowEnd: pta}},
		},
	}

	trip := svc.computeTrip(
		&repositories.BoardDriver{WorkerID: workerID},
		&repositories.BoardMove{
			Distance:          ptr(200),
			OriginWindowStart: now + 2*3600,
		},
		snapshot,
	)

	assert.Equal(t, pta, trip.projectedArrival,
		"with no deadhead leg the driver arrives when their current work ends")
	assert.InDelta(t, -120.0, trip.slackMinutes, 0.001,
		"a driver busy past the appointment must project late, not on time")
	assert.True(t, trip.appointmentKnown)
}

func TestComputeTrip_NoAppointmentLeavesSlackUnknown(t *testing.T) {
	t.Parallel()

	svc := &Service{}

	trip := svc.computeTrip(
		&repositories.BoardDriver{},
		&repositories.BoardMove{Distance: ptr(200)},
		&FleetSnapshot{Now: 1_000_000},
	)

	assert.False(t, trip.appointmentKnown)
	assert.Zero(t, trip.slackMinutes)
}

func blockingFinding() []dispatcheligibility.Finding {
	return []dispatcheligibility.Finding{{
		Code:     dispatcheligibility.CodeHOSNoDriveTime,
		Severity: dispatcheligibility.SeverityBlock,
	}}
}

func TestScoreDriver_StaleHOSNeverFeedsTheMarginFactor(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := timeutils.NowUnix()
	workerID := pulid.MustNew("wrk_")

	snapshot := &FleetSnapshot{
		Now:              now,
		TelematicsActive: true,
		WorkersByID: map[pulid.ID]*worker.Worker{
			workerID: {ID: workerID, DriverType: worker.DriverTypeOTR},
		},
		HOSByWorker: map[pulid.ID]*telematics.WorkerHOSState{
			workerID: {
				WorkerID:         workerID,
				DriveRemainingMs: 8 * 3_600_000,
				ShiftRemainingMs: 10 * 3_600_000,
				RecordedAt:       now - 13*3600,
			},
		},
		Control: dispatchcontrol.NewDefaultDispatchControl(
			pulid.MustNew("org_"),
			pulid.MustNew("bu_"),
		),
	}

	score := svc.scoreDriver(&scoreDriverParams{
		Driver:   &repositories.BoardDriver{WorkerID: workerID, DriverType: worker.DriverTypeOTR},
		Move:     &repositories.BoardMove{MoveID: pulid.MustNew("smv_"), Distance: ptr(200)},
		Snapshot: snapshot,
		Weights:  proximityWeights(),
	})

	require.NotNil(t, score)
	_, present := factorByKey(score.Factors, dispatchcontrol.FactorHOSMargin)
	assert.False(t, present, "a 13-hour-old snapshot must not feed the hours factor")
	assert.Empty(t, score.HOSStrategy, "no projection may run on stale clocks")
}

func TestScoreDriver_NoTelematicsMeansNoHOSAnywhere(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	now := timeutils.NowUnix()
	workerID := pulid.MustNew("wrk_")

	snapshot := &FleetSnapshot{
		Now:              now,
		TelematicsActive: false,
		WorkersByID: map[pulid.ID]*worker.Worker{
			workerID: {ID: workerID, DriverType: worker.DriverTypeOTR},
		},
		HOSByWorker: map[pulid.ID]*telematics.WorkerHOSState{},
		Control: dispatchcontrol.NewDefaultDispatchControl(
			pulid.MustNew("org_"),
			pulid.MustNew("bu_"),
		),
	}

	score := svc.scoreDriver(&scoreDriverParams{
		Driver: &repositories.BoardDriver{WorkerID: workerID, DriverType: worker.DriverTypeOTR},
		Move: &repositories.BoardMove{
			MoveID:            pulid.MustNew("smv_"),
			Distance:          ptr(200),
			OriginWindowStart: now + 12*3600,
		},
		Snapshot: snapshot,
		Weights:  proximityWeights(),
	})

	require.NotNil(t, score)
	_, present := factorByKey(score.Factors, dispatchcontrol.FactorHOSMargin)
	assert.False(t, present)
	for i := range score.Findings {
		assert.NotEqual(t, "hos", score.Findings[i].Field,
			"no HOS findings may surface without a telematics integration")
	}
	assert.NotEqual(t, telematics.FeasibilityVerdictUnknown, score.Verdict,
		"missing HOS data is expected without an integration, not unknown")
}

func TestDeadheadOrigin_MidRunDriverDepartsFromCommitmentDestination(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trac_")
	destLat, destLon := 35.0, -90.0

	snapshot := &FleetSnapshot{
		Now: now,
		CommitmentsByWorker: map[pulid.ID][]*repositories.WorkerCommitment{
			workerID: {
				{WindowEnd: now + 3600, DestLatitude: &destLat, DestLongitude: &destLon},
				{WindowEnd: now + 2*3600, DestLatitude: new(36.0), DestLongitude: new(-91.0)},
			},
		},
		PosByTractor: map[pulid.ID]*telematics.VehiclePosition{
			tractorID: {TractorID: tractorID, Latitude: 40.0, Longitude: -100.0},
		},
	}

	origin, ok := deadheadOrigin(
		&repositories.BoardDriver{WorkerID: workerID, TractorID: tractorID},
		snapshot,
	)

	require.True(t, ok)
	assert.InDelta(t, 36.0, origin.lat, 0.001,
		"the latest commitment's destination wins over the live position")
	assert.InDelta(t, -91.0, origin.lon, 0.001)
}

func TestDeadheadOrigin_IdleDriverUsesTheLivePosition(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trac_")

	snapshot := &FleetSnapshot{
		Now: now,
		PosByTractor: map[pulid.ID]*telematics.VehiclePosition{
			tractorID: {TractorID: tractorID, Latitude: 40.0, Longitude: -100.0},
		},
	}

	origin, ok := deadheadOrigin(
		&repositories.BoardDriver{WorkerID: workerID, TractorID: tractorID},
		snapshot,
	)

	require.True(t, ok)
	assert.InDelta(t, 40.0, origin.lat, 0.001)

	_, ok = deadheadOrigin(&repositories.BoardDriver{WorkerID: workerID}, snapshot)
	assert.False(t, ok, "no commitment destination and no position means no deadhead")
}

func TestAppointmentFinding_EnforcementDecidesBlockOrWarn(t *testing.T) {
	t.Parallel()

	late := tripEstimate{appointmentKnown: true, slackMinutes: -45}
	control := dispatchcontrol.NewDefaultDispatchControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)

	control.EnforceWorkerPTARestrictions = true
	finding := appointmentFinding(late, control)
	require.NotNil(t, finding)
	assert.Equal(t, dispatcheligibility.CodeAppointmentMissed, finding.Code)
	assert.Equal(t, dispatcheligibility.SeverityBlock, finding.Severity)

	control.EnforceWorkerPTARestrictions = false
	finding = appointmentFinding(late, control)
	require.NotNil(t, finding)
	assert.Equal(t, dispatcheligibility.CodeAppointmentAtRisk, finding.Code)
	assert.Equal(t, dispatcheligibility.SeverityWarn, finding.Severity)

	onTime := tripEstimate{appointmentKnown: true, slackMinutes: 30}
	assert.Nil(t, appointmentFinding(onTime, control))
	assert.Nil(t, appointmentFinding(tripEstimate{}, control))
}

func TestNextApprovedPTOStart_SkipsActiveAndDistantWindows(t *testing.T) {
	t.Parallel()

	now := int64(1_000_000)
	timeOff := []*repositories.WorkerTimeOff{
		{StartDate: now - 3600, EndDate: now + 3600},
		{StartDate: now + 5*86400, EndDate: now + 6*86400},
		{StartDate: now + 2*86400, EndDate: now + 3*86400},
		{StartDate: now + 30*86400, EndDate: now + 31*86400},
	}

	assert.Equal(t, now+2*86400, nextApprovedPTOStart(timeOff, now),
		"the nearest upcoming window within the lookahead wins")
	assert.Zero(t, nextApprovedPTOStart(nil, now))
}
