package dispatchplanner_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routeOracle is a one-dimensional stand-in for the real thing: a resource sits at a
// position, a task costs the distance to reach it, and taking a task moves the
// resource there. That is the same shape as a driver whose next deadhead is measured
// from where their last move ended.
type routeOracle struct {
	taskAt    []float64
	startAt   []float64
	currentAt []float64
}

func newRouteOracle(taskAt, startAt []float64) *routeOracle {
	current := make([]float64, len(startAt))
	copy(current, startAt)

	return &routeOracle{taskAt: taskAt, startAt: startAt, currentAt: current}
}

func (o *routeOracle) Cost(task, resource int) float64 {
	return math.Abs(o.taskAt[task] - o.currentAt[resource])
}

func (o *routeOracle) Commit(task, resource int) {
	o.currentAt[resource] = o.taskAt[task]
}

func (o *routeOracle) Rebuild(
	assignments []dispatchplanner.Assignment,
) []dispatchplanner.Assignment {
	copy(o.currentAt, o.startAt)

	replayed := make([]dispatchplanner.Assignment, len(assignments))
	for i, assignment := range assignments {
		assignment.Cost = o.Cost(assignment.Task, assignment.Resource)
		o.Commit(assignment.Task, assignment.Resource)
		replayed[i] = assignment
	}

	return replayed
}

// greedyTrap is built so the myopic choice is wrong: taking the globally cheapest
// pairing first fills one resource and strands the far task on the other.
//
// Greedy reaches 130. Splitting the tasks by neighbourhood reaches 90.
func greedyTrap() (*routeOracle, dispatchplanner.SolveParams) {
	oracle := newRouteOracle(
		[]float64{10, 40, 60, 100},
		[]float64{0, 50},
	)

	return oracle, dispatchplanner.SolveParams{
		Tasks:     4,
		Resources: 2,
		Oracle:    oracle,
		Options:   dispatchplanner.Options{MaxPerResource: 2},
	}
}

func TestImprove_EscapesAGreedyLocalOptimum(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()

	initial := dispatchplanner.Solve(params)
	require.Len(t, initial.Assignments, 4)
	assert.InDelta(t, 130.0, initial.TotalCost, 0.001,
		"the greedy pass takes the cheapest pairing first and strands the far task")

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 200,
		Seed:       7,
	})

	assert.Len(t, improved.Assignments, 4, "every task stays covered")
	assert.Less(t, improved.TotalCost, initial.TotalCost)
	assert.InDelta(t, 90.0, improved.TotalCost, 0.001,
		"splitting the tasks by neighbourhood is the arrangement worth finding")
}

func TestImprove_IsNeverWorseThanTheInitialSolution(t *testing.T) {
	t.Parallel()

	for seed := range int64(20) {
		oracle, params := greedyTrap()
		initial := dispatchplanner.Solve(params)

		improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
			Resources:  params.Resources,
			Oracle:     oracle,
			Options:    params.Options,
			Initial:    initial,
			Iterations: 40,
			Seed:       seed,
		})

		assert.GreaterOrEqual(t, len(improved.Assignments), len(initial.Assignments),
			"seed %d lost coverage", seed)
		assert.LessOrEqual(t, improved.TotalCost, initial.TotalCost,
			"seed %d got more expensive", seed)
	}
}

func TestImprove_IsDeterministicForASeed(t *testing.T) {
	t.Parallel()

	run := func() dispatchplanner.Result {
		oracle, params := greedyTrap()
		initial := dispatchplanner.Solve(params)

		return dispatchplanner.Improve(dispatchplanner.ImproveParams{
			Resources:  params.Resources,
			Oracle:     oracle,
			Options:    params.Options,
			Initial:    initial,
			Iterations: 60,
			Seed:       99,
		})
	}

	first := run()
	for range 10 {
		assert.Equal(t, first, run())
	}
}

