package dispatchplanner

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyChoice(t *testing.T, s *search) int64 {
	t.Helper()

	require.True(t, s.choice.valid, "the operator found no move")
	delta := s.choice.delta
	s.apply(&s.choice)
	requireConsistent(t, s)

	return delta
}

func TestBestInsertion_CoversAFreeTaskCheaperThanItsCoveragePrice(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{1_000},
		first:     map[pair]int64{{0, 0}: 100},
	})
	s := searchWith(t, p, nil)

	s.choice.reset()
	s.bestInsertion(0)

	assert.Equal(t, int64(-900), applyChoice(t, s))
	assert.Equal(t, []int{0}, s.cur.chains[0])
}

func TestBestInsertion_PlacesATaskInItsTimeOrderedSlot(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{50_000, 10_000, 50_000},
		opens:     []int64{0, 10, 20},
		first:     map[pair]int64{{0, 0}: 100},
		next:      map[pair]int64{{0, 1}: 100, {1, 2}: 100, {0, 2}: 500},
	})
	s := searchWith(t, p, map[int][]int{0: {0, 2}})

	s.choice.reset()
	s.bestInsertion(1)

	assert.Equal(t, int64(-300-10_000), applyChoice(t, s))
	assert.Equal(t, []int{0, 1, 2}, s.cur.chains[0])
}

func TestBestInsertion_SkipsAFullChain(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{50_000, 50_000},
		first:     map[pair]int64{{0, 0}: 100, {0, 1}: 100},
		next:      map[pair]int64{{0, 1}: 100},
		options:   Options{MaxPerResource: 1},
	})
	s := searchWith(t, p, map[int][]int{0: {0}})

	s.choice.reset()
	s.bestInsertion(1)

	assert.False(t, s.choice.valid)
}

func TestRemoval_LeavesATaskThatCostsMoreThanItsCoveragePrice(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{1_000},
		first:     map[pair]int64{{0, 0}: 5_000},
	})
	s := searchWith(t, p, map[int][]int{0: {0}})

	s.choice.reset()
	s.removal(0)

	assert.Equal(t, int64(-4_000), applyChoice(t, s))
	assert.Empty(t, s.cur.chains[0])
	assert.Equal(t, Unassigned, s.cur.owner[0])
}

func TestBestRelocation_MovesATaskToACheaperDriver(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 2,
		coverage:  []int64{10_000},
		first:     map[pair]int64{{0, 0}: 500, {1, 0}: 100},
	})
	s := searchWith(t, p, map[int][]int{0: {0}})

	s.choice.reset()
	s.bestRelocation(0)

	assert.Equal(t, int64(-400), applyChoice(t, s))
	assert.Empty(t, s.cur.chains[0])
	assert.Equal(t, []int{0}, s.cur.chains[1])
}

func TestBestExchange_SwapsTasksBetweenTwoDrivers(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 2,
		coverage:  []int64{10_000, 10_000},
		first: map[pair]int64{
			{0, 0}: 500, {0, 1}: 100,
			{1, 0}: 100, {1, 1}: 500,
		},
	})
	s := searchWith(t, p, map[int][]int{0: {0}, 1: {1}})

	s.choice.reset()
	s.bestExchange(0)

	assert.Equal(t, int64(-800), applyChoice(t, s))
	assert.Equal(t, []int{1}, s.cur.chains[0])
	assert.Equal(t, []int{0}, s.cur.chains[1])
}

func TestBestExchange_RespectsTheSparseNeighborhood(t *testing.T) {
	t.Parallel()

	p, timeline := tableProblem(&tableSpec{
		resources: 2,
		coverage:  []int64{10_000, 10_000},
		first: map[pair]int64{
			{0, 0}: 500,
			{1, 0}: 100, {1, 1}: 500,
		},
	})
	s := searchWith(t, p, map[int][]int{0: {0}, 1: {1}})
	before := timeline.calls

	s.choice.reset()
	s.bestExchange(0)

	assert.False(t, s.choice.valid)
	assert.Equal(t, before, timeline.calls, "a pair outside the candidate arcs is never evaluated")
}

