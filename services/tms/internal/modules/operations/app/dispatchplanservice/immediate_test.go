package dispatchplanservice

import (
	"context"
	"strconv"
	"testing"

	"github.com/emoss08/trenova/shared/dispatchplanner"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const softFloorCents = int64(7)

type arcSpec struct {
	driver int
	cost   int64
}

func testGraph(drivers int, coverage []int64, rows ...[]arcSpec) *dispatchcandidateservice.Graph {
	graph := &dispatchcandidateservice.Graph{
		Moves:    make([]*dispatchboardports.BoardMove, len(rows)),
		Drivers:  make([]*dispatchboardports.BoardDriver, drivers),
		Arcs:     make([][]dispatchcandidateservice.Arc, len(rows)),
		Coverage: coverage,
		Rejected: make([]*dispatchplanports.CandidateScore, len(rows)),

		PreferenceFloor: make([]int64, len(rows)),
	}
	for i := range graph.Drivers {
		graph.Drivers[i] = &dispatchboardports.BoardDriver{
			WorkerID:  pulid.MustNew("wrk_"),
			FirstName: driverName(i),
		}
	}
	for i, arcs := range rows {
		graph.Moves[i] = &dispatchboardports.BoardMove{
			MoveID:            pulid.MustNew("smv_"),
			MoveVersion:       int64(i + 3),
			ProNumber:         "PRO-" + strconv.Itoa(i),
			OriginCity:        "Chicago",
			OriginState:       "IL",
			DestinationCity:   "Denver",
			DestinationState:  "CO",
			OriginWindowStart: int64(1_000 + i),
		}
		if len(arcs) > 0 {
			graph.PreferenceFloor[i] = softFloorCents
		}
		for _, spec := range arcs {
			graph.Arcs[i] = append(graph.Arcs[i], dispatchcandidateservice.Arc{
				Driver:    spec.driver,
				CostCents: spec.cost,
				Candidate: candidateFor(graph, i, spec),
			})
		}
	}

	return graph
}

func driverName(i int) string {
	return "Driver " + strconv.Itoa(i)
}

func candidateFor(
	graph *dispatchcandidateservice.Graph,
	move int,
	spec arcSpec,
) *dispatchplanports.CandidateScore {
	return &dispatchplanports.CandidateScore{
		WorkerID:         graph.Drivers[spec.driver].WorkerID,
		WorkerName:       driverName(spec.driver),
		MoveID:           graph.Moves[move].MoveID,
		Cost:             dispatcheconomics.Cost{TotalCents: spec.cost + softFloorCents},
		ProjectedArrival: int64(5_000 + move),
		ProjectedFreeAt:  int64(9_000 + move),
		DeadheadMiles:    new(10.0),
		OriginSource:     dispatchplanports.OriginPosition,
		HardCents:        spec.cost,
		SoftCents:        softFloorCents,
	}
}

func emptyPlan() *dispatchplanports.Plan {
	return &dispatchplanports.Plan{
		Pairings:  []*dispatchplanports.Pairing{},
		Uncovered: []*dispatchplanports.Uncovered{},
	}
}

func runImmediate(graph *dispatchcandidateservice.Graph) *dispatchplanports.Plan {
	plan := emptyPlan()
	planImmediate(context.Background(), plan, graph, stability{})

	return plan
}

func TestPlanImmediate_ChoosesTheLeastTotalCostAcrossTheBoard(t *testing.T) {
	t.Parallel()

	graph := testGraph(2, []int64{5_000, 5_000},
		[]arcSpec{{driver: 0, cost: 100}, {driver: 1, cost: 300}},
		[]arcSpec{{driver: 0, cost: 150}, {driver: 1, cost: 1_000}},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 2)
	assert.Empty(t, plan.Uncovered)
	assert.Equal(t, graph.Drivers[1].WorkerID, plan.Pairings[0].Candidate.WorkerID,
		"the first move gives up its cheapest driver so the second avoids a $10 one")
	assert.Equal(t, graph.Drivers[0].WorkerID, plan.Pairings[1].Candidate.WorkerID)
	assert.Equal(t, int64(300+150), plan.CostCents,
		"the plan's cost is the candidates' full cost, not the solve's normalized arcs")
	assert.Zero(t, plan.UncoveredCents)
}

func TestPlanImmediate_DriverValueIsWhatLosingTheDriverCostsThePlan(t *testing.T) {
	t.Parallel()

	graph := testGraph(2, []int64{5_000, 5_000},
		[]arcSpec{{driver: 0, cost: 100}, {driver: 1, cost: 300}},
		[]arcSpec{{driver: 0, cost: 150}, {driver: 1, cost: 1_000}},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 2)
	assert.Equal(t, int64(4_650), plan.Pairings[0].DriverValueCents,
		"without driver 1 the best is driver 0 on the first move and the second uncovered: "+
			"5,100 against 450")
	assert.Equal(t, int64(4_850), plan.Pairings[1].DriverValueCents,
		"without driver 0 the best is driver 1 on the first move and the second uncovered: "+
			"5,300 against 450")
}

func TestPlanImmediate_AlternativesCarryTheExactOverrideCost(t *testing.T) {
	t.Parallel()

	graph := testGraph(2, []int64{5_000, 5_000},
		[]arcSpec{{driver: 0, cost: 100}, {driver: 1, cost: 300}},
		[]arcSpec{{driver: 0, cost: 150}, {driver: 1, cost: 1_000}},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 2)
	assert.Equal(t, []dispatchplanprice.Alternative{{
		WorkerID:   graph.Drivers[0].WorkerID,
		WorkerName: driverName(0),
		ExtraCents: 650,
		CostCents:  100,
	}}, plan.Pairings[0].Alternatives,
		"forcing driver 0 onto the first move pushes the second onto driver 1: 1,100 against 450")
	assert.Equal(t, []dispatchplanprice.Alternative{{
		WorkerID:   graph.Drivers[1].WorkerID,
		WorkerName: driverName(1),
		ExtraCents: 650,
		CostCents:  1_000,
	}}, plan.Pairings[1].Alternatives)
}

func TestPlanImmediate_AlternativesAreSortedByExtraCostAndCappedAtThree(t *testing.T) {
	t.Parallel()

	graph := testGraph(5, []int64{100_000},
		[]arcSpec{
			{driver: 0, cost: 10},
			{driver: 3, cost: 40},
			{driver: 1, cost: 20},
			{driver: 2, cost: 30},
			{driver: 4, cost: 50},
		},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 1)
	require.Equal(t, graph.Drivers[0].WorkerID, plan.Pairings[0].Candidate.WorkerID)
	alternatives := plan.Pairings[0].Alternatives
	require.Len(t, alternatives, 3)
	assert.Equal(t, []dispatchplanprice.Alternative{
		{
			WorkerID:   graph.Drivers[1].WorkerID,
			WorkerName: driverName(1),
			ExtraCents: 10,
			CostCents:  20,
		},
		{
			WorkerID:   graph.Drivers[2].WorkerID,
			WorkerName: driverName(2),
			ExtraCents: 20,
			CostCents:  30,
		},
		{
			WorkerID:   graph.Drivers[3].WorkerID,
			WorkerName: driverName(3),
			ExtraCents: 30,
			CostCents:  40,
		},
	}, alternatives)
}

func TestPlanImmediate_AlternativesAreTheCheapestOverridesNotTheNearestArcs(t *testing.T) {
	t.Parallel()

	graph := testGraph(5, []int64{100_000, 100_000},
		[]arcSpec{
			{driver: 0, cost: 10},
			{driver: 1, cost: 20},
			{driver: 2, cost: 30},
			{driver: 3, cost: 40},
			{driver: 4, cost: 50},
		},
		[]arcSpec{{driver: 1, cost: 0}},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 2)
	assert.Equal(t, []dispatchplanprice.Alternative{
		{
			WorkerID:   graph.Drivers[2].WorkerID,
			WorkerName: driverName(2),
			ExtraCents: 20,
			CostCents:  30,
		},
		{
			WorkerID:   graph.Drivers[3].WorkerID,
			WorkerName: driverName(3),
			ExtraCents: 30,
			CostCents:  40,
		},
		{
			WorkerID:   graph.Drivers[4].WorkerID,
			WorkerName: driverName(4),
			ExtraCents: 40,
			CostCents:  50,
		},
	}, plan.Pairings[0].Alternatives, "taking driver 1 costs the second move its only driver")
}

func TestPlanImmediate_LeavesAMoveUncoveredWhenThatCostsTheBoardLess(t *testing.T) {
	t.Parallel()

	graph := testGraph(2, []int64{5_000, 200},
		[]arcSpec{{driver: 0, cost: 100}, {driver: 1, cost: 300}},
		[]arcSpec{{driver: 0, cost: 150}, {driver: 1, cost: 1_000}},
	)

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 1)
	require.Len(t, plan.Uncovered, 1)
	pairing := plan.Pairings[0]
	assert.Equal(t, graph.Moves[0].MoveID, pairing.MoveID)
	assert.Equal(t, graph.Drivers[0].WorkerID, pairing.Candidate.WorkerID)
	assert.Equal(t, int64(200), pairing.DriverValueCents,
		"without driver 0 the first move takes driver 1 at 300: 500 against 300")
	assert.Equal(t, []dispatchplanprice.Alternative{{
		WorkerID:   graph.Drivers[1].WorkerID,
		WorkerName: driverName(1),
		ExtraCents: 150,
		CostCents:  300,
	}}, pairing.Alternatives, "driver 1 on the first move lets driver 0 cover the second: 450")

	uncovered := plan.Uncovered[0]
	assert.Equal(t, graph.Moves[1].MoveID, uncovered.MoveID)
	assert.Equal(t, dispatchplanprice.UncoveredOutbid, uncovered.Reason)
	assert.Same(t, graph.Arcs[1][0].Candidate, uncovered.BestCandidate)

	assert.Equal(t, int64(100), plan.CostCents)
	assert.Equal(t, int64(200), plan.UncoveredCents)
}

