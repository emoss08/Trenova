package etaprojection

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	now  = int64(1_800_000_000)
	hour = int64(3600)
)

type fixture struct {
	shipment *shipment.Shipment
	move     *shipment.ShipmentMove
	pickup   *shipment.Stop
	delivery *shipment.Stop
	tractor  pulid.ID
	worker   pulid.ID
	input    *Input
}

func ptr[T any](v T) *T { return &v }

func newFixture() *fixture {
	tractor := pulid.MustNew("tr_")
	worker := pulid.MustNew("wrk_")
	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Sequence:             0,
		Type:                 shipment.StopTypePickup,
		Status:               shipment.StopStatusNew,
		ScheduledWindowStart: now + 2*hour,
		ScheduledWindowEnd:   ptr(now + 4*hour),
		Location:             &location.Location{Latitude: ptr(41.0), Longitude: ptr(-87.0)},
	}
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Sequence:             1,
		Type:                 shipment.StopTypeDelivery,
		Status:               shipment.StopStatusNew,
		ScheduledWindowStart: now + 8*hour,
		ScheduledWindowEnd:   ptr(now + 10*hour),
		Location:             &location.Location{Latitude: ptr(40.0), Longitude: ptr(-83.0)},
	}
	move := &shipment.ShipmentMove{
		ID:       pulid.MustNew("smv_"),
		Status:   shipment.MoveStatusAssigned,
		Sequence: 0,
		Stops:    []*shipment.Stop{delivery, pickup},
	}
	sp := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         shipment.StatusAssigned,
		Moves:          []*shipment.ShipmentMove{move},
	}

	return &fixture{
		shipment: sp,
		move:     move,
		pickup:   pickup,
		delivery: delivery,
		tractor:  tractor,
		worker:   worker,
		input: &Input{
			Now:      now,
			Shipment: sp,
			Assignments: map[pulid.ID]*shipmentports.BoardMove{
				move.ID: {
					MoveID:            move.ID,
					AssignedWorkerID:  worker,
					AssignedTractorID: tractor,
					CoverageType:      string(shipment.MoveCoverageTypeDriver),
				},
			},
			Positions:  map[pulid.ID]*telematics.VehiclePosition{},
			HOS:        map[pulid.ID]*telematics.WorkerHOSState{},
			HOSExempt:  map[pulid.ID]bool{},
			CheckCalls: map[pulid.ID]*checkcall.CheckCall{},
			Settings:   Settings{ThresholdMinutes: 60, DwellMinutes: 60},
		},
	}
}

func (f *fixture) withPosition(recordedAt int64) {
	f.input.Positions[f.tractor] = &telematics.VehiclePosition{
		TractorID:  f.tractor,
		Latitude:   41.5,
		Longitude:  -87.5,
		RecordedAt: recordedAt,
	}
}

func (f *fixture) withHOS(driveMs, shiftMs int64) {
	f.input.HOS[f.worker] = &telematics.WorkerHOSState{
		WorkerID:         f.worker,
		DriveRemainingMs: driveMs,
		ShiftRemainingMs: shiftMs,
		CycleRemainingMs: 60 * hour * msPerSecond,
		BreakRemainingMs: 8 * hour * msPerSecond,
		RecordedAt:       now - 600,
	}
}

func legs(driveHours ...float64) []routeestimateports.LegEstimate {
	out := make([]routeestimateports.LegEstimate, 0, len(driveHours))
	for _, h := range driveHours {
		out = append(out, routeestimateports.LegEstimate{
			Miles:        h * 50,
			DriveSeconds: int64(h * float64(hour)),
			Routed:       true,
		})
	}
	return out
}

func project(
	t *testing.T,
	in *Input,
	estimates []routeestimateports.LegEstimate,
) []*stopeta.StopETA {
	t.Helper()
	plan := Prepare(in)
	require.Len(t, plan.Legs(), len(estimates))
	return plan.Project(estimates)
}

