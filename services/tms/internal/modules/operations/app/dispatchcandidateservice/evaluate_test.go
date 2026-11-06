package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/hosprojection"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hosState(workerID pulid.ID, driveMs int64) *telematics.WorkerHOSState {
	return &telematics.WorkerHOSState{
		WorkerID:         workerID,
		DriveRemainingMs: driveMs,
		ShiftRemainingMs: 13 * testHourMs,
		CycleRemainingMs: 60 * testHourMs,
		BreakRemainingMs: 8 * testHourMs,
		RecordedAt:       testNow,
	}
}

func TestEvaluate_HOSProjectionCarriesTheCommittedDrive(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.TelematicsActive = true
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	s.HOSByWorker[driver.WorkerID] = hosState(driver.WorkerID, 5*testHourMs)

	move := pickupAt(chicagoLat, chicagoLon, 100)
	move.OriginWindowStart = testNow + 6*3600
	move.DestinationWindowStart = testNow + 30*3600

	idle := evaluateFor(s, driver, move, nil)
	require.NotNil(t, idle.projection)
	assert.Equal(t, hosprojection.StrategyCurrentClocks, idle.projection.Strategy)
	assert.Equal(t, 2*testHourMs, idle.projection.Trip.TotalMs,
		"two hours of driving fit the five left on the clock")
	assert.Zero(t, idle.trip.restMs)
	assert.Zero(t, costLine(candidateFrom(s, idle).Cost, dispatcheconomics.ComponentRest))

	s.CommitmentsByWorker[driver.WorkerID] = []*dispatchboardports.WorkerCommitment{{
		WorkerID:              driver.WorkerID,
		WindowStart:           testNow - 3600,
		WindowEnd:             testNow + 4*3600,
		RemainingDriveMinutes: 240,
		DestLatitude:          new(chicagoLat),
		DestLongitude:         new(chicagoLon),
	}}

	busy := evaluateFor(s, driver, move, nil)
	require.NotNil(t, busy.projection)
	assert.True(t, busy.projection.Feasible)
	assert.Equal(t, 1, busy.projection.Trip.Resets,
		"four committed hours plus two new ones overrun the five left, so a reset is needed")
	assert.Equal(t, 16*testHourMs, busy.projection.Trip.TotalMs)
	assert.Equal(t, 10*testHourMs, busy.trip.restMs,
		"the rest is what the chain needs beyond the committed and new driving")
	assert.InDelta(t, 10.0, busy.features.RestHours, 0.0001)
	assert.False(t, busy.blocked())

	score := candidateFrom(s, busy)
	assert.Equal(t, 10*testHourMs, score.RestMs)
	assert.Equal(t, int64(10*3_000), costLine(score.Cost, dispatcheconomics.ComponentRest),
		"ten hours of rest at the default driver hour cost")
}

func TestEvaluate_RestIsSplitAcrossTheEmptyAndLoadedLegsByDriveShare(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.TelematicsActive = true
	driver := addPlacedDriver(s, "Ada", denverLat, denverLon)
	s.HOSByWorker[driver.WorkerID] = hosState(driver.WorkerID, 3*testHourMs)

	move := pickupAt(chicagoLat, chicagoLon, 100)
	eval := evaluateFor(s, driver, move, &routedLeg{miles: 100, seconds: 7_200})

	require.NotNil(t, eval.projection)
	assert.Equal(t, 4*testHourMs, eval.trip.driveMs, "two hours empty and two loaded")
	assert.Equal(t, 10*testHourMs, eval.trip.restMs)

	const dwell = int64(dispatchcontrol.DefaultEtaDwellMinutes) * 60
	assert.Equal(t, testNow+7_200+18_000, eval.trip.pickupArrival,
		"half the rest falls on the empty leg, which is half the driving")
	assert.Equal(t, eval.trip.pickupArrival+dwell+7_200+18_000, eval.trip.deliveryArrival,
		"the other half falls on the loaded leg")
	assert.Equal(t, eval.trip.deliveryArrival+dwell, eval.trip.freeAt)

	score := candidateFrom(s, eval)
	assert.Equal(t, eval.trip.pickupArrival, score.ProjectedArrival)
	assert.Equal(t, eval.trip.deliveryArrival, score.ProjectedDelivery)
	require.NotNil(t, score.DeadheadMiles)
	assert.InDelta(t, 100.0, *score.DeadheadMiles, 0.0001, "routed miles replace the estimate")
	assert.Equal(t, int64(30_000), costLine(score.Cost, dispatcheconomics.ComponentRest))
}

