package dispatchplanner_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticOracle scores from a fixed matrix and ignores state changes.
type staticOracle struct {
	cost    [][]float64
	commits [][2]int
}

func (o *staticOracle) Cost(task, resource int) float64 { return o.cost[task][resource] }

func (o *staticOracle) Commit(task, resource int) {
	o.commits = append(o.commits, [2]int{task, resource})
}

// chainOracle models the property the planner exists for: once a resource takes a
// task, the next task becomes cheaper for that same resource and no other.
type chainOracle struct {
	base     [][]float64
	discount float64
	taken    map[int]int
}

func newChainOracle(base [][]float64, discount float64) *chainOracle {
	return &chainOracle{base: base, discount: discount, taken: make(map[int]int)}
}

func (o *chainOracle) Cost(task, resource int) float64 {
	cost := o.base[task][resource]
	if cost >= dispatchplanner.Forbidden {
		return cost
	}
	return cost - o.discount*float64(o.taken[resource])
}

func (o *chainOracle) Commit(_, resource int) { o.taken[resource]++ }

func TestSolve_AssignsCheapestPairFirst(t *testing.T) {
	t.Parallel()

	oracle := &staticOracle{cost: [][]float64{
		{9, 4},
		{3, 8},
	}}

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     2,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 2)
	assert.Equal(t, 1, result.Assignments[0].Task)
	assert.Equal(t, 0, result.Assignments[0].Resource)
	assert.Equal(t, 0, result.Assignments[1].Task)
	assert.Equal(t, 1, result.Assignments[1].Resource)
	assert.Empty(t, result.Unassigned)
	assert.InDelta(t, 7.0, result.TotalCost, 1e-9)
}

func TestSolve_ChainsMultipleTasksOntoOneResource(t *testing.T) {
	t.Parallel()

	// Resource 0 starts marginally worse everywhere, but each committed task makes
	// it cheaper. A one-to-one matcher would spread these across both resources.
	oracle := newChainOracle([][]float64{
		{10, 11},
		{10, 11},
		{10, 11},
	}, 6)

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     3,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 3)
	for _, assignment := range result.Assignments {
		assert.Equal(t, 0, assignment.Resource)
	}
	assert.Equal(t, []int{0, 1, 2}, sequences(result))
	assert.Empty(t, result.Unassigned)
}

func TestSolve_SequenceIsPerResource(t *testing.T) {
	t.Parallel()

	oracle := newChainOracle([][]float64{
		{1, 50},
		{50, 1},
		{2, 50},
		{50, 2},
	}, 0)

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     4,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 4)

	byResource := make(map[int][]int)
	for _, assignment := range result.Assignments {
		byResource[assignment.Resource] = append(
			byResource[assignment.Resource],
			assignment.Sequence,
		)
	}

	assert.Equal(t, []int{0, 1}, byResource[0])
	assert.Equal(t, []int{0, 1}, byResource[1])
}

func TestSolve_SkipsForbiddenPairings(t *testing.T) {
	t.Parallel()

	oracle := &staticOracle{cost: [][]float64{
		{dispatchplanner.Forbidden, 5},
		{dispatchplanner.Forbidden, dispatchplanner.Forbidden},
	}}

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     2,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 1)
	assert.Equal(t, 0, result.Assignments[0].Task)
	assert.Equal(t, 1, result.Assignments[0].Resource)
	assert.Equal(t, []int{1}, result.Unassigned)
}

func TestSolve_HonorsMaxPerResource(t *testing.T) {
	t.Parallel()

	oracle := newChainOracle([][]float64{
		{1, 40},
		{1, 40},
		{1, 40},
	}, 0)

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     3,
		Resources: 2,
		Oracle:    oracle,
		Options:   dispatchplanner.Options{MaxPerResource: 2},
	})

	require.Len(t, result.Assignments, 3)

	counts := make(map[int]int)
	for _, assignment := range result.Assignments {
		counts[assignment.Resource]++
	}
	assert.Equal(t, 2, counts[0])
	assert.Equal(t, 1, counts[1])
}

func TestSolve_LeavesTasksUnassignedWhenResourcesAreCapped(t *testing.T) {
	t.Parallel()

	oracle := &staticOracle{cost: [][]float64{{1}, {1}, {1}}}

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     3,
		Resources: 1,
		Oracle:    oracle,
		Options:   dispatchplanner.Options{MaxPerResource: 1},
	})

	require.Len(t, result.Assignments, 1)
	assert.Equal(t, []int{1, 2}, result.Unassigned)
}

func TestSolve_RecostsOnlyTheCommittedResource(t *testing.T) {
	t.Parallel()

	oracle := newChainOracle([][]float64{
		{5, 5},
		{5, 5},
	}, 1)

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     2,
		Resources: 2,
		Oracle:    oracle,
	})

	require.Len(t, result.Assignments, 2)
	// Both land on resource 0: after the first commit it costs 4 against the
	// untouched 5 on resource 1.
	assert.Equal(t, 0, result.Assignments[0].Resource)
	assert.Equal(t, 0, result.Assignments[1].Resource)
	assert.InDelta(t, 9.0, result.TotalCost, 1e-9)
}

func TestSolve_IsDeterministicOnTies(t *testing.T) {
	t.Parallel()

	build := func() *staticOracle {
		return &staticOracle{cost: [][]float64{
			{3, 3, 3},
			{3, 3, 3},
			{3, 3, 3},
		}}
	}

	first := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks: 3, Resources: 3, Oracle: build(),
	})

	for range 25 {
		repeat := dispatchplanner.Solve(dispatchplanner.SolveParams{
			Tasks: 3, Resources: 3, Oracle: build(),
		})
		assert.Equal(t, first.Assignments, repeat.Assignments)
		assert.Equal(t, first.Unassigned, repeat.Unassigned)
	}
}

func TestSolve_CommitsOnlyChosenPairings(t *testing.T) {
	t.Parallel()

	oracle := &staticOracle{cost: [][]float64{
		{1, 9},
		{9, 2},
	}}

	dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks: 2, Resources: 2, Oracle: oracle,
	})

	assert.Equal(t, [][2]int{{0, 0}, {1, 1}}, oracle.commits)
}

func TestSolve_HandlesDegenerateInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		params         dispatchplanner.SolveParams
		wantUnassigned []int
	}{
		{
			name:           "no tasks",
			params:         dispatchplanner.SolveParams{Tasks: 0, Resources: 3, Oracle: &staticOracle{}},
			wantUnassigned: []int{},
		},
		{
			name:           "no resources",
			params:         dispatchplanner.SolveParams{Tasks: 2, Resources: 0, Oracle: &staticOracle{}},
			wantUnassigned: []int{0, 1},
		},
		{
			name:           "nil oracle",
			params:         dispatchplanner.SolveParams{Tasks: 2, Resources: 2},
			wantUnassigned: []int{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := dispatchplanner.Solve(tt.params)

			assert.Empty(t, result.Assignments)
			assert.Equal(t, tt.wantUnassigned, result.Unassigned)
			assert.Zero(t, result.TotalCost)
		})
	}
}

func sequences(result dispatchplanner.Result) []int {
	out := make([]int, 0, len(result.Assignments))
	for _, assignment := range result.Assignments {
		out = append(out, assignment.Sequence)
	}
	return out
}
