package dispatchplanner

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolve_RejectsInputsOfDifferentLengths(t *testing.T) {
	t.Parallel()

	_, err := Solve(context.Background(), &Problem{
		Resources: 1,
		Arcs:      make([][]assignmentsolver.Arc, 2),
		Coverage:  []int64{1},
		Timeline:  &tableTimeline{},
	})
	require.ErrorIs(t, err, ErrShape)

	_, err = Solve(context.Background(), &Problem{
		Resources: 1,
		Arcs:      make([][]assignmentsolver.Arc, 1),
		Coverage:  []int64{1},
	})
	require.ErrorIs(t, err, ErrMissingTimeline)
}

func TestSolve_LeavesEveryTaskUncoveredWithoutResources(t *testing.T) {
	t.Parallel()

	result, err := Solve(context.Background(), &Problem{
		Arcs:     make([][]assignmentsolver.Arc, 3),
		Coverage: []int64{100, 200, 300},
	})

	require.NoError(t, err)
	assert.Empty(t, result.Assignments)
	assert.Equal(t, []int{0, 1, 2}, result.Unassigned)
	assert.Equal(t, int64(600), result.Objective)
}

func TestSolve_SeedsFromTheExactAssignment(t *testing.T) {
	t.Parallel()

	spec := &tableSpec{
		resources: 3,
		coverage:  []int64{900, 900, 900, 900},
		first: map[pair]int64{
			{0, 0}: 400, {1, 0}: 100, {2, 0}: 300,
			{0, 1}: 200, {1, 1}: 150,
			{1, 2}: 700, {2, 2}: 250,
			{0, 3}: 950,
		},
		options: Options{MaxPerResource: 1},
	}
	p, _ := tableProblem(spec)
	exact := assignmentsolver.Solve(assignmentsolver.Problem{
		Cols:      p.Resources,
		Rows:      p.Arcs,
		Uncovered: p.Coverage,
	})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	assert.Equal(t, exact.Objective, result.SeedObjective)
	assert.Equal(t, exact.Objective, result.Objective)
	for _, assignment := range result.Assignments {
		assert.Equal(t, exact.RowAssignment[assignment.Task], assignment.Resource)
	}
	assert.Equal(t, []int{3}, result.Unassigned)
}

func TestSolve_ChainsAFollowOnTaskThroughItsPredecessor(t *testing.T) {
	t.Parallel()

	p, timeline := tableProblem(&tableSpec{
		resources: 3,
		coverage:  []int64{10_000, 10_000},
		opens:     []int64{0, 100},
		first:     map[pair]int64{{0, 0}: 300},
		next:      map[pair]int64{{0, 1}: 50},
		options:   Options{MaxPerResource: 3, Iterations: 200},
	})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	require.Len(t, result.Assignments, 2)
	assert.Equal(
		t,
		Assignment{Task: 0, Resource: 0, Sequence: 0, Cost: 300},
		withoutPrices(result.Assignments[0]),
	)
	assert.Equal(
		t,
		Assignment{Task: 1, Resource: 0, Sequence: 1, Cost: 50},
		withoutPrices(result.Assignments[1]),
	)
	assert.Empty(t, result.Unassigned)
	assert.Equal(t, int64(350), result.Objective)
	assert.Equal(t, int64(300+10_000), result.SeedObjective)
	assert.LessOrEqual(
		t,
		timeline.calls,
		4,
		"only the candidate arcs and the chain successor are evaluated",
	)
}

func withoutPrices(a Assignment) Assignment {
	a.Alternatives = nil
	return a
}

func TestSolve_StabilityPenaltyKeepsTheStandingDriver(t *testing.T) {
	t.Parallel()

	problem := func(stability int64) *Problem {
		p, _ := tableProblem(&tableSpec{
			resources: 2,
			coverage:  []int64{10_000},
			first:     map[pair]int64{{0, 0}: 500, {1, 0}: 400},
			incumbent: []int{0},
			options: Options{
				MaxPerResource: 2,
				Iterations:     100,
				StabilityCents: stability,
			},
		})
		return p
	}

	churn, err := Solve(context.Background(), problem(0))
	require.NoError(t, err)
	require.Len(t, churn.Assignments, 1)
	assert.Equal(t, 1, churn.Assignments[0].Resource)

	stable, err := Solve(context.Background(), problem(250))
	require.NoError(t, err)
	require.Len(t, stable.Assignments, 1)
	assert.Equal(t, 0, stable.Assignments[0].Resource)
	assert.Equal(t, int64(500), stable.Objective)
}

func TestSolve_PricesAlternativesAndDriverValue(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 3,
		coverage:  []int64{10_000},
		first:     map[pair]int64{{0, 0}: 100, {1, 0}: 300, {2, 0}: 700},
		options:   Options{MaxPerResource: 2, Alternatives: 3},
	})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	require.Len(t, result.Assignments, 1)
	assert.Equal(t, 0, result.Assignments[0].Resource)
	assert.Equal(t, []Alternative{
		{Resource: 1, ExtraCost: 200},
		{Resource: 2, ExtraCost: 600},
	}, result.Assignments[0].Alternatives)
	assert.Equal(t, []int64{200, 0, 0}, result.ResourceValue)
}

