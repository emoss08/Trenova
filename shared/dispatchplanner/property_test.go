package dispatchplanner_test

import (
	"math/rand"
	"testing"

	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type matrixOracle struct {
	cost [][]float64
}

func (o *matrixOracle) Cost(task, resource int) float64 { return o.cost[task][resource] }

func (o *matrixOracle) Commit(int, int) {}

func (o *matrixOracle) Rebuild(
	assignments []dispatchplanner.Assignment,
) []dispatchplanner.Assignment {
	replayed := make([]dispatchplanner.Assignment, len(assignments))
	for i, assignment := range assignments {
		assignment.Cost = o.Cost(assignment.Task, assignment.Resource)
		replayed[i] = assignment
	}
	return replayed
}

func randomInstance(rng *rand.Rand) (*matrixOracle, int, int, dispatchplanner.Options) {
	tasks := 1 + rng.Intn(30)
	resources := 1 + rng.Intn(12)

	cost := make([][]float64, tasks)
	for task := range cost {
		cost[task] = make([]float64, resources)
		for resource := range cost[task] {
			if rng.Float64() < 0.35 {
				cost[task][resource] = dispatchplanner.Forbidden
				continue
			}
			cost[task][resource] = rng.Float64() * 100
		}
	}

	options := dispatchplanner.Options{}
	if rng.Float64() < 0.7 {
		options.MaxPerResource = 1 + rng.Intn(4)
	}

	return &matrixOracle{cost: cost}, tasks, resources, options
}

func assertPlanIsSound(
	t *testing.T,
	instance int,
	result dispatchplanner.Result,
	oracle *matrixOracle,
	tasks int,
	options dispatchplanner.Options,
) {
	t.Helper()

	seen := make(map[int]bool, tasks)
	perResource := make(map[int]int, tasks)
	nextSequence := make(map[int]int, tasks)
	summed := 0.0

	for _, assignment := range result.Assignments {
		assert.False(t, seen[assignment.Task],
			"instance %d: task %d was assigned more than once", instance, assignment.Task)
		seen[assignment.Task] = true

		assert.Less(t, oracle.Cost(assignment.Task, assignment.Resource),
			dispatchplanner.Forbidden,
			"instance %d: task %d was given to resource %d despite being blocked for it",
			instance, assignment.Task, assignment.Resource)

		assert.Equal(t, nextSequence[assignment.Resource], assignment.Sequence,
			"instance %d: resource %d has a gap in its sequence", instance, assignment.Resource)
		nextSequence[assignment.Resource]++

		perResource[assignment.Resource]++
		summed += assignment.Cost
	}

	if options.MaxPerResource > 0 {
		for resource, count := range perResource {
			assert.LessOrEqual(t, count, options.MaxPerResource,
				"instance %d: resource %d exceeded its cap", instance, resource)
		}
	}

	for _, task := range result.Unassigned {
		assert.False(t, seen[task],
			"instance %d: task %d is both assigned and unassigned", instance, task)
		seen[task] = true
	}

	assert.Len(t, seen, tasks,
		"instance %d: every task must be either assigned or reported uncovered", instance)
	assert.InDelta(t, summed, result.TotalCost, 0.001)
}

func TestSolve_NeverViolatesItsConstraints(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(20260817)) //nolint:gosec // reproducible fixtures

	for instance := range 300 {
		oracle, tasks, resources, options := randomInstance(rng)

		result := dispatchplanner.Solve(dispatchplanner.SolveParams{
			Tasks:     tasks,
			Resources: resources,
			Oracle:    oracle,
			Options:   options,
		})

		assertPlanIsSound(t, instance, result, oracle, tasks, options)
	}
}

func TestImprove_NeverViolatesItsConstraints(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(20260818)) //nolint:gosec // reproducible fixtures

	for instance := range 200 {
		oracle, tasks, resources, options := randomInstance(rng)

		initial := dispatchplanner.Solve(dispatchplanner.SolveParams{
			Tasks:     tasks,
			Resources: resources,
			Oracle:    oracle,
			Options:   options,
		})

		improved := dispatchplanner.Improve(dispatchplanner.ImproveParams{
			Resources:  resources,
			Oracle:     oracle,
			Options:    options,
			Initial:    initial,
			Iterations: 20,
			Seed:       int64(instance),
		})

		assertPlanIsSound(t, instance, improved, oracle, tasks, options)

		assert.GreaterOrEqual(t, len(improved.Assignments), len(initial.Assignments),
			"instance %d: search covered fewer moves than it started with", instance)
	}
}

func TestSolve_ReportsTasksNoResourceCanTake(t *testing.T) {
	t.Parallel()

	oracle := &matrixOracle{cost: [][]float64{
		{dispatchplanner.Forbidden, dispatchplanner.Forbidden},
		{5, dispatchplanner.Forbidden},
		{dispatchplanner.Forbidden, dispatchplanner.Forbidden},
	}}

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     3,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 1)
	assert.Equal(t, []int{0, 2}, result.Unassigned)
}
