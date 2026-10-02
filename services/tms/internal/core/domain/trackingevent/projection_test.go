package trackingevent_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/trackingevent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	move  *shipment.ShipmentMove
	stops []*shipment.Stop
}

func newFixture(stopCount int) *fixture {
	f := &fixture{move: &shipment.ShipmentMove{ID: pulid.MustNew("smv_"), Status: shipment.MoveStatusAssigned}}
	for i := range stopCount {
		stop := &shipment.Stop{
			ID:         pulid.MustNew("stp_"),
			Sequence:   int64(i),
			LocationID: pulid.MustNew("loc_"),
			Status:     shipment.StopStatusNew,
		}
		f.stops = append(f.stops, stop)
		f.move.Stops = append(f.move.Stops, stop)
	}
	return f
}

func (f *fixture) clone() *fixture {
	c := &fixture{move: &shipment.ShipmentMove{ID: f.move.ID, Status: f.move.Status}}
	for _, stop := range f.stops {
		copied := *stop
		c.stops = append(c.stops, &copied)
		c.move.Stops = append(c.move.Stops, &copied)
	}
	return c
}

func event(
	stop *shipment.Stop,
	kind shipment.VisitKind,
	source trackingevent.Source,
	key string,
	at int64,
) *trackingevent.TrackingEvent {
	return &trackingevent.TrackingEvent{
		ID:        pulid.MustNew("tkev_"),
		StopID:    stop.ID,
		Kind:      kind,
		Source:    source,
		SourceKey: key,
		EventAt:   at,
	}
}

func settleAll(
	t *testing.T,
	f *fixture,
	events []*trackingevent.TrackingEvent,
	opts trackingevent.Options,
) trackingevent.Projection {
	t.Helper()
	for range 2 * len(f.stops) + 2 {
		projection := trackingevent.Project(f.move, events, opts)
		if !projection.HasChanges() {
			return projection
		}
		_, err := f.move.ApplyStopActualChanges(projection.Changes)
		require.NoError(t, err)
	}
	t.Fatal("projection did not settle")
	return trackingevent.Projection{}
}

type actual struct {
	arrival   *int64
	departure *int64
}

func actuals(f *fixture) []actual {
	out := make([]actual, 0, len(f.stops))
	for _, stop := range f.stops {
		out = append(out, actual{arrival: stop.ActualArrival, departure: stop.ActualDeparture})
	}
	return out
}

func ptr(v int64) *int64 { return &v }

func scenario(f *fixture) []*trackingevent.TrackingEvent {
	s1, s2, s3 := f.stops[0], f.stops[1], f.stops[2]
	return []*trackingevent.TrackingEvent{
		event(s1, shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1000),
		event(s1, shipment.VisitArrival, trackingevent.SourceEDI, "e-1", 990),
		event(s1, shipment.VisitArrival, trackingevent.SourceTelematics, "t-2", 1005),
		event(s1, shipment.VisitDeparture, trackingevent.SourceTelematics, "t-3", 2000),
		event(s1, shipment.VisitDeparture, trackingevent.SourceDriver, "d-1", 2010),
		event(s2, shipment.VisitArrival, trackingevent.SourceTelematics, "t-4", 3000),
		event(s2, shipment.VisitArrival, trackingevent.SourceTelematics, "t-5", 3000),
		event(s2, shipment.VisitDeparture, trackingevent.SourceEDI, "e-2", 4000),
		event(s3, shipment.VisitArrival, trackingevent.SourceTelematics, "t-6", 5000),
		event(s3, shipment.VisitDeparture, trackingevent.SourceTelematics, "t-7", 6000),
	}
}

func TestProjectGivesTheSameActualsInAnyArrivalOrder(t *testing.T) {
	base := newFixture(3)
	events := scenario(base)

	inOrder := base.clone()
	want := settleAll(t, inOrder, events, trackingevent.Options{})
	wantActuals := actuals(inOrder)

	assert.Equal(t, []actual{
		{arrival: ptr(1000), departure: ptr(2010)},
		{arrival: ptr(3000), departure: ptr(4000)},
		{arrival: ptr(5000), departure: ptr(6000)},
	}, wantActuals)

	rng := rand.New(rand.NewPCG(7, 11))
	for trial := range 500 {
		order := make([]*trackingevent.TrackingEvent, len(events))
		copy(order, events)
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

		f := base.clone()
		received := make([]*trackingevent.TrackingEvent, 0, len(order))
		var last trackingevent.Projection
		for _, next := range order {
			received = append(received, next)
			last = settleAll(t, f, received, trackingevent.Options{})
		}

		require.Equal(t, wantActuals, actuals(f), fmt.Sprintf("trial %d", trial))
		require.Equal(t, want.Verdicts, last.Verdicts, fmt.Sprintf("trial %d", trial))
	}
}

func TestProjectVerdictsForTheScenario(t *testing.T) {
	f := newFixture(3)
	events := scenario(f)
	projection := settleAll(t, f, events, trackingevent.Options{})

	byKey := make(map[string]trackingevent.Verdict, len(events))
	for _, e := range events {
		byKey[e.SourceKey] = projection.Verdicts[e.ID]
	}

	assert.Equal(t, trackingevent.OutcomeApplied, byKey["t-1"].Outcome)
	assert.Equal(t, trackingevent.OutcomeSuperseded, byKey["e-1"].Outcome)
	assert.Equal(t, trackingevent.OutcomeSuperseded, byKey["t-2"].Outcome)
	assert.Equal(t, trackingevent.OutcomeSuperseded, byKey["t-3"].Outcome)
	assert.Equal(t, trackingevent.OutcomeApplied, byKey["d-1"].Outcome)
	assert.Equal(t, trackingevent.OutcomeApplied, byKey["t-4"].Outcome)
	assert.Equal(t, trackingevent.OutcomeDuplicate, byKey["t-5"].Outcome)
	assert.Equal(t, trackingevent.OutcomeApplied, byKey["e-2"].Outcome)
	assert.NotEmpty(t, byKey["e-1"].Reason)
}

