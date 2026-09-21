package shipmenttracking

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_790_000_000)

func ptr[T any](v T) *T { return &v }

func place(name, city, state string, lat, lon float64) *location.Location {
	return &location.Location{
		Name:      name,
		City:      city,
		State:     &usstate.UsState{Abbreviation: state},
		Latitude:  ptr(lat),
		Longitude: ptr(lon),
	}
}

// A two-stop move: picked up in Dallas an hour late, headed to Houston with
// a window that closes in three hours.
func inTransitShipment() (*shipment.Shipment, *shipment.ShipmentMove) {
	move := &shipment.ShipmentMove{
		ID:           pulid.MustNew("smv_"),
		Sequence:     0,
		Status:       shipment.MoveStatusInTransit,
		Loaded:       true,
		CoverageType: shipment.MoveCoverageTypeDriver,
		Distance:     ptr(240.0),
		Stops: []*shipment.Stop{
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             1,
				Type:                 shipment.StopTypeDelivery,
				Status:               shipment.StopStatusNew,
				ScheduledWindowStart: now + 2*3600,
				ScheduledWindowEnd:   ptr(now + 3*3600),
				Location:             place("Houston DC", "Houston", "TX", 29.7604, -95.3698),
			},
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             0,
				Type:                 shipment.StopTypePickup,
				Status:               shipment.StopStatusCompleted,
				ScheduledWindowStart: now - 5*3600,
				ScheduledWindowEnd:   ptr(now - 4*3600),
				ActualArrival:        ptr(now - 3*3600),
				ActualDeparture:      ptr(now - 2*3600),
				Location:             place("Dallas Yard", "Dallas", "TX", 32.7767, -96.797),
			},
		},
	}

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S12345",
		Status:    shipment.StatusInTransit,
		Customer:  &customer.Customer{Name: "Acme"},
		Moves:     []*shipment.ShipmentMove{move},
	}, move
}

func TestBuild_ReadsStopsInOrderAndFlagsTheLateOne(t *testing.T) {
	t.Parallel()

	sp, _ := inTransitShipment()
	snapshot := Build(Input{Shipment: sp, Now: now, Timezone: "America/Chicago"})

	require.Len(t, snapshot.Moves, 1)
	stops := snapshot.Moves[0].Stops
	require.Len(t, stops, 2)
	assert.Equal(t, "Pickup", stops[0].Type, "stops come back in sequence order")
	assert.True(t, stops[0].Late)
	assert.EqualValues(t, 60, stops[0].LateMinutes)
	assert.False(t, stops[1].Late)
	assert.True(t, stops[1].Next)

	require.NotNil(t, snapshot.NextStop)
	assert.Equal(t, "Houston DC", snapshot.NextStop.Location)
	assert.Contains(t, snapshot.Flags, "Stop 0 (Pickup at Dallas Yard in Dallas, TX) arrived 60 minutes late")
	assert.Contains(t, snapshot.Summary, "PRO S12345 is InTransit for Acme.")
	assert.Contains(t, snapshot.Summary, "Next: delivery Houston DC in Houston, TX")
}

func TestBuild_EstimatesArrivalFromTheTractorsPosition(t *testing.T) {
	t.Parallel()

	sp, move := inTransitShipment()
	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")
	snapshot := Build(Input{
		Shipment: sp,
		Now:      now,
		Timezone: "UTC",
		Assignments: map[pulid.ID]*repositories.BoardMove{
			move.ID: {
				AssignedWorkerID:    workerID,
				AssignedWorkerName:  "Maria Ortiz",
				AssignedTractorID:   tractorID,
				AssignedTractorCode: "T-104",
				CoverageType:        "driver",
			},
		},
		Positions: map[pulid.ID]*telematics.VehiclePosition{
			// Huntsville, TX: roughly 70 miles north of Houston.
			tractorID: {
				TractorID:         tractorID,
				Latitude:          30.7235,
				Longitude:         -95.5508,
				FormattedLocation: "Huntsville, TX",
				SpeedMph:          62,
				EngineState:       telematics.EngineStateOn,
				RecordedAt:        now - 300,
			},
		},
		HOS: map[pulid.ID]*telematics.WorkerHOSState{
			workerID: {
				WorkerID:         workerID,
				DutyStatus:       telematics.DutyStatusDriving,
				DriveRemainingMs: 5 * 3_600_000,
				ShiftRemainingMs: 7 * 3_600_000,
				RecordedAt:       now - 600,
			},
		},
	})

	require.NotNil(t, snapshot.Position)
	assert.Equal(t, "T-104", snapshot.Position.Tractor)
	assert.False(t, snapshot.Position.Stale)
	assert.EqualValues(t, 5, snapshot.Position.AgeMinutes)

	require.NotNil(t, snapshot.Driver)
	assert.Equal(t, "Maria Ortiz", snapshot.Driver.Name)
	assert.EqualValues(t, 300, snapshot.Driver.DriveRemainingMinutes)

	require.NotNil(t, snapshot.Estimate)
	assert.InDelta(t, 83, snapshot.Estimate.MilesRemaining, 8, "straight line widened for roads")
	assert.Equal(t, VerdictOnTime, snapshot.Estimate.Verdict)
	assert.Positive(t, snapshot.Estimate.SlackMinutes)
	assert.Contains(t, snapshot.Estimate.Basis, "Not routed")
	assert.Equal(t, "driver", snapshot.Moves[0].Coverage)
	assert.Len(t, snapshot.Flags, 1, "only the late pickup; the position is fresh and the move covered")
}

