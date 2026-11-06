package assignmentsolver_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const never = int64(1_000_000_000)

func dense(cost [][]int64, uncovered int64) assignmentsolver.Problem {
	p := assignmentsolver.Problem{Rows: make([][]assignmentsolver.Arc, len(cost))}
	for i, row := range cost {
		p.Cols = max(p.Cols, len(row))
		for j, c := range row {
			if c >= 0 {
				p.Rows[i] = append(p.Rows[i], assignmentsolver.Arc{Col: j, Cost: c})
			}
		}
		p.Uncovered = append(p.Uncovered, uncovered)
	}
	return p
}

func TestSolve_FindsTheOptimumOfASquareBoard(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve(dense([][]int64{
		{4, 1, 3},
		{2, 0, 5},
		{3, 2, 2},
	}, never))

	assert.Equal(t, int64(5), result.TotalCost)
	assert.Equal(t, []int{1, 0, 2}, result.RowAssignment)
	assert.Equal(t, []int{1, 0, 2}, result.ColAssignment)
}

func TestSolve_BeatsTakingTheCheapestPairFirst(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve(dense([][]int64{
		{1, 2},
		{9, 100},
	}, never))

	assert.Equal(t, int64(11), result.TotalCost)
	assert.Equal(t, []int{1, 0}, result.RowAssignment)
}

func TestSolve_NeverPairsARowWithAColumnItHasNoArcTo(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve(dense([][]int64{
		{-1, 4},
		{-1, -1},
	}, never))

	assert.Equal(t, []int{1, assignmentsolver.Unassigned}, result.RowAssignment)
	assert.Equal(t, []int{assignmentsolver.Unassigned, 0}, result.ColAssignment)
}

func TestSolve_LeavesARowUncoveredWhenCoveringItCostsMoreThanLeavingIt(t *testing.T) {
	t.Parallel()

	p := dense([][]int64{{50}, {80}}, 0)
	p.Uncovered = []int64{100, 60}

	result := assignmentsolver.Solve(p)

	assert.Equal(t, []int{0, assignmentsolver.Unassigned}, result.RowAssignment)
	assert.Equal(t, int64(50), result.TotalCost)
}

func TestSolve_GivesAScarceColumnToTheRowThatLosesMostWithoutIt(t *testing.T) {
	t.Parallel()

	p := dense([][]int64{{10}, {30}}, 0)
	p.Uncovered = []int64{20, 500}

	result := assignmentsolver.Solve(p)

	assert.Equal(t, []int{assignmentsolver.Unassigned, 0}, result.RowAssignment,
		"the urgent row takes the only driver even though the other pairs more cheaply")
}

func TestSolve_HandlesMoreRowsThanColumns(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve(dense([][]int64{
		{5, 1},
		{2, 8},
		{9, 4},
	}, never))

	assert.Equal(t, []int{1, 0, assignmentsolver.Unassigned}, result.RowAssignment)
	assert.Equal(t, int64(3), result.TotalCost)
}

func TestSolve_ReturnsEmptyAssignmentsForAnEmptyProblem(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve(assignmentsolver.Problem{Cols: 3})

	assert.Empty(t, result.RowAssignment)
	assert.Equal(t, []int{-1, -1, -1}, result.ColAssignment)
	assert.Zero(t, result.TotalCost)
}

func TestSolve_GivesTheSameAnswerEveryTimeWhenCostsTie(t *testing.T) {
	t.Parallel()

	p := dense([][]int64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}, never)
	first := assignmentsolver.Solve(p)

	for range 20 {
		assert.Equal(t, first, assignmentsolver.Solve(p))
	}
}

func bruteForce(p assignmentsolver.Problem) int64 {
	used := make([]bool, p.Cols)
	var best int64 = -1
	var walk func(row int, total int64)
	walk = func(row int, total int64) {
		if row == len(p.Rows) {
			if best < 0 || total < best {
				best = total
			}
			return
		}
		walk(row+1, total+p.Uncovered[row])
		for _, arc := range p.Rows[row] {
			if used[arc.Col] {
				continue
			}
			used[arc.Col] = true
			walk(row+1, total+arc.Cost)
			used[arc.Col] = false
		}
	}
	walk(0, 0)
	return best
}