func TestProjectHoldsALaterStopUntilTheEarlierStopsComplete(t *testing.T) {
	f := newFixture(2)
	early := event(f.stops[1], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 3000)

	projection := trackingevent.Project(f.move, []*trackingevent.TrackingEvent{early}, trackingevent.Options{})

	assert.False(t, projection.HasChanges())
	assert.Equal(t, trackingevent.OutcomePending, projection.Verdicts[early.ID].Outcome)
	assert.Equal(t, "Waiting for the earlier stops on this load", projection.Verdicts[early.ID].Reason)
}

func TestProjectRefusesTimesThatBreakTheTimeline(t *testing.T) {
	f := newFixture(2)
	f.stops[0].ActualArrival = ptr(1000)
	f.stops[0].ActualDeparture = ptr(2000)
	arrival := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceDriver, "d-1", 1000)
	departure := event(f.stops[0], shipment.VisitDeparture, trackingevent.SourceDriver, "d-2", 2000)
	tooEarly := event(f.stops[1], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1500)

	projection := trackingevent.Project(
		f.move,
		[]*trackingevent.TrackingEvent{arrival, departure, tooEarly},
		trackingevent.Options{},
	)

	assert.False(t, projection.HasChanges())
	assert.Equal(t, trackingevent.OutcomeRefused, projection.Verdicts[tooEarly.ID].Outcome)
	assert.Equal(t, "Earlier than the previous stop's departure", projection.Verdicts[tooEarly.ID].Reason)
	assert.Equal(t, trackingevent.OutcomeApplied, projection.Verdicts[arrival.ID].Outcome)
}

func TestProjectKeepsATimeEnteredOnTheShipment(t *testing.T) {
	f := newFixture(1)
	f.stops[0].ActualArrival = ptr(900)
	reported := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1000)

	projection := trackingevent.Project(f.move, []*trackingevent.TrackingEvent{reported}, trackingevent.Options{})

	assert.False(t, projection.HasChanges())
	assert.Equal(t, trackingevent.OutcomeSuperseded, projection.Verdicts[reported.ID].Outcome)
}

func TestProjectCorrectsACompletedLoadButNotABilledOne(t *testing.T) {
	f := newFixture(1)
	f.move.Status = shipment.MoveStatusCompleted
	f.stops[0].ActualArrival = ptr(1005)
	f.stops[0].ActualDeparture = ptr(2000)
	recorded := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceTelematics, "t-2", 1005)
	departed := event(f.stops[0], shipment.VisitDeparture, trackingevent.SourceTelematics, "t-3", 2000)
	earlier := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1000)
	events := []*trackingevent.TrackingEvent{recorded, departed, earlier}

	open := trackingevent.Project(f.move, events, trackingevent.Options{})
	require.Len(t, open.Changes, 1)
	assert.Equal(t, ptr(1000), open.Changes[0].Arrival)

	locked := trackingevent.Project(f.move, events, trackingevent.Options{LockedReason: "This load has been invoiced"})
	assert.False(t, locked.HasChanges())
	assert.Equal(t, trackingevent.OutcomeApplied, locked.Verdicts[recorded.ID].Outcome)
	assert.Equal(t, trackingevent.OutcomeSuperseded, locked.Verdicts[earlier.ID].Outcome)
	assert.Equal(t, "This load has been invoiced", locked.Verdicts[earlier.ID].Reason)
}

func TestProjectRefusesEventsOnACanceledLoadOrARemovedStop(t *testing.T) {
	f := newFixture(1)
	f.move.Status = shipment.MoveStatusCanceled
	onLoad := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1000)
	removed := &trackingevent.TrackingEvent{
		ID: pulid.MustNew("tkev_"), StopID: pulid.MustNew("stp_"),
		Kind: shipment.VisitArrival, Source: trackingevent.SourceEDI, SourceKey: "e-1", EventAt: 1000,
	}

	projection := trackingevent.Project(f.move, []*trackingevent.TrackingEvent{onLoad, removed}, trackingevent.Options{})

	assert.False(t, projection.HasChanges())
	assert.Equal(t, trackingevent.OutcomeRefused, projection.Verdicts[onLoad.ID].Outcome)
	assert.Equal(t, "This load has been canceled", projection.Verdicts[onLoad.ID].Reason)
	assert.Equal(t, trackingevent.OutcomeRefused, projection.Verdicts[removed.ID].Outcome)
	assert.Equal(t, "This stop is no longer on the load", projection.Verdicts[removed.ID].Reason)
}

func TestProjectPrefersAPersonOverADevice(t *testing.T) {
	f := newFixture(1)
	device := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceTelematics, "t-1", 1000)
	person := event(f.stops[0], shipment.VisitArrival, trackingevent.SourceDispatcher, "m-1", 1100)

	projection := settleAll(t, f, []*trackingevent.TrackingEvent{device, person}, trackingevent.Options{})

	assert.Equal(t, ptr(1100), f.stops[0].ActualArrival)
	assert.Equal(t, trackingevent.OutcomeApplied, projection.Verdicts[person.ID].Outcome)
	assert.Equal(t, trackingevent.OutcomeSuperseded, projection.Verdicts[device.ID].Outcome)
}
