package dispatchautoassignservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const horizonNow = int64(1_700_000_000)

func horizonMove(pro string) *repositories.BoardMove {
	return &repositories.BoardMove{MoveID: pulid.MustNew("mov_"), ProNumber: pro}
}

func horizonScore(
	workerID pulid.ID,
	name string,
	value int,
	deadhead float64,
) *dispatchcandidateservice.CandidateScore {
	return &dispatchcandidateservice.CandidateScore{
		WorkerID:           workerID,
		WorkerName:         name,
		TractorID:          pulid.MustNew("trac_"),
		Score:              value,
		DeadheadMiles:      &deadhead,
		ProjectedAvailable: horizonNow,
		EstimatedDriveMs:   4 * 3600 * 1000,
	}
}

type horizonFixture struct {
	oracle *horizonOracle
	moves  []*repositories.BoardMove
}

// newHorizonFixture wires an oracle with pre-computed scores so plan building can be
// exercised without standing up the scorer.
func newHorizonFixture(
	moves []*repositories.BoardMove,
	scores [][]*dispatchcandidateservice.CandidateScore,
) horizonFixture {
	drivers := make([]*repositories.BoardDriver, 0)
	if len(scores) > 0 {
		drivers = make([]*repositories.BoardDriver, len(scores[0]))
		for i := range drivers {
			drivers[i] = &repositories.BoardDriver{WorkerID: pulid.MustNew("wrk_")}
		}
	}

	return horizonFixture{
		oracle: &horizonOracle{moves: moves, drivers: drivers, scores: scores},
		moves:  moves,
	}
}

func horizonControls() (*dispatchcontrol.DispatchControl, *tenant.AgentControl) {
	return &dispatchcontrol.DispatchControl{
			PlanningMode:             dispatchcontrol.PlanningModeHorizon,
			HorizonMaxMovesPerDriver: 3,
		}, &tenant.AgentControl{
			DispatchAgentEnabled: true,
			DispatchAutonomyTier: string(agent.TierPropose),
		}
}

func TestBuildHorizonPlan_GroupsChainedMovesIntoOneTour(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	moves := []*repositories.BoardMove{horizonMove("P1"), horizonMove("P2")}
	fixture := newHorizonFixture(moves, [][]*dispatchcandidateservice.CandidateScore{
		{horizonScore(worker, "Dana", 90, 10)},
		{horizonScore(worker, "Dana", 80, 25)},
	})
	control, agentControl := horizonControls()

	plan := buildHorizonPlan(&buildHorizonPlanParams{
		Moves: moves,
		Result: dispatchplanner.Result{
			Assignments: []dispatchplanner.Assignment{
				{Task: 0, Resource: 0, Sequence: 0},
				{Task: 1, Resource: 0, Sequence: 1},
			},
			Unassigned: []int{},
		},
		Oracle:       fixture.oracle,
		Control:      control,
		AgentControl: agentControl,
		Now:          horizonNow,
	})

	require.Len(t, plan.Assignments, 2)
	require.Len(t, plan.Tours, 1, "both moves belong to the same driver's tour")

	tour := plan.Tours[0]
	assert.Equal(t, worker, tour.WorkerID)
	assert.Equal(t, "Dana", tour.WorkerName)
	assert.Equal(t, []pulid.ID{moves[0].MoveID, moves[1].MoveID}, tour.MoveIDs)
	assert.Equal(t, 170, tour.TotalScore)
	assert.InDelta(t, 35.0, tour.TotalDeadheadMiles, 0.001)

	assert.Equal(t, tour.TourID, plan.Assignments[0].TourID)
	assert.Equal(t, tour.TourID, plan.Assignments[1].TourID)
	assert.Equal(t, 0, plan.Assignments[0].SequenceIndex)
	assert.Equal(t, 1, plan.Assignments[1].SequenceIndex)
	assert.Equal(t, 170, plan.TotalScore)
}