func TestBestReplacement_CoversAValuableTaskInPlaceOfACheapOne(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{1_000, 50_000},
		first:     map[pair]int64{{0, 0}: 100, {0, 1}: 100},
		options:   Options{MaxPerResource: 1},
	})
	s := searchWith(t, p, map[int][]int{0: {0}})

	s.choice.reset()
	s.bestReplacement(1)

	assert.Equal(t, int64(-49_000), applyChoice(t, s))
	assert.Equal(t, []int{1}, s.cur.chains[0])
	assert.Equal(t, Unassigned, s.cur.owner[0])
}

func TestBestTailExchange_SwapsChainTails(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 2,
		coverage:  []int64{50_000, 50_000, 50_000, 50_000},
		first:     map[pair]int64{{0, 0}: 100, {1, 2}: 100},
		next: map[pair]int64{
			{0, 1}: 900, {0, 3}: 100,
			{2, 3}: 900, {2, 1}: 100,
		},
	})
	s := searchWith(t, p, map[int][]int{0: {0, 1}, 1: {2, 3}})

	s.choice.reset()
	s.bestTailExchange(3)

	assert.Equal(t, int64(-1_600), applyChoice(t, s))
	assert.Equal(t, []int{0, 3}, s.cur.chains[0])
	assert.Equal(t, []int{2, 1}, s.cur.chains[1])
}

func TestReorder_SwapsAdjacentTasksIntoTheCheaperOrder(t *testing.T) {
	t.Parallel()

	p, _ := tableProblem(&tableSpec{
		resources: 1,
		coverage:  []int64{50_000, 50_000},
		first:     map[pair]int64{{0, 0}: 100, {0, 1}: 100},
		next:      map[pair]int64{{0, 1}: 100, {1, 0}: 900},
	})
	s := searchWith(t, p, map[int][]int{0: {1, 0}})

	s.choice.reset()
	s.reorder(1)

	assert.Equal(t, int64(-800), applyChoice(t, s))
	assert.Equal(t, []int{0, 1}, s.cur.chains[0])
}

func TestOperators_KeepTheIncrementalObjectiveExact(t *testing.T) {
	t.Parallel()

	for seed := range uint64(20) {
		rng := rand.New(rand.NewPCG(seed, 7))
		spec := &tableSpec{
			resources: 4,
			coverage:  make([]int64, 9),
			opens:     make([]int64, 9),
			first:     map[pair]int64{},
			next:      map[pair]int64{},
			incumbent: make([]int, 9),
			options:   Options{MaxPerResource: 3, StabilityCents: 300, RandomSeed: seed},
		}
		for task := range spec.coverage {
			spec.coverage[task] = 500 + rng.Int64N(3_000)
			spec.opens[task] = rng.Int64N(100)
			spec.incumbent[task] = rng.IntN(spec.resources+1) - 1
			for resource := range spec.resources {
				if rng.IntN(2) == 0 {
					spec.first[pair{resource, task}] = rng.Int64N(2_000)
				}
			}
			for other := range spec.coverage {
				if other != task && rng.IntN(3) == 0 {
					spec.next[pair{task, other}] = rng.Int64N(2_000) - 200
				}
			}
		}
		p, _ := tableProblem(spec)
		s := newSearch(p)
		s.seed()
		requireConsistent(t, s)

		for range 300 {
			s.propose(operator(rng.IntN(int(operatorCount))))
			if !s.choice.valid {
				continue
			}
			s.apply(&s.choice)
			requireConsistent(t, s)
		}
	}
}

func TestAccepts_IsDeterministicForASeed(t *testing.T) {
	t.Parallel()

	draw := func() []bool {
		rng := rand.New(rand.NewPCG(42, 42^seedMixer))
		out := make([]bool, 200)
		for i := range out {
			out[i] = accepts(int64(i%50), 20, rng)
		}
		return out
	}

	first := draw()
	assert.Equal(t, first, draw())
	assert.Contains(t, first, true)
	assert.Contains(t, first, false)
}

func TestAccepts_AlwaysTakesImprovementsAndNeverClimbsCold(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		assert.True(t, accepts(-5, 0, rng))
		assert.True(t, accepts(0, 0, rng))
		assert.False(t, accepts(1, 0, rng))
	}
}

func TestInitialTemperature_AcceptsTheMedianClimbHalfTheTime(t *testing.T) {
	t.Parallel()

	temperature := initialTemperature([]int64{10, 300, 100})

	assert.InDelta(t, 0.5, math.Exp(-100/temperature), 1e-9)
	assert.InDelta(t, 1.0, initialTemperature(nil), 1e-9)
}