func TestBuild_CallsAnEstimatePastTheWindowLateAndAStalePositionStale(t *testing.T) {
	t.Parallel()

	sp, move := inTransitShipment()
	// The window closes in thirty minutes and the truck is 200 miles out.
	sp.Moves[0].Stops[0].ScheduledWindowEnd = ptr(now + 1800)
	tractorID := pulid.MustNew("trc_")
	snapshot := Build(Input{
		Shipment: sp,
		Now:      now,
		Timezone: "UTC",
		Assignments: map[pulid.ID]*repositories.BoardMove{
			move.ID: {AssignedTractorID: tractorID, CoverageType: "driver"},
		},
		Positions: map[pulid.ID]*telematics.VehiclePosition{
			tractorID: {
				TractorID:  tractorID,
				Latitude:   32.7767,
				Longitude:  -96.797,
				RecordedAt: now - 2*3600,
			},
		},
	})

	require.NotNil(t, snapshot.Estimate)
	assert.Equal(t, VerdictLate, snapshot.Estimate.Verdict)
	assert.Negative(t, snapshot.Estimate.SlackMinutes)
	require.NotNil(t, snapshot.Position)
	assert.True(t, snapshot.Position.Stale)
	assert.Contains(t, snapshot.Flags, "The last position is 120 minutes old")
	assert.Contains(t, snapshot.Summary, "(stale)")
}

func TestBuild_SaysWhenThereIsNothingToEstimateFrom(t *testing.T) {
	t.Parallel()

	sp, move := inTransitShipment()
	snapshot := Build(Input{
		Shipment: sp,
		Now:      now,
		Timezone: "UTC",
		Assignments: map[pulid.ID]*repositories.BoardMove{
			move.ID: {AssignedTractorID: pulid.MustNew("trc_"), CoverageType: "driver"},
		},
	})

	assert.Nil(t, snapshot.Position)
	require.NotNil(t, snapshot.Estimate)
	assert.Equal(t, VerdictUnknown, snapshot.Estimate.Verdict)
	assert.Contains(t, snapshot.Estimate.Basis, "no position on file")
	assert.Contains(t, snapshot.Flags, "No telematics position is on file for the assigned tractor")
}

func TestBuild_FlagsAnOverdueStopAndAnUncoveredMove(t *testing.T) {
	t.Parallel()

	sp, _ := inTransitShipment()
	sp.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned
	sp.Moves[0].Stops[0].ScheduledWindowEnd = ptr(now - 45*60)

	snapshot := Build(Input{Shipment: sp, Now: now, Timezone: "UTC"})

	next := snapshot.NextStop
	require.NotNil(t, next)
	assert.True(t, next.Overdue)
	assert.EqualValues(t, 45, next.LateMinutes)
	assert.Contains(t, snapshot.Flags, "Move 0 has no driver or carrier")
	assert.Contains(t, snapshot.Flags, "Stop 1 (Delivery at Houston DC in Houston, TX) is 45 minutes past its window with no arrival recorded")
	require.NotNil(t, snapshot.Estimate)
	assert.Equal(t, VerdictUnknown, snapshot.Estimate.Verdict, "nobody is on the move, so nothing to estimate from")
}

func TestBuild_ADeliveredShipmentHasNoNextStop(t *testing.T) {
	t.Parallel()

	sp, _ := inTransitShipment()
	for _, stop := range sp.Moves[0].Stops {
		stop.Status = shipment.StopStatusCompleted
		stop.ActualArrival = ptr(now - 7200)
	}
	sp.Status = shipment.StatusCompleted

	snapshot := Build(Input{Shipment: sp, Now: now, Timezone: "UTC"})

	assert.Nil(t, snapshot.NextStop)
	assert.Nil(t, snapshot.Estimate)
	assert.Contains(t, snapshot.Summary, "Every stop is complete.")
}
