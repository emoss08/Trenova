package dispatchplanner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolve_IsNeverWorseThanItsExactSeed(t *testing.T) {
	t.Parallel()

	for seed := range uint64(25) {
		board := newSynthBoard(seed, 40+int(seed)*3, 20+int(seed)*2)
		p, _ := board.problem(10, Options{
			MaxPerResource: 1 + int(seed%4),
			Iterations:     1_500,
			RandomSeed:     seed,
			Alternatives:   2,
		})

		result, err := Solve(context.Background(), p)
		require.NoError(t, err)

		assert.LessOrEqual(t, result.Objective, result.SeedObjective, "seed %d", seed)
		metrics := board.measure(&result, p.Options.MaxPerResource)
		assert.True(t, metrics.feasible, "seed %d planned an infeasible chain", seed)
		assert.Equal(t, metrics.objective, result.Objective, "seed %d objective drifted", seed)
		assert.Equal(t, len(board.moves), metrics.covered+len(result.Unassigned))
		requirePartition(t, &result, len(board.moves))
	}
}

func TestSolve_ChainsEveryTaskAtMostOnceInSequence(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(99, 150, 60)
	p, _ := board.problem(12, Options{MaxPerResource: 4, Iterations: 6_000, RandomSeed: 3})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	requirePartition(t, &result, len(board.moves))
	next := make(map[int]int)
	for _, assignment := range result.Assignments {
		assert.Equal(t, next[assignment.Resource], assignment.Sequence)
		next[assignment.Resource]++
		assert.LessOrEqual(t, next[assignment.Resource], 4)
	}
	metrics := board.measure(&result, 4)
	assert.True(t, metrics.feasible)
	assert.Greater(t, metrics.covered, 60, "chains cover more moves than there are drivers")
}

func requirePartition(t *testing.T, result *Result, tasks int) {
	t.Helper()

	seen := make([]bool, tasks)
	for _, assignment := range result.Assignments {
		require.False(t, seen[assignment.Task], "task %d is assigned twice", assignment.Task)
		seen[assignment.Task] = true
	}
	for _, task := range result.Unassigned {
		require.False(t, seen[task], "task %d is both assigned and uncovered", task)
		seen[task] = true
	}
	for task, ok := range seen {
		require.True(t, ok, "task %d is missing from the result", task)
	}
}