func TestEvaluate_DeliveryLatenessIsPricedPerMinute(t *testing.T) {
	t.Parallel()

	const dwell = int64(dispatchcontrol.DefaultEtaDwellMinutes) * 60
	earliest := testNow + dwell + 14_400

	tests := []struct {
		name        string
		windowStart int64
		windowEnd   *int64
		lateMinutes int64
	}{
		{
			name:        "lateness every driver would have costs nothing",
			windowStart: testNow + 3_600,
			lateMinutes: 0,
		},
		{
			name:        "the window end counts, not its start",
			windowStart: testNow + 3_600,
			windowEnd:   new(earliest + 600),
			lateMinutes: 0,
		},
		{
			name:        "no delivery appointment",
			lateMinutes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestSnapshot()
			driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
			move := pickupAt(chicagoLat, chicagoLon, 200)
			move.DestinationWindowStart = tt.windowStart
			move.DestinationWindowEnd = tt.windowEnd

			eval := evaluateFor(s, driver, move, nil)
			score := candidateFrom(s, eval)

			assert.Equal(t, earliest, eval.trip.deliveryArrival)
			assert.InDelta(t, float64(tt.lateMinutes), eval.features.DeliveryLateMinutes, 0.0001)
			assert.Equal(t, tt.lateMinutes, score.DeliveryLateMinutes)
			assert.Equal(t, tt.lateMinutes*100,
				costLine(score.Cost, dispatcheconomics.ComponentLateness))
		})
	}
}

func TestEvaluate_ADistantDriverPaysOnlyForTheDelayTheyAdd(t *testing.T) {
	t.Parallel()

	const dwell = int64(dispatchcontrol.DefaultEtaDwellMinutes) * 60
	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", denverLat, denverLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.DestinationWindowStart = testNow + 3_600

	eval := evaluateFor(s, driver, move, nil)

	deadhead := geoutils.PlanningDriveSeconds(
		geoutils.PlanningRoadMiles(denverLat, denverLon, chicagoLat, chicagoLon),
	)
	added := eval.trip.deliveryArrival - (testNow + dwell + 14_400)
	require.Positive(t, added)
	assert.InDelta(t, float64(added)/60, eval.features.DeliveryLateMinutes, 0.0001,
		"the four hours the move is late anyway are not this driver's")
	assert.GreaterOrEqual(t, added, deadhead)
}

func TestEvaluate_EarlyArrivalPastTheGraceIsPricedAsIdleWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opensIn      int64
		earlyMinutes float64
		idleCents    int64
	}{
		{
			name:         "five hours early waits three past the two-hour grace",
			opensIn:      5 * 3600,
			earlyMinutes: 300,
			idleCents:    3 * 3_000,
		},
		{
			name:         "an hour early is inside the grace",
			opensIn:      3600,
			earlyMinutes: 60,
			idleCents:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestSnapshot()
			driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
			move := pickupAt(chicagoLat, chicagoLon, 100)
			move.OriginWindowStart = testNow + tt.opensIn

			eval := evaluateFor(s, driver, move, nil)

			assert.InDelta(t, tt.earlyMinutes, eval.features.EarlyMinutes, 0.0001)
			assert.True(t, eval.trip.appointmentKnown)
			assert.InDelta(t, tt.earlyMinutes, eval.trip.slackMinutes, 0.0001)
			assert.Equal(t, tt.idleCents,
				costLine(candidateFrom(s, eval).Cost, dispatcheconomics.ComponentIdleWait))
		})
	}
}