func randomProblem(rng *rand.Rand, rows, cols int, density float64) assignmentsolver.Problem {
	p := assignmentsolver.Problem{Cols: cols, Rows: make([][]assignmentsolver.Arc, rows)}
	for i := range rows {
		for j := range cols {
			if rng.Float64() < density {
				p.Rows[i] = append(
					p.Rows[i],
					assignmentsolver.Arc{Col: j, Cost: int64(rng.Intn(50))},
				)
			}
		}
		p.Uncovered = append(p.Uncovered, int64(20+rng.Intn(60)))
	}
	return p
}

func TestSolve_MatchesExhaustiveSearchOnRandomBoards(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(11))
	for trial := range 400 {
		rows, cols := 1+rng.Intn(6), 1+rng.Intn(6)
		p := randomProblem(rng, rows, cols, 0.2+rng.Float64()*0.8)

		result := assignmentsolver.Solve(p)

		var total int64
		seen := map[int]bool{}
		for row, col := range result.RowAssignment {
			if col == assignmentsolver.Unassigned {
				total += p.Uncovered[row]
				continue
			}
			require.False(t, seen[col], "trial %d gave column %d twice", trial, col)
			seen[col] = true
			require.Equal(t, row, result.ColAssignment[col])
			found := false
			for _, arc := range p.Rows[row] {
				if arc.Col == col {
					found = true
				}
			}
			require.True(
				t,
				found,
				"trial %d paired row %d with column %d it has no arc to",
				trial,
				row,
				col,
			)
		}
		for row, col := range result.RowAssignment {
			if col != assignmentsolver.Unassigned {
				for _, arc := range p.Rows[row] {
					if arc.Col == col {
						total += arc.Cost
					}
				}
			}
		}

		require.Equal(t, bruteForce(p), total, "trial %d (%dx%d)", trial, rows, cols)
		require.Equal(t, total, result.Objective, "trial %d objective", trial)
	}
}

var benchSizes = [][3]int{{400, 3000, 40}, {1000, 3000, 40}, {2000, 3000, 40}, {5000, 8000, 60}, {5000, 5000, 60}}

func candidateGraph(rows, cols, degree int) assignmentsolver.Problem {
	return candidateGraphPriced(rows, cols, degree, 1_000_000)
}

func candidateGraphPriced(rows, cols, degree int, uncovered int64) assignmentsolver.Problem {
	rng := rand.New(rand.NewSource(7))
	p := assignmentsolver.Problem{Cols: cols, Rows: make([][]assignmentsolver.Arc, rows)}
	for i := range rows {
		for _, j := range rng.Perm(cols)[:degree] {
			p.Rows[i] = append(
				p.Rows[i],
				assignmentsolver.Arc{Col: j, Cost: int64(rng.Intn(100_000))},
			)
		}
		p.Uncovered = append(p.Uncovered, uncovered)
	}

	return p
}

func BenchmarkSolve_CandidateGraph(b *testing.B) {
	for _, size := range benchSizes {
		rows, cols, degree := size[0], size[1], size[2]
		b.Run(fmt.Sprintf("%dx%d_k=%d", rows, cols, degree), func(b *testing.B) {
			p := candidateGraph(rows, cols, degree)
			for b.Loop() {
				assignmentsolver.Solve(p)
			}
		})
	}
}

func BenchmarkPrices_EveryAssignedDriverAndSixOverrides(b *testing.B) {
	for _, size := range benchSizes {
		rows, cols, degree := size[0], size[1], size[2]
		b.Run(fmt.Sprintf("%dx%d_k=%d", rows, cols, degree), func(b *testing.B) {
			p := candidateGraphPriced(rows, cols, degree, 80_000)
			solution := assignmentsolver.SolveWithPrices(p)
			unknown := 0
			for b.Loop() {
				solution.ColumnValues()
				work := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
				var missing atomic.Int64
				for row, col := range solution.RowAssignment {
					if col == assignmentsolver.Unassigned {
						continue
					}
					work.Go(func() {
						for _, arc := range p.Rows[row][:6] {
							if _, ok := solution.OverrideCost(row, arc.Col); !ok {
								missing.Add(1)
							}
						}
					})
				}
				work.Wait()
				unknown = int(missing.Load())
			}
			b.ReportMetric(float64(unknown), "unknown_overrides")
		})
	}
}