func TestBuildHorizonPlan_SeparateDriversGetSeparateTours(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("wrk_")
	second := pulid.MustNew("wrk_")
	moves := []*repositories.BoardMove{horizonMove("P1"), horizonMove("P2")}
	fixture := newHorizonFixture(moves, [][]*dispatchcandidateservice.CandidateScore{
		{horizonScore(first, "Dana", 90, 10), horizonScore(second, "Sam", 70, 40)},
		{horizonScore(first, "Dana", 60, 50), horizonScore(second, "Sam", 85, 15)},
	})
	control, agentControl := horizonControls()

	plan := buildHorizonPlan(&buildHorizonPlanParams{
		Moves: moves,
		Result: dispatchplanner.Result{
			Assignments: []dispatchplanner.Assignment{
				{Task: 0, Resource: 0, Sequence: 0},
				{Task: 1, Resource: 1, Sequence: 0},
			},
			Unassigned: []int{},
		},
		Oracle:       fixture.oracle,
		Control:      control,
		AgentControl: agentControl,
		Now:          horizonNow,
	})

	require.Len(t, plan.Tours, 2)
	assert.Equal(t, first, plan.Tours[0].WorkerID)
	assert.Equal(t, second, plan.Tours[1].WorkerID)
	assert.NotEqual(t, plan.Tours[0].TourID, plan.Tours[1].TourID)
}

func TestBuildHorizonPlan_UnassignedMovesBecomeUncovered(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	moves := []*repositories.BoardMove{horizonMove("P1"), horizonMove("P2")}
	fixture := newHorizonFixture(moves, [][]*dispatchcandidateservice.CandidateScore{
		{horizonScore(worker, "Dana", 90, 10)},
		{nil},
	})
	control, agentControl := horizonControls()

	plan := buildHorizonPlan(&buildHorizonPlanParams{
		Moves: moves,
		Result: dispatchplanner.Result{
			Assignments: []dispatchplanner.Assignment{{Task: 0, Resource: 0, Sequence: 0}},
			Unassigned:  []int{1},
		},
		Oracle:       fixture.oracle,
		Control:      control,
		AgentControl: agentControl,
		Now:          horizonNow,
	})

	require.Len(t, plan.Assignments, 1)
	require.Len(t, plan.Uncovered, 1)
	assert.Equal(t, moves[1].MoveID, plan.Uncovered[0].MoveID)
	assert.Equal(t, "P2", plan.Uncovered[0].ProNumber)
}

func TestBuildHorizonPlan_CarriesProjectionsOntoAssignments(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	moves := []*repositories.BoardMove{horizonMove("P1")}
	scored := horizonScore(worker, "Dana", 90, 10)
	scored.HOSProjectedDriveMs = 5 * 3600 * 1000
	fixture := newHorizonFixture(moves, [][]*dispatchcandidateservice.CandidateScore{{scored}})
	control, agentControl := horizonControls()

	plan := buildHorizonPlan(&buildHorizonPlanParams{
		Moves: moves,
		Result: dispatchplanner.Result{
			Assignments: []dispatchplanner.Assignment{{Task: 0, Resource: 0, Sequence: 0}},
			Unassigned:  []int{},
		},
		Oracle:       fixture.oracle,
		Control:      control,
		AgentControl: agentControl,
		Now:          horizonNow,
	})

	require.Len(t, plan.Assignments, 1)
	assigned := plan.Assignments[0]
	assert.Equal(t, horizonNow, assigned.ProjectedStartAt)
	assert.Equal(t, horizonNow+4*3600, assigned.ProjectedCompleteAt)
	assert.Equal(t, int64(5*3600*1000), assigned.ProjectedDriveRemains)
	assert.Equal(t, dispatchcontrol.PlanningModeHorizon.String(), plan.PlanningMode)
}

func TestSolve_RoutesOnPlanningMode(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	_, agentControl := horizonControls()

	t.Run("no drivers short-circuits in either mode", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []dispatchcontrol.PlanningMode{
			dispatchcontrol.PlanningModeImmediate,
			dispatchcontrol.PlanningModeHorizon,
		} {
			plan := svc.solve(&solveParams{
				Moves:        []*repositories.BoardMove{horizonMove("P1")},
				Snapshot:     &dispatchcandidateservice.FleetSnapshot{},
				Control:      &dispatchcontrol.DispatchControl{PlanningMode: mode},
				AgentControl: agentControl,
				Now:          horizonNow,
			})

			assert.Empty(t, plan.Assignments)
			require.Len(t, plan.Uncovered, 1)
			assert.Equal(t, mode.String(), plan.PlanningMode)
		}
	})
}