func TestImprove_ReportedCostMatchesTheAssignments(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()
	initial := dispatchplanner.Solve(params)

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 50,
		Seed:       3,
	})

	summed := 0.0
	for _, assignment := range improved.Assignments {
		summed += assignment.Cost
	}

	assert.InDelta(t, summed, improved.TotalCost, 0.001)
}

// The oracle has to end up holding the solution that was returned, because the caller
// reads per-assignment detail back out of it after planning.
func TestImprove_LeavesTheOracleOnTheReturnedSolution(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()
	initial := dispatchplanner.Solve(params)

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 50,
		Seed:       11,
	})

	settled := make([]float64, len(oracle.currentAt))
	copy(settled, oracle.currentAt)

	replayed := oracle.Rebuild(improved.Assignments)

	assert.Equal(t, settled, oracle.currentAt)
	for i, assignment := range replayed {
		assert.InDelta(t, improved.Assignments[i].Cost, assignment.Cost, 0.001)
	}
}

func TestImprove_RenumbersSequencesPerResource(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()
	initial := dispatchplanner.Solve(params)

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 80,
		Seed:       5,
	})

	next := make(map[int]int, params.Resources)
	for _, assignment := range improved.Assignments {
		assert.Equal(t, next[assignment.Resource], assignment.Sequence,
			"sequences must run 0..n-1 within each resource")
		next[assignment.Resource]++
	}
}

func TestImprove_ReturnsTheInitialSolutionWhenThereIsNothingToDo(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()
	initial := dispatchplanner.Solve(params)

	tests := []struct {
		name   string
		params dispatchplanner.ImproveParams
	}{
		{
			name: "no iterations",
			params: dispatchplanner.ImproveParams{
				Resources: params.Resources, Oracle: oracle, Initial: initial, Iterations: 0,
			},
		},
		{
			name: "negative iterations",
			params: dispatchplanner.ImproveParams{
				Resources: params.Resources, Oracle: oracle, Initial: initial, Iterations: -5,
			},
		},
		{
			name: "nil oracle",
			params: dispatchplanner.ImproveParams{
				Resources: params.Resources, Initial: initial, Iterations: 50,
			},
		},
		{
			name: "nothing assigned to tear out",
			params: dispatchplanner.ImproveParams{
				Resources:  params.Resources,
				Oracle:     oracle,
				Initial:    dispatchplanner.Result{Unassigned: []int{0, 1}},
				Iterations: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := dispatchplanner.Improve(tt.params)

			assert.Len(t, result.Assignments, len(tt.params.Initial.Assignments))
			assert.InDelta(t, tt.params.Initial.TotalCost, result.TotalCost, 0.001)
		})
	}
}

func TestImprove_RespectsMaxPerResource(t *testing.T) {
	t.Parallel()

	oracle, params := greedyTrap()
	initial := dispatchplanner.Solve(params)

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 150,
		Seed:       23,
	})

	counts := make(map[int]int, params.Resources)
	for _, assignment := range improved.Assignments {
		counts[assignment.Resource]++
	}

	for resource, count := range counts {
		assert.LessOrEqual(t, count, params.Options.MaxPerResource,
			"resource %d exceeded its cap", resource)
	}
}

// Coverage must outrank cost. A search that could shed an expensive move to lower its
// total would quietly stop covering loads.
func TestImprove_KeepsCoverageWhenDroppingWouldBeCheaper(t *testing.T) {
	t.Parallel()

	// One resource, two tasks: the second is far more expensive than the first, so
	// abandoning it would more than halve the total.
	oracle := newRouteOracle([]float64{1, 500}, []float64{0})
	params := dispatchplanner.SolveParams{
		Tasks:     2,
		Resources: 1,
		Oracle:    oracle,
		Options:   dispatchplanner.Options{MaxPerResource: 2},
	}

	initial := dispatchplanner.Solve(params)
	require.Len(t, initial.Assignments, 2)

	improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  params.Resources,
		Oracle:     oracle,
		Options:    params.Options,
		Initial:    initial,
		Iterations: 100,
		Seed:       42,
	})

	assert.Len(t, improved.Assignments, 2)
	assert.Empty(t, improved.Unassigned)
}
