package shipmenttracking

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacts_DateWhatTheTruckReported(t *testing.T) {
	t.Parallel()

	snapshot := &Snapshot{
		ProNumber: "SEED-DET-001",
		Status:    "InTransit",
		Customer:  "Acme Manufacturing",
		Flags:     []string{"delivery is overdue"},
		Moves: []MoveSnapshot{{
			ID: "sm_1", Driver: "Jane Doe", Tractor: "TRC-003", Coverage: "driver",
		}},
		NextStop: &StopSnapshot{
			MoveID: "sm_1", Type: "Delivery", Location: "Chicago DC", City: "Chicago", State: "IL",
			ScheduledEndText: "Oct 6 16:00", ActualArrival: 10, ActualArrivalText: "Oct 6 14:32",
		},
		Position: &PositionSnapshot{
			Tractor: "TRC-003", FormattedLocation: "Chicago, IL", RecordedAt: 500, Stale: true,
		},
		Driver: &DriverSnapshot{
			Name: "Jane Doe", DutyStatus: "OnDuty", DriveRemainingMinutes: 125, RecordedAt: 400,
		},
		Estimate: &ArrivalEstimate{
			StopLabel: "Chicago DC", EstimatedArrivalText: "Oct 6 15:10", SlackMinutes: 50,
			Verdict: VerdictOnTime,
		},
	}

	facts := snapshot.Facts()
	byName := make(map[string]Fact, len(facts))
	for _, fact := range facts {
		byName[fact.Name] = fact
	}

	assert.Equal(t, "InTransit for Acme Manufacturing", byName["status"].Value)
	assert.Equal(t, "delivery is overdue", byName["needs attention"].Value)
	assert.Contains(t, byName["next stop"].Value, "delivery at Chicago DC in Chicago, IL")
	assert.Contains(t, byName["next stop"].Value, "arrived Oct 6 14:32 and not yet departed")
	assert.Equal(t, "Jane Doe on TRC-003", byName["covered by"].Value)

	require.Contains(t, byName, "last position")
	assert.EqualValues(t, 500, byName["last position"].SeenAt)
	assert.True(t, byName["last position"].Stale)
	assert.EqualValues(t, 400, byName["driver hours"].SeenAt)
	assert.Contains(t, byName["driver hours"].Value, "2h 05m drive")
	assert.EqualValues(t, 500, byName["estimated arrival"].SeenAt,
		"an estimate is as old as the position it was worked from")
}

func TestFacts_SayWhenNobodyIsOnTheLoad(t *testing.T) {
	t.Parallel()

	snapshot := &Snapshot{
		ProNumber: "SEED-SHP-010",
		Status:    "Delayed",
		Moves:     []MoveSnapshot{{ID: "sm_1"}},
		NextStop:  &StopSnapshot{MoveID: "sm_1", Type: "Delivery", City: "Los Angeles", Overdue: true, LateMinutes: 90},
	}

	facts := snapshot.Facts()
	values := make(map[string]string, len(facts))
	for _, fact := range facts {
		values[fact.Name] = fact.Value
	}

	assert.Contains(t, values["covered by"], "needs a driver")
	assert.Contains(t, values["next stop"], "overdue by 1h 30m")
	assert.NotContains(t, values, "estimated arrival")
}

func TestFacts_FinishedMoveIsNotAskedForADriver(t *testing.T) {
	t.Parallel()

	snapshot := &Snapshot{
		Status: "Completed",
		Moves:  []MoveSnapshot{{ID: "sm_1", Status: "Completed", Coverage: "Unassigned"}},
	}

	for _, fact := range snapshot.Facts() {
		if fact.Name == "covered by" {
			assert.Equal(t, "nobody on record", fact.Value)
			return
		}
	}
	t.Fatal("no coverage fact")
}

func TestFacts_NameTheFreight(t *testing.T) {
	t.Parallel()

	weight, pieces := int64(38000), int64(20)
	snapshot := &Snapshot{Status: "New", Freight: freightText(&weight, &pieces)}

	for _, fact := range snapshot.Facts() {
		if fact.Name == "freight" {
			assert.Equal(t, "38,000 lb, 20 pieces", fact.Value)
			return
		}
	}
	t.Fatal("no freight fact")
}