// A control that predates horizon planning, or one built in memory, must keep the
// single-period behaviour it was configured against.
func TestResolvedPlanningMode_DefaultsToImmediate(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		dispatchcontrol.PlanningModeImmediate,
		(&dispatchcontrol.DispatchControl{}).ResolvedPlanningMode(),
	)
	assert.Equal(
		t,
		dispatchcontrol.PlanningModeImmediate,
		(*dispatchcontrol.DispatchControl)(nil).ResolvedPlanningMode(),
	)
	assert.Equal(
		t,
		dispatchcontrol.PlanningModeHorizon,
		(&dispatchcontrol.DispatchControl{
			PlanningMode: dispatchcontrol.PlanningModeHorizon,
		}).ResolvedPlanningMode(),
	)
}

// A search pass rewinds the oracle many times. If a rewind ever left a planned move
// behind, later scoring would depart from a position the driver was never given.
func TestHorizonOracle_RebuildRestoresTheRealWorkload(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	existing := &repositories.WorkerCommitment{
		WorkerID:  worker,
		MoveID:    pulid.MustNew("mov_"),
		WindowEnd: horizonNow + 3600,
	}

	snapshot := &dispatchcandidateservice.FleetSnapshot{
		Now:     horizonNow,
		Drivers: []*repositories.BoardDriver{{WorkerID: worker}},
		CommitmentsByWorker: map[pulid.ID][]*repositories.WorkerCommitment{
			worker: {existing},
		},
	}

	moves := []*repositories.BoardMove{horizonMove("P1")}
	control, _ := horizonControls()

	oracle := newHorizonOracle(&solveParams{
		Moves:    moves,
		Snapshot: snapshot,
		Control:  control,
		Now:      horizonNow,
	}, &dispatchcandidateservice.Service{})

	dispatchcandidateservice.CommitPlannedMove(&dispatchcandidateservice.CommitPlannedMoveRequest{
		Snapshot: snapshot,
		Move:     moves[0],
		Score: &dispatchcandidateservice.CandidateScore{
			WorkerID:           worker,
			ProjectedAvailable: horizonNow,
			EstimatedDriveMs:   3600 * 1000,
		},
	})
	require.Len(t, snapshot.CommitmentsByWorker[worker], 2, "the planned move was folded in")

	oracle.Rebuild(nil)

	require.Len(t, snapshot.CommitmentsByWorker[worker], 1)
	assert.Equal(t, existing.MoveID, snapshot.CommitmentsByWorker[worker][0].MoveID)
}

func TestHorizonOracle_RebuildDoesNotMutateTheCapturedBaseline(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	snapshot := &dispatchcandidateservice.FleetSnapshot{
		Now:     horizonNow,
		Drivers: []*repositories.BoardDriver{{WorkerID: worker}},
		CommitmentsByWorker: map[pulid.ID][]*repositories.WorkerCommitment{
			worker: {{WorkerID: worker, WindowEnd: horizonNow + 3600}},
		},
	}

	moves := []*repositories.BoardMove{horizonMove("P1")}
	control, _ := horizonControls()
	oracle := newHorizonOracle(&solveParams{
		Moves:    moves,
		Snapshot: snapshot,
		Control:  control,
		Now:      horizonNow,
	}, &dispatchcandidateservice.Service{})

	for range 5 {
		dispatchcandidateservice.CommitPlannedMove(
			&dispatchcandidateservice.CommitPlannedMoveRequest{
				Snapshot: snapshot,
				Move:     moves[0],
				Score: &dispatchcandidateservice.CandidateScore{
					WorkerID:           worker,
					ProjectedAvailable: horizonNow,
				},
			},
		)
		oracle.Rebuild(nil)

		assert.Len(t, snapshot.CommitmentsByWorker[worker], 1,
			"repeated rewinds must not accumulate")
	}
}