func TestProject_ChainsStopsFromFreshPosition(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 300)

	rows := project(t, f.input, legs(1, 5))
	require.Len(t, rows, 2)

	pickup := rows[0]
	assert.Equal(t, f.pickup.ID, pickup.StopID)
	assert.Equal(t, stopeta.BasisPosition, pickup.Basis)
	assert.Equal(t, stopeta.DriveMethodRouted, pickup.DriveMethod)
	assert.Equal(t, now+hour, *pickup.EstimatedArrival)
	assert.Equal(t, now+3*hour, *pickup.EstimatedDeparture)
	assert.Equal(t, int64(180), *pickup.SlackMinutes)
	assert.Equal(t, stopeta.VerdictOnTime, pickup.Verdict)
	assert.False(t, pickup.InputStale)
	assert.Equal(t, now-300, *pickup.InputAt)

	delivery := rows[1]
	assert.Equal(t, now+8*hour, *delivery.EstimatedArrival)
	assert.Equal(t, int64(120), *delivery.SlackMinutes)
	assert.Equal(t, stopeta.VerdictOnTime, delivery.Verdict)
	assert.Equal(t, int64(360), delivery.DriveMinutes)
	assert.InDelta(t, 300.0, delivery.MilesRemaining, 0.01)
}

func TestProject_SlackBelowThresholdIsAtRiskAndPastCutoffIsLate(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 60)

	rows := project(t, f.input, legs(3.5, 6))
	require.Len(t, rows, 2)
	assert.Equal(t, stopeta.VerdictAtRisk, rows[0].Verdict)
	assert.Equal(t, int64(30), *rows[0].SlackMinutes)
	assert.Equal(t, stopeta.VerdictLate, rows[1].Verdict)
	assert.Equal(t, int64(-30), *rows[1].SlackMinutes)
}

func TestProject_HoursOfServiceAddRest(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 60)
	f.withHOS(hour*msPerSecond, 14*hour*msPerSecond)

	rows := project(t, f.input, legs(2, 1))
	require.Len(t, rows, 2)
	assert.Equal(t, int64(600), rows[0].RestMinutes)
	assert.Equal(t, now+12*hour, *rows[0].EstimatedArrival)
	assert.Equal(t, stopeta.VerdictLate, rows[0].Verdict)
	assert.Contains(t, rows[0].Reason, "10 hour reset")
}

func TestProject_ExemptDriverSkipsRest(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 60)
	f.withHOS(hour*msPerSecond, 14*hour*msPerSecond)
	f.input.HOSExempt[f.worker] = true

	rows := project(t, f.input, legs(2, 1))
	assert.Zero(t, rows[0].RestMinutes)
	assert.Contains(t, rows[0].Reason, "exempt")
}

func TestProject_OverdueStopWithoutInputsIsLate(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pickup.ScheduledWindowStart = now - 3*hour
	f.pickup.ScheduledWindowEnd = ptr(now - hour)

	rows := project(t, f.input, nil)
	require.Len(t, rows, 2)
	assert.Equal(t, stopeta.BasisNone, rows[0].Basis)
	assert.Nil(t, rows[0].EstimatedArrival)
	assert.Equal(t, stopeta.VerdictLate, rows[0].Verdict)
	assert.Equal(t, int64(-60), *rows[0].SlackMinutes)
	assert.Equal(t, stopeta.VerdictUnknown, rows[1].Verdict)
}

func TestProject_TruckAtStopDepartsAfterDwell(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pickup.ScheduledWindowStart = now - hour
	f.pickup.ActualArrival = ptr(now - 30*60)
	f.pickup.Status = shipment.StopStatusInTransit

	rows := project(t, f.input, legs(5))
	require.Len(t, rows, 2)
	assert.Equal(t, stopeta.BasisAtStop, rows[0].Basis)
	assert.True(t, rows[0].Arrived)
	assert.False(t, rows[1].Arrived)
	assert.Equal(t, now-30*60, *rows[0].EstimatedArrival)
	assert.Equal(t, now+30*60, *rows[0].EstimatedDeparture)
	assert.Equal(t, stopeta.VerdictOnTime, rows[0].Verdict)
	assert.Equal(t, now+30*60+5*hour, *rows[1].EstimatedArrival)
}