func TestEvaluate_EquipmentTypeMismatchBlocks(t *testing.T) {
	t.Parallel()

	required := pulid.MustNew("et_")
	other := pulid.MustNew("et_")

	tests := []struct {
		name  string
		setup func(*FleetSnapshot, *dispatchboardports.BoardDriver, *dispatchboardports.BoardMove)
		code  string
	}{
		{
			name: "tractor of another type",
			setup: func(
				s *FleetSnapshot,
				driver *dispatchboardports.BoardDriver,
				move *dispatchboardports.BoardMove,
			) {
				s.TractorByID[driver.TractorID].EquipmentTypeID = other
				move.RequiredTractorTypeID = required
			},
			code: dispatchfinding.CodeTractorTypeMismatch,
		},
		{
			name: "trailer of another type",
			setup: func(
				s *FleetSnapshot,
				_ *dispatchboardports.BoardDriver,
				move *dispatchboardports.BoardMove,
			) {
				trailerID := pulid.MustNew("tr_")
				s.TrailerByID[trailerID] = &trailer.Trailer{
					ID:              trailerID,
					Code:            "TR-1",
					Status:          domaintypes.EquipmentStatusAvailable,
					EquipmentTypeID: other,
				}
				move.AssignedTrailerID = trailerID
				move.RequiredTrailerTypeID = required
			},
			code: dispatchfinding.CodeTrailerTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestSnapshot()
			driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
			move := pickupAt(chicagoLat, chicagoLon, 100)
			tt.setup(s, driver, move)

			eval := evaluateFor(s, driver, move, nil)

			require.True(t, eval.blocked())
			var found *dispatchfinding.Finding
			for i := range eval.findings {
				if eval.findings[i].Code == tt.code {
					found = &eval.findings[i]
				}
			}
			require.NotNil(t, found, "findings: %v", findingCodes(eval))
			assert.Equal(t, dispatchfinding.SeverityBlock, found.Severity)

			score := candidateFrom(s, eval)
			assert.True(t, score.Blocked())
			assert.Equal(t, telematics.FeasibilityVerdictInfeasible, score.Verdict)
		})
	}
}

func TestEvaluate_MatchingEquipmentTypesDoNotBlock(t *testing.T) {
	t.Parallel()

	required := pulid.MustNew("et_")
	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	s.TractorByID[driver.TractorID].EquipmentTypeID = required
	move := pickupAt(chicagoLat, chicagoLon, 100)
	move.RequiredTractorTypeID = required

	eval := evaluateFor(s, driver, move, nil)

	assert.False(t, eval.blocked(), "findings: %v", findingCodes(eval))
}

func TestEvaluate_UnknownLocationWarnsAndNeverFakesAZeroDeadhead(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addDriver(s, "Ada")
	move := pickupAt(chicagoLat, chicagoLon, 100)

	eval := evaluateFor(s, driver, move, nil)

	assert.False(t, eval.trip.emptyKnown)
	assert.Contains(t, findingCodes(eval), dispatchfinding.CodeLocationUnknown)
	for i := range eval.findings {
		if eval.findings[i].Code == dispatchfinding.CodeLocationUnknown {
			assert.Equal(t, dispatchfinding.SeverityWarn, eval.findings[i].Severity)
		}
	}
	assert.False(t, eval.blocked(), "an unknown position is a warning, not a block")

	score := candidateFrom(s, eval)
	assert.Nil(t, score.DeadheadMiles, "unknown, not zero")
	assert.False(t, score.LocationKnown())
	assert.Equal(t, dispatchplanports.OriginUnknown, score.OriginSource)
	assert.Zero(t, costLine(score.Cost, dispatcheconomics.ComponentEmptyMiles))
	assert.Equal(t, int64(100/50*3600*1000), score.EstimatedDriveMs,
		"only the loaded leg can be timed")
}

func TestEvaluate_StaleHOSRunsNoProjection(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.TelematicsActive = true
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	state := hosState(driver.WorkerID, 8*testHourMs)
	state.RecordedAt = testNow - dispatcheligibility.HOSStateMaxAgeSeconds - 3600
	s.HOSByWorker[driver.WorkerID] = state

	eval := evaluateFor(s, driver, pickupAt(chicagoLat, chicagoLon, 200), nil)

	assert.Nil(t, eval.projection, "no projection may run on stale clocks")
	assert.Zero(t, eval.trip.restMs)
	assert.Empty(t, candidateFrom(s, eval).HOSStrategy)
	assert.Equal(t, telematics.FeasibilityVerdictUnknown, eval.verdict)
}

func TestEvaluate_NoTelematicsMeansNoHOSAnywhere(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.Control = dispatchcontrol.NewDefaultDispatchControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	s.HOSByWorker[driver.WorkerID] = hosState(driver.WorkerID, 0)
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.OriginWindowStart = testNow + 12*3600

	eval := evaluateFor(s, driver, move, nil)

	assert.Nil(t, eval.projection)
	for i := range eval.findings {
		assert.NotEqual(t, "hos", eval.findings[i].Field,
			"no HOS findings may surface without a telematics integration")
	}
	assert.NotEqual(t, telematics.FeasibilityVerdictUnknown, eval.verdict,
		"missing HOS data is expected without an integration, not unknown")
}

