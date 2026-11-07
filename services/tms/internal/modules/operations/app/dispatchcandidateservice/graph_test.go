package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildGraph(
	t *testing.T,
	routes fakeRoutes,
	s *FleetSnapshot,
	moves ...*dispatchboardports.BoardMove,
) *Graph {
	t.Helper()

	return newTestService(routes).BuildGraph(t.Context(), &GraphRequest{
		Moves:    moves,
		Snapshot: s,
	})
}

func TestBuildGraph_KeepsTheCheapestCandidatesPerMove(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	withCandidatesPerMove(s, 5)
	for i := range 8 {
		addPlacedDriver(s, string(rune('A'+i)), chicagoLat+0.05*float64(i+1), chicagoLon)
	}
	move := pickupAt(chicagoLat, chicagoLon, 200)

	graph := buildGraph(t, fakeRoutes{}, s, move)

	require.Len(t, graph.Arcs, 1)
	require.Len(t, graph.Arcs[0], 5, "only the top five candidates become arcs")
	for i, arc := range graph.Arcs[0] {
		assert.Equal(t, i, arc.Driver, "the nearest drivers are the cheapest")
		assert.Equal(t, s.Drivers[i].WorkerID, arc.Candidate.WorkerID)
		assert.Equal(t, move.MoveID, arc.Candidate.MoveID)
		if i > 0 {
			assert.Greater(t, arc.CostCents, graph.Arcs[0][i-1].CostCents)
		}
	}
	assert.Nil(t, graph.Rejected[0], "every candidate looked at was admissible")
	assert.Equal(t, s.Drivers, graph.Drivers)
}

func TestBuildGraph_RecordsTheCheapestDriverPastTheDeadheadCap(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.Control = &dispatchcontrol.DispatchControl{AutoAssignMaxDeadheadMiles: new(int32(50))}
	near := addPlacedDriver(s, "Near", chicagoLat+0.1, chicagoLon)
	far := addPlacedDriver(s, "Far", chicagoLat+3, chicagoLon)
	addPlacedDriver(s, "Farther", chicagoLat+5, chicagoLon)

	graph := buildGraph(t, fakeRoutes{}, s, pickupAt(chicagoLat, chicagoLon, 200))

	require.Len(t, graph.Arcs[0], 1, "drivers past the cap get no arc")
	assert.Equal(t, near.WorkerID, graph.Arcs[0][0].Candidate.WorkerID)

	rejected := graph.Rejected[0]
	require.NotNil(t, rejected)
	assert.Equal(t, far.WorkerID, rejected.WorkerID, "the cheapest rejected driver is kept")
	assert.False(t, rejected.Blocked())
	require.NotNil(t, rejected.DeadheadMiles)
	assert.Greater(t, *rejected.DeadheadMiles, 50.0)
}

func TestBuildGraph_MoveBeyondEveryDriverHasNoArcsButARejection(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.Control = &dispatchcontrol.DispatchControl{AutoAssignMaxDeadheadMiles: new(int32(50))}
	far := addPlacedDriver(s, "Far", denverLat, denverLon)

	graph := buildGraph(t, fakeRoutes{}, s, pickupAt(chicagoLat, chicagoLon, 200))

	assert.Empty(t, graph.Arcs[0])
	require.NotNil(t, graph.Rejected[0])
	assert.Equal(t, far.WorkerID, graph.Rejected[0].WorkerID)
	assert.True(t, graph.Rejected[0].LocationKnown())
}

func TestBuildGraph_BlockedDriversNeverBecomeArcs(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	blocked := addPlacedDriver(s, "Blocked", chicagoLat, chicagoLon)
	s.WorkersByID[blocked.WorkerID].AvailableForDispatch = false

	graph := buildGraph(t, fakeRoutes{}, s, pickupAt(chicagoLat, chicagoLon, 200))

	assert.Empty(t, graph.Arcs[0])
	require.NotNil(t, graph.Rejected[0])
	assert.True(t, graph.Rejected[0].Blocked())
}

func TestBuildGraph_MoveWithoutCoordinatesRejectsForUnknownLocation(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)
	move.OriginLatitude = nil
	move.OriginLongitude = nil

	graph := buildGraph(t, fakeRoutes{}, s, move)

	assert.Empty(t, graph.Arcs[0], "no empty miles can be judged without a pickup position")
	require.NotNil(t, graph.Rejected[0])
	assert.False(t, graph.Rejected[0].LocationKnown())
}