func TestProject_CarrierReportedArrival(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.input.Assignments[f.move.ID] = &shipmentports.BoardMove{
		MoveID:            f.move.ID,
		CoverageType:      string(shipment.MoveCoverageTypeCarrier),
		AssignedCarrierID: pulid.MustNew("car_"),
	}
	f.input.CheckCalls[f.move.ID] = &checkcall.CheckCall{
		ShipmentMoveID:   f.move.ID,
		ReportedAt:       now - 1800,
		EstimatedArrival: ptr(now + 3*hour + 30*60),
	}

	rows := project(t, f.input, legs(4))
	require.Len(t, rows, 2)
	assert.Equal(t, stopeta.BasisCarrierReported, rows[0].Basis)
	assert.Equal(t, stopeta.DriveMethodReported, rows[0].DriveMethod)
	assert.Equal(t, stopeta.VerdictAtRisk, rows[0].Verdict)
	assert.Equal(t, now+3*hour+30*60+hour+4*hour, *rows[1].EstimatedArrival)
}

func TestProject_NewerCheckCallBeatsStalePosition(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 5*hour)
	f.input.CheckCalls[f.move.ID] = &checkcall.CheckCall{
		ShipmentMoveID: f.move.ID,
		ReportedAt:     now - 600,
		Latitude:       ptr(41.2),
		Longitude:      ptr(-87.2),
	}

	rows := project(t, f.input, legs(1, 5))
	assert.Equal(t, stopeta.BasisCheckCall, rows[0].Basis)
	assert.False(t, rows[0].InputStale)
}

func TestProject_StalePositionIsMarked(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 3*hour)

	rows := project(t, f.input, legs(1, 5))
	assert.Equal(t, stopeta.BasisPosition, rows[0].Basis)
	assert.True(t, rows[0].InputStale)
	assert.True(t, rows[1].InputStale)
}

func TestProject_LastDepartureClampsToNow(t *testing.T) {
	t.Parallel()

	f := newFixture()
	origin := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Sequence:             0,
		Status:               shipment.StopStatusCompleted,
		ScheduledWindowStart: now - 10*hour,
		ActualArrival:        ptr(now - 9*hour),
		ActualDeparture:      ptr(now - 8*hour),
		Location:             &location.Location{Latitude: ptr(42.0), Longitude: ptr(-88.0)},
	}
	f.pickup.Sequence = 1
	f.delivery.Sequence = 2
	f.move.Stops = append(f.move.Stops, origin)

	rows := project(t, f.input, legs(2, 5))
	require.Len(t, rows, 2)
	assert.Equal(t, stopeta.BasisLastDeparture, rows[0].Basis)
	assert.Equal(t, now, *rows[0].EstimatedArrival)
	assert.True(t, rows[0].InputStale)
}

func TestProject_MissingCoordinatesBreakTheChain(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.withPosition(now - 60)
	f.pickup.Location = nil

	rows := project(t, f.input, nil)
	require.Len(t, rows, 2)
	assert.Nil(t, rows[0].EstimatedArrival)
	assert.Equal(t, stopeta.VerdictUnknown, rows[0].Verdict)
	assert.Nil(t, rows[1].EstimatedArrival)
}

func TestPrepare_SkipsFinishedShipmentsAndStops(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.shipment.Status = shipment.StatusCompleted
	assert.Empty(t, Prepare(f.input).Project(nil))

	f = newFixture()
	f.withPosition(now - 60)
	f.pickup.Status = shipment.StopStatusCompleted
	f.pickup.ActualDeparture = ptr(now - 600)
	rows := project(t, f.input, legs(4))
	require.Len(t, rows, 1)
	assert.Equal(t, f.delivery.ID, rows[0].StopID)
}

func TestFloorMinutes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(1), floorMinutes(119))
	assert.Equal(t, int64(-1), floorMinutes(-1))
	assert.Equal(t, int64(-2), floorMinutes(-61))
	assert.Equal(t, int64(0), floorMinutes(0))
}