func TestEvaluate_ClosedPickupWindowWarnsInsteadOfBlocking(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.Control = &dispatchcontrol.DispatchControl{EnforceWorkerPTARestrictions: true}
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	closed := testNow - 20*3600
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.OriginWindowStart = closed - 4*3600
	move.OriginWindowEnd = &closed

	eval := evaluateFor(s, driver, move, nil)

	assert.False(t, eval.trip.appointmentKnown,
		"every driver is equally late to a window already gone")
	assert.Equal(t, closed, eval.trip.windowClosedAt)
	assert.Zero(t, eval.trip.slackMinutes)
	assert.Contains(t, findingCodes(eval), dispatchfinding.CodeWindowPassed)
	assert.False(t, eval.blocked(),
		"a window nobody can make does not block the one driver who ran the load")
}

func TestAppointmentFinding_EnforcementDecidesBlockOrWarn(t *testing.T) {
	t.Parallel()

	late := trip{appointmentKnown: true, slackMinutes: -45}
	control := dispatchcontrol.NewDefaultDispatchControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)

	control.EnforceWorkerPTARestrictions = true
	finding := appointmentFinding(&late, control)
	require.NotNil(t, finding)
	assert.Equal(t, dispatchfinding.CodeAppointmentMissed, finding.Code)
	assert.Equal(t, dispatchfinding.SeverityBlock, finding.Severity)

	control.EnforceWorkerPTARestrictions = false
	finding = appointmentFinding(&late, control)
	require.NotNil(t, finding)
	assert.Equal(t, dispatchfinding.CodeAppointmentAtRisk, finding.Code)
	assert.Equal(t, dispatchfinding.SeverityWarn, finding.Severity)

	assert.Nil(t, appointmentFinding(&trip{appointmentKnown: true, slackMinutes: 30}, control))
	assert.Nil(t, appointmentFinding(&trip{}, control))
}

func TestResolveTrailer_PrefersTheExistingAssignment(t *testing.T) {
	t.Parallel()

	assigned := pulid.MustNew("tr_")
	previous := pulid.MustNew("tr_")

	assert.Equal(t, assigned, resolveTrailer(&dispatchboardports.BoardMove{
		AssignedTrailerID:     assigned,
		PreviousMoveTrailerID: previous,
	}))
	assert.Equal(t, previous, resolveTrailer(&dispatchboardports.BoardMove{
		PreviousMoveTrailerID: previous,
	}))
	assert.True(t, resolveTrailer(&dispatchboardports.BoardMove{}).IsNil())
}

func TestEvaluate_WorkDoneBeforeALateLoadsStaleWindowDoesNotBlockADriverFreeNow(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.OriginWindowStart = testNow - 2*24*3600
	move.DestinationWindowStart = testNow - 2*24*3600 + 8*3600
	s.CommitmentsByWorker[driver.WorkerID] = []*dispatchboardports.WorkerCommitment{{
		WorkerID:    driver.WorkerID,
		MoveID:      pulid.MustNew("smv_"),
		WindowStart: testNow - 2*24*3600,
		WindowEnd:   testNow - 2*24*3600 + 6*3600,
		ExpectedEnd: testNow - 2*24*3600 + 6*3600,
	}}

	eval := evaluateFor(s, driver, move, nil)

	assert.NotContains(t, findingCodes(eval), dispatchfinding.CodeWorkerOverlappingMove,
		"the driver did that work two days ago and is free to take the load now")
}

func TestEvaluate_ADriverWithLaterWorkTakesTheLoadAfterItAndPaysTheWait(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.OriginWindowStart = testNow
	committedEnd := testNow + 10*3600
	s.CommitmentsByWorker[driver.WorkerID] = []*dispatchboardports.WorkerCommitment{{
		WorkerID:    driver.WorkerID,
		MoveID:      pulid.MustNew("smv_"),
		WindowStart: testNow + 2*3600,
		WindowEnd:   committedEnd,
		ExpectedEnd: committedEnd,
	}}

	eval := evaluateFor(s, driver, move, nil)

	assert.GreaterOrEqual(t, eval.trip.departure, committedEnd,
		"the trip is planned after the work the driver already holds")
	assert.NotContains(t, findingCodes(eval), dispatchfinding.CodeWorkerOverlappingMove)
	assert.Positive(t, eval.features.PickupLateMinutes, "waiting for the driver shows up as cost")
}