func TestPlanImmediate_PairingCarriesTheMoveAndCandidate(t *testing.T) {
	t.Parallel()

	graph := testGraph(1, []int64{5_000}, []arcSpec{{driver: 0, cost: 100}})

	plan := runImmediate(graph)

	require.Len(t, plan.Pairings, 1)
	pairing := plan.Pairings[0]
	move := graph.Moves[0]
	assert.Equal(t, move.MoveID, pairing.MoveID)
	assert.Equal(t, move.MoveVersion, pairing.MoveVersion)
	assert.Equal(t, move.ProNumber, pairing.ProNumber)
	assert.Equal(t, "Chicago, IL", pairing.OriginLabel)
	assert.Equal(t, "Denver, CO", pairing.DestinationLabel)
	assert.Equal(t, move.OriginWindowStart, pairing.PickupOpens)
	assert.Same(t, graph.Arcs[0][0].Candidate, pairing.Candidate)
	assert.Equal(t, int64(5_000), pairing.CoveragePriceCents)
	assert.Equal(t, pairing.Candidate.ProjectedArrival, pairing.ProjectedStartAt)
	assert.Equal(t, pairing.Candidate.ProjectedFreeAt, pairing.ProjectedCompleteAt)
	assert.Empty(t, pairing.Alternatives, "a lone driver has no alternative")
}