func forcedOptimum(p assignmentsolver.Problem, row, col int, cost int64) int64 {
	reduced := assignmentsolver.Problem{Cols: p.Cols, Uncovered: make([]int64, 0, len(p.Rows)-1)}
	for i, arcs := range p.Rows {
		if i == row {
			continue
		}
		kept := make([]assignmentsolver.Arc, 0, len(arcs))
		for _, arc := range arcs {
			if arc.Col != col {
				kept = append(kept, arc)
			}
		}
		reduced.Rows = append(reduced.Rows, kept)
		reduced.Uncovered = append(reduced.Uncovered, p.Uncovered[i])
	}

	return bruteForce(reduced) + cost
}

func TestOverrideCost_IsWhatForcingThatPairCostsTheWholeBoard(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(23))
	checked := 0
	for trial := range 2000 {
		rows, cols := 2+rng.Intn(5), 2+rng.Intn(5)
		p := randomProblem(rng, rows, cols, 0.3+rng.Float64()*0.7)
		solution := assignmentsolver.SolveWithPrices(p)

		for row, arcs := range p.Rows {
			for _, arc := range arcs {
				got, ok := solution.OverrideCost(row, arc.Col)
				require.True(t, ok, "trial %d: row %d col %d", trial, row, arc.Col)
				want := forcedOptimum(p, row, arc.Col, arc.Cost) - solution.Objective
				require.Equal(
					t,
					want,
					got,
					"trial %d: forcing row %d onto col %d",
					trial,
					row,
					arc.Col,
				)
				checked++
			}
		}
	}
	assert.Positive(t, checked)
}

func TestOverrideCost_RefusesAPairThatHasNoArc(t *testing.T) {
	t.Parallel()

	solution := assignmentsolver.SolveWithPrices(dense([][]int64{{1, -1}}, never))

	_, ok := solution.OverrideCost(0, 1)
	assert.False(t, ok)
}

func TestColumnValue_IsWhatADriverIsWorthToThePlan(t *testing.T) {
	t.Parallel()

	p := dense([][]int64{{10, 40}, {20, -1}}, 0)
	p.Uncovered = []int64{100, 100}
	solution := assignmentsolver.SolveWithPrices(p)

	require.Equal(t, []int{1, 0}, solution.RowAssignment)
	assert.Equal(t, int64(80), solution.ColumnValue(0),
		"without the first driver the second move goes uncovered: 100 - 20")
	assert.Equal(t, int64(60), solution.Objective)
	assert.Equal(
		t,
		int64(50),
		solution.ColumnValue(1),
		"without the second driver the first move takes the first driver and the second goes uncovered",
	)
}

func TestColumnValue_MatchesRemovingTheDriverOnRandomBoards(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(31))
	for trial := range 300 {
		p := randomProblem(rng, 2+rng.Intn(4), 2+rng.Intn(4), 0.6)
		solution := assignmentsolver.SolveWithPrices(p)

		for col := range p.Cols {
			without := assignmentsolver.Problem{Cols: p.Cols, Uncovered: p.Uncovered}
			for _, arcs := range p.Rows {
				kept := make([]assignmentsolver.Arc, 0, len(arcs))
				for _, arc := range arcs {
					if arc.Col != col {
						kept = append(kept, arc)
					}
				}
				without.Rows = append(without.Rows, kept)
			}

			require.Equal(
				t,
				bruteForce(without)-solution.Objective,
				solution.ColumnValue(col),
				"trial %d: removing column %d", trial, col,
			)
		}
	}
}

func TestColumnValues_MatchEachDriversOwnReroute(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(11))
	for board := range 3000 {
		rows, cols := 1+rng.Intn(7), 1+rng.Intn(7)
		p := assignmentsolver.Problem{Cols: cols, Rows: make([][]assignmentsolver.Arc, rows)}
		for i := range rows {
			for _, j := range rng.Perm(cols)[:1+rng.Intn(cols)] {
				p.Rows[i] = append(p.Rows[i], assignmentsolver.Arc{Col: j, Cost: int64(rng.Intn(50))})
			}
			p.Uncovered = append(p.Uncovered, int64(rng.Intn(80)))
		}

		solution := assignmentsolver.SolveWithPrices(p)
		values := solution.ColumnValues()
		require.Len(t, values, cols)
		for col := range cols {
			require.Equal(t, solution.ColumnValue(col), values[col], "board %d column %d", board, col)
		}
	}
}