func TestSolve_DriverValueFallsBackToTheCoveragePrice(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{1_000},
		first:     map[pair]int64{{0, 0}: 100},
		options:   Options{Alternatives: 3},
	})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	assert.Empty(t, result.Assignments[0].Alternatives)
	assert.Equal(t, []int64{900}, result.ResourceValue)
}

func TestSolve_IsDeterministicForASeed(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(11, 80, 50)
	run := func() Result {
		p, _ := board.problem(12, Options{
			MaxPerResource: 3,
			Iterations:     4_000,
			RandomSeed:     5,
			Alternatives:   2,
			Clock:          steppingClock(time.Millisecond),
		})
		result, err := Solve(context.Background(), p)
		require.NoError(t, err)
		result.Stats.Elapsed = 0
		return result
	}

	assert.Equal(t, run(), run())
}

func TestSolve_StopsAtTheBudget(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(3, 120, 80)
	p, _ := board.problem(12, Options{
		MaxPerResource: 3,
		Iterations:     1 << 30,
		Budget:         200 * time.Millisecond,
		Clock:          steppingClock(time.Millisecond),
	})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	assert.Less(t, result.Stats.Iterations, 1<<30)
	assert.LessOrEqual(t, result.Stats.Iterations, 200*checkEvery)
	assert.LessOrEqual(t, result.Objective, result.SeedObjective)
}

func TestSolve_StopsAtTheWallClockBudget(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(4, 300, 200)
	p, _ := board.problem(16, Options{
		MaxPerResource: 3,
		Iterations:     1 << 30,
		Budget:         100 * time.Millisecond,
	})

	started := time.Now()
	result, err := Solve(context.Background(), p)
	elapsed := time.Since(started)

	require.NoError(t, err)
	assert.Less(t, result.Stats.Iterations, 1<<30)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestSolve_StopsWhenTheContextIsDone(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(5, 60, 40)
	p, _ := board.problem(12, Options{MaxPerResource: 3, Iterations: 1 << 30})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Solve(ctx, p)

	require.NoError(t, err)
	assert.Zero(t, result.Stats.Iterations)
	assert.Equal(t, result.SeedObjective, result.Objective)
}

func TestSolve_EvaluatesOnlySparseCandidates(t *testing.T) {
	t.Parallel()

	board := newSynthBoard(6, 200, 200)
	p, timeline := board.problem(8, Options{MaxPerResource: 3, Iterations: 2_000, RandomSeed: 2})

	result, err := Solve(context.Background(), p)

	require.NoError(t, err)
	for _, assignment := range result.Assignments {
		if assignment.Sequence > 0 {
			continue
		}
		assert.True(t, hasArc(p.Arcs[assignment.Task], assignment.Resource),
			"task %d starts a chain on resource %d outside its candidate arcs",
			assignment.Task, assignment.Resource)
	}
	assert.Less(t, timeline.calls, 200*200, "the search never scans every pair")
}

func hasArc(arcs []assignmentsolver.Arc, resource int) bool {
	for _, arc := range arcs {
		if arc.Col == resource {
			return true
		}
	}
	return false
}

func TestPrefixCache_ReusesSharedPrefixes(t *testing.T) {
	t.Parallel()

	timeline := &tableTimeline{
		first: map[pair]int64{{0, 0}: 10},
		next:  map[pair]int64{{0, 1}: 20, {0, 2}: 30},
	}
	cache := newPrefixCache(timeline, minCacheNodes)

	cost, ok := cache.chain(0, []int{0, 1})
	require.True(t, ok)
	assert.Equal(t, int64(30), cost)
	assert.Equal(t, 2, timeline.calls)

	cost, ok = cache.chain(0, []int{0, 1})
	require.True(t, ok)
	assert.Equal(t, int64(30), cost)
	assert.Equal(t, 2, timeline.calls, "a repeated chain is free")

	cost, ok = cache.chain(0, []int{0, 2})
	require.True(t, ok)
	assert.Equal(t, int64(40), cost)
	assert.Equal(t, 3, timeline.calls, "a shared prefix is evaluated once")

	_, ok = cache.chain(0, []int{1})
	assert.False(t, ok)
	_, ok = cache.chain(0, []int{1})
	assert.False(t, ok)
	assert.Equal(t, 4, timeline.calls, "an infeasible step is cached too")
}

func TestPrefixCache_FlushesAtItsLimitAndResetsTheTimeline(t *testing.T) {
	t.Parallel()

	timeline := &tableTimeline{
		first: map[pair]int64{{0, 0}: 10},
		next:  map[pair]int64{{0, 1}: 20, {0, 2}: 30},
	}
	cache := newPrefixCache(timeline, 3)

	_, ok := cache.chain(0, []int{0, 1})
	require.True(t, ok)
	cost, ok := cache.chain(0, []int{0, 2})

	require.True(t, ok)
	assert.Equal(t, int64(40), cost)
	assert.Equal(t, 1, cache.flushes)
	assert.Equal(t, 1, timeline.resets)
	assert.Equal(t, 4, timeline.calls)
}