func TestPlaceLabel_FallsBackFromCityAndStateToName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Chicago, IL", placeLabel("Chicago", "IL", "Dock 4"))
	assert.Equal(t, "Chicago", placeLabel("Chicago", "", "Dock 4"))
	assert.Equal(t, "Dock 4", placeLabel("", "IL", "Dock 4"))
}

func TestPlanImmediate_ClassifiesWhyAMoveIsUncovered(t *testing.T) {
	t.Parallel()

	blocked := &dispatchplanports.CandidateScore{
		WorkerName:    "Blocked",
		DeadheadMiles: new(5.0),
		Findings: []dispatchfinding.Finding{{
			Code:     dispatchfinding.CodeWorkerNotForDispatch,
			Severity: dispatchfinding.SeverityBlock,
		}},
	}
	nowhere := &dispatchplanports.CandidateScore{WorkerName: "Nowhere"}
	distant := &dispatchplanports.CandidateScore{
		WorkerName:    "Distant",
		DeadheadMiles: new(300.0),
	}

	single := func(rejected *dispatchplanports.CandidateScore) *dispatchcandidateservice.Graph {
		graph := testGraph(1, []int64{5_000}, nil)
		graph.Rejected[0] = rejected
		return graph
	}

	tests := []struct {
		name   string
		graph  *dispatchcandidateservice.Graph
		reason dispatchplanprice.UncoveredReason
		best   func(*dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore
	}{
		{
			name:   "nobody was even considered",
			graph:  single(nil),
			reason: dispatchplanprice.UncoveredNoCandidate,
			best: func(*dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return nil
			},
		},
		{
			name:   "the best driver is disqualified",
			graph:  single(blocked),
			reason: dispatchplanprice.UncoveredDisqualified,
			best: func(*dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return blocked
			},
		},
		{
			name:   "the best driver's position is unknown",
			graph:  single(nowhere),
			reason: dispatchplanprice.UncoveredLocationUnknown,
			best: func(*dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return nowhere
			},
		},
		{
			name:   "the best driver is past the empty-mile limit",
			graph:  single(distant),
			reason: dispatchplanprice.UncoveredBeyondDeadhead,
			best: func(*dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return distant
			},
		},
		{
			name: "every driver costs more than leaving it",
			graph: testGraph(2, []int64{500},
				[]arcSpec{{driver: 0, cost: 900}, {driver: 1, cost: 700}},
			),
			reason: dispatchplanprice.UncoveredNotWorthIt,
			best: func(g *dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return g.Arcs[0][1].Candidate
			},
		},
		{
			name: "its only driver is worth more on another move",
			graph: testGraph(1, []int64{10_000, 5_000},
				[]arcSpec{{driver: 0, cost: 100}},
				[]arcSpec{{driver: 0, cost: 100}},
			),
			reason: dispatchplanprice.UncoveredOutbid,
			best: func(g *dispatchcandidateservice.Graph) *dispatchplanports.CandidateScore {
				return g.Arcs[1][0].Candidate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := runImmediate(tt.graph)

			target := tt.graph.Moves[len(tt.graph.Moves)-1]
			require.Len(t, plan.Uncovered, 1)
			uncovered := plan.Uncovered[0]
			assert.Equal(t, target.MoveID, uncovered.MoveID)
			assert.Equal(t, target.ProNumber, uncovered.ProNumber)
			assert.Equal(t, target.OriginWindowStart, uncovered.StartsAt)
			assert.Equal(t, tt.reason, uncovered.Reason)
			if best := tt.best(tt.graph); best == nil {
				assert.Nil(t, uncovered.BestCandidate)
			} else {
				assert.Same(t, best, uncovered.BestCandidate)
			}
			assert.Equal(t, tt.graph.Coverage[len(tt.graph.Moves)-1], uncovered.CoveragePriceCents)
			assert.Equal(t, uncovered.CoveragePriceCents, plan.UncoveredCents)
		})
	}
}

func TestPlanImmediate_KeepsTheLastPlansDriverUnlessChangingSavesMoreThanTheReplanCost(
	t *testing.T,
) {
	t.Parallel()

	newGraph := func() *dispatchcandidateservice.Graph {
		return testGraph(2, []int64{100_000},
			[]arcSpec{{driver: 0, cost: 100}, {driver: 1, cost: 130}},
		)
	}

	kept := newGraph()
	keptPlan := emptyPlan()
	planImmediate(context.Background(), keptPlan, kept, stability{incumbents: []int{1}, cents: 50})
	require.Len(t, keptPlan.Pairings, 1)
	assert.Equal(t, kept.Drivers[1].WorkerID, keptPlan.Pairings[0].Candidate.WorkerID,
		"saving 30 is not worth a change that costs 50")

	changed := newGraph()
	changedPlan := emptyPlan()
	planImmediate(context.Background(), changedPlan, changed, stability{incumbents: []int{1}, cents: 20})
	require.Len(t, changedPlan.Pairings, 1)
	assert.Equal(t, changed.Drivers[0].WorkerID, changedPlan.Pairings[0].Candidate.WorkerID,
		"saving 30 is worth a change that costs 20")
}

func TestStabilityOf_MapsTheLastPlansWorkersOntoGraphDrivers(t *testing.T) {
	t.Parallel()

	graph := testGraph(2, []int64{1_000, 1_000},
		[]arcSpec{{driver: 0, cost: 10}},
		[]arcSpec{{driver: 1, cost: 10}},
	)
	standing := map[pulid.ID]pulid.ID{
		graph.Moves[0].MoveID: graph.Drivers[1].WorkerID,
		graph.Moves[1].MoveID: pulid.MustNew("wrk_"),
	}

	got := stabilityOf(graph, standing, -5)

	assert.Equal(t, []int{1, dispatchplanner.Unassigned}, got.incumbents,
		"a worker no longer on the board has no standing")
	assert.Equal(t, int64(0), got.cents)
}
