package dispatchplanner_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A full board as the service caps it: maxPlanMoves is 400, against a large fleet.
const (
	benchTasks     = 400
	benchResources = 250
)

// countingOracle reports how many times the planner asks for a cost. Every one of
// those is a scoreDriver call in the real system, so the count is what determines
// whether a planning run fits inside a request.
type countingOracle struct {
	*routeOracle
	costCalls int
}

func (o *countingOracle) Cost(task, resource int) float64 {
	o.costCalls++
	return o.routeOracle.Cost(task, resource)
}

func (o *countingOracle) Rebuild(
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

func newBoard(tasks, resources int) *countingOracle {
	taskAt := make([]float64, tasks)
	for i := range taskAt {
		taskAt[i] = float64((i * 37) % 1000)
	}

	startAt := make([]float64, resources)
	for i := range startAt {
		startAt[i] = float64((i * 91) % 1000)
	}

	return &countingOracle{routeOracle: newRouteOracle(taskAt, startAt)}
}

func benchOptions() dispatchplanner.Options {
	return dispatchplanner.Options{MaxPerResource: 3}
}

func BenchmarkSolve_FullBoard(b *testing.B) {
	for b.Loop() {
		oracle := newBoard(benchTasks, benchResources)
		dispatchplanner.Solve(dispatchplanner.SolveParams{
			Tasks:     benchTasks,
			Resources: benchResources,
			Oracle:    oracle,
			Options:   benchOptions(),
		})
	}
}

func BenchmarkImprove_FullBoard(b *testing.B) {
	for _, iterations := range []int{5, 25, 100} {
		b.Run(fmt.Sprintf("iterations=%d", iterations), func(b *testing.B) {
			for b.Loop() {
				oracle := newBoard(benchTasks, benchResources)
				initial := dispatchplanner.Solve(dispatchplanner.SolveParams{
					Tasks:     benchTasks,
					Resources: benchResources,
					Oracle:    oracle,
					Options:   benchOptions(),
				})

				dispatchplanner.Improve(dispatchplanner.ImproveParams{
					Resources:  benchResources,
					Oracle:     oracle,
					Options:    benchOptions(),
					Initial:    initial,
					Iterations: iterations,
					Seed:       1,
				})
			}
		})
	}
}

// The scorer is the expensive part of real planning, so the call count is the budget
// that matters. This pins it: a change that makes search cost materially more per
// iteration should fail here rather than in production.
func TestPlanningStaysWithinItsScoringBudget(t *testing.T) {
	t.Parallel()

	oracle := newBoard(benchTasks, benchResources)

	initial := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     benchTasks,
		Resources: benchResources,
		Oracle:    oracle,
		Options:   benchOptions(),
	})
	solveCalls := oracle.costCalls

	require.NotEmpty(t, initial.Assignments)
	assert.LessOrEqual(t, solveCalls, 250_000,
		"the initial pass costs the full matrix plus one column per commit")

	oracle.costCalls = 0
	dispatchplanner.Improve(dispatchplanner.ImproveParams{
		Resources:  benchResources,
		Oracle:     oracle,
		Options:    benchOptions(),
		Initial:    initial,
		Iterations: 25,
		Seed:       1,
	})
	searchCalls := oracle.costCalls

	t.Logf(
		"full board %dx%d: solve=%d scoring calls, 25 search rounds=%d (%d per round)",
		benchTasks, benchResources, solveCalls, searchCalls, searchCalls/25,
	)

	assert.LessOrEqual(t, searchCalls, 6*solveCalls,
		"25 search rounds should not cost more than a handful of full solves")
}
