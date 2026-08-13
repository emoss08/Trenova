package dispatchplanner

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMostRegret_PrefersTheTaskWithTheMostToLose(t *testing.T) {
	t.Parallel()

	// Task 0 barely cares which resource it gets; task 1 has one good option and one
	// terrible one. Greedy would take task 0 first because it is cheaper outright.
	cost := [][]float64{
		{5, 6},
		{10, 90},
	}
	placed := make([]bool, 2)
	committed := make([]int, 2)

	index, resource, best := mostRegret(cost, placed, committed, Options{})

	assert.Equal(t, 1, index)
	assert.Equal(t, 0, resource)
	assert.InDelta(t, 10.0, best, 0.001)

	greedyIndex, _, _ := cheapest(cost, placed, committed, Options{})
	assert.Equal(t, 0, greedyIndex, "greedy and regret genuinely disagree here")
}

func TestMostRegret_PlacesSingleOptionTasksFirst(t *testing.T) {
	t.Parallel()

	// Task 1 can only go to resource 1. Even though task 0 has a wider spread of
	// costs, the task with nowhere else to go must be placed before its one option
	// is taken.
	cost := [][]float64{
		{10, 80},
		{Forbidden, 40},
	}
	placed := make([]bool, 2)
	committed := make([]int, 2)

	index, resource, _ := mostRegret(cost, placed, committed, Options{})

	assert.Equal(t, 1, index)
	assert.Equal(t, 1, resource)
}

func TestMostRegret_SkipsPlacedAndCappedOptions(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{1, 50},
		{2, 60},
	}

	t.Run("placed tasks are ignored", func(t *testing.T) {
		t.Parallel()

		index, _, _ := mostRegret(cost, []bool{true, false}, make([]int, 2), Options{})
		assert.Equal(t, 1, index)
	})

	t.Run("capped resources drop out of the regret calculation", func(t *testing.T) {
		t.Parallel()

		index, resource, best := mostRegret(
			cost,
			make([]bool, 2),
			[]int{1, 0},
			Options{MaxPerResource: 1},
		)

		assert.NotEqual(t, Unassigned, index)
		assert.Equal(t, 1, resource, "resource 0 is full, so only resource 1 remains")
		assert.GreaterOrEqual(t, best, 50.0)
	})

	t.Run("nothing feasible reports unassigned", func(t *testing.T) {
		t.Parallel()

		index, resource, _ := mostRegret(
			[][]float64{{Forbidden, Forbidden}},
			make([]bool, 1),
			make([]int, 2),
			Options{},
		)

		assert.Equal(t, Unassigned, index)
		assert.Equal(t, Unassigned, resource)
	})
}

func TestRuinCount_NeverTearsOutASingleAssignment(t *testing.T) {
	t.Parallel()

	rng := newTestRand()

	for total := 2; total <= 50; total++ {
		for range 20 {
			count := ruinCount(total, rng)
			assert.GreaterOrEqual(t, count, min(minRuinCount, total),
				"total %d produced a removal too small to restructure", total)
			assert.LessOrEqual(t, count, total)
		}
	}

	assert.Equal(t, 1, ruinCount(1, rng))
	assert.Equal(t, 0, ruinCount(0, rng))
}

func newTestRand() *rand.Rand { return rand.New(rand.NewSource(1)) }