func TestBuildGraph_NoDriverWithAPositionRejectsForUnknownLocation(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addDriver(s, "Ada")

	graph := buildGraph(t, fakeRoutes{}, s, pickupAt(chicagoLat, chicagoLon, 200))

	assert.Empty(t, graph.Arcs[0])
	require.NotNil(t, graph.Rejected[0], "the move says why instead of reading as no candidate")
	assert.False(t, graph.Rejected[0].LocationKnown())
}

func TestBuildGraph_SoftCostsAreMeasuredFromTheCheapestCandidate(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	reliable := addPlacedDriver(s, "Reliable", chicagoLat+0.1, chicagoLon)
	unknown := addPlacedDriver(s, "Unknown", chicagoLat+0.1, chicagoLon)
	s.AcceptanceByWorker[reliable.WorkerID] = &dispatchboardports.WorkerAcceptanceStats{
		WorkerID: reliable.WorkerID,
		Accepted: 10,
	}

	graph := buildGraph(t, fakeRoutes{}, s, pickupAt(chicagoLat, chicagoLon, 200))

	arcs := graph.Arcs[0]
	require.Len(t, arcs, 2)
	assert.Equal(t, reliable.WorkerID, arcs[0].Candidate.WorkerID)
	assert.Equal(t, unknown.WorkerID, arcs[1].Candidate.WorkerID)

	assert.Equal(t, arcs[0].Candidate.HardCents, arcs[0].CostCents,
		"the candidate with the least soft cost carries none in the solve")
	assert.Equal(t, arcs[1].Candidate.HardCents+1_500, arcs[1].CostCents,
		"an unknown acceptance record costs half the 3,000-cent preference more")

	floor := arcs[0].Candidate.SoftCents
	for _, arc := range arcs {
		assert.Equal(t, arc.Candidate.HardCents+arc.Candidate.SoftCents-floor, arc.CostCents)
		assert.Equal(t, arc.Candidate.Cost.TotalCents,
			arc.Candidate.HardCents+arc.Candidate.SoftCents,
			"the candidate keeps its full cost for display")
	}
}

func TestBuildGraph_PricesCoveragePerMoveFromRevenueAndUrgency(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)

	planned := pickupAt(chicagoLat, chicagoLon, 200)
	planned.Revenue = new(2_000.0)
	planned.OriginWindowStart = testNow + 5*86400

	dueNow := pickupAt(chicagoLat, chicagoLon, 200)
	dueNow.OriginWindowStart = testNow + 3600

	late := pickupAt(chicagoLat, chicagoLon, 200)
	late.OriginWindowStart = testNow - 2*3600

	graph := buildGraph(t, fakeRoutes{}, s, planned, dueNow, late)

	require.Len(t, graph.Coverage, 3)
	assert.Equal(t, int64(60_000), graph.Coverage[0], "30% of $2,000 beats the $400 floor")
	assert.Equal(t, int64(40_000+75_000), graph.Coverage[1], "the floor plus the due-now premium")
	assert.Equal(t, int64(40_000+150_000), graph.Coverage[2], "the floor plus the late premium")
}

func TestBuildGraph_RoutedMilesReplaceTheEstimateForTopCandidates(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addPlacedDriver(s, "Ada", chicagoLat+0.5, chicagoLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)

	graph := buildGraph(t, fakeRoutes{miles: 42, seconds: 3_000, routed: true}, s, move)

	require.Len(t, graph.Arcs[0], 1)
	candidate := graph.Arcs[0][0].Candidate
	require.NotNil(t, candidate.DeadheadMiles)
	assert.InDelta(t, 42.0, *candidate.DeadheadMiles, 0.0001)
	assert.Equal(t, (3_000+geoutils.PlanningDriveSeconds(200))*1000, candidate.EstimatedDriveMs)
}

func TestBuildGraph_UnroutedLegsKeepTheStraightLineEstimate(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addPlacedDriver(s, "Ada", chicagoLat+0.5, chicagoLon)

	graph := buildGraph(
		t,
		fakeRoutes{miles: 42, routed: false},
		s,
		pickupAt(chicagoLat, chicagoLon, 200),
	)

	require.Len(t, graph.Arcs[0], 1)
	require.NotNil(t, graph.Arcs[0][0].Candidate.DeadheadMiles)
	assert.InDelta(t,
		geoutils.PlanningRoadMiles(chicagoLat+0.5, chicagoLon, chicagoLat, chicagoLon),
		*graph.Arcs[0][0].Candidate.DeadheadMiles,
		0.0001,
	)
}
