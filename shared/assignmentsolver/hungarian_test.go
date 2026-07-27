package assignmentsolver_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolve_SquareMatrixFindsTheOptimum(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{4, 1, 3},
		{2, 0, 5},
		{3, 2, 2},
	}

	result := assignmentsolver.Solve(cost)

	assert.InDelta(t, 5.0, result.TotalCost, 0.0001)
	assertValidAssignment(t, result, 3, 3)
}

// The whole reason for a global solver: a greedy pass takes the single cheapest cell and
// strands the rest. Here greedy would take (0,0)=1 then be forced into (1,1)=100 for a
// total of 101; the optimum is 2+9=11.
func TestSolve_BeatsGreedy(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{1, 2},
		{9, 100},
	}

	result := assignmentsolver.Solve(cost)

	assert.InDelta(t, 11.0, result.TotalCost, 0.0001)
	assert.Equal(t, 1, result.RowAssignment[0])
	assert.Equal(t, 0, result.RowAssignment[1])
}

func TestSolve_MoreColumnsThanRows(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{5, 1, 9, 3},
		{2, 8, 4, 7},
	}

	result := assignmentsolver.Solve(cost)

	require.Len(t, result.RowAssignment, 2)
	require.Len(t, result.ColAssignment, 4)
	assert.InDelta(t, 3.0, result.TotalCost, 0.0001)
	assertValidAssignment(t, result, 2, 4)
}

func TestSolve_MoreRowsThanColumns(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{5, 1},
		{2, 8},
		{9, 4},
	}

	result := assignmentsolver.Solve(cost)

	require.Len(t, result.RowAssignment, 3)
	require.Len(t, result.ColAssignment, 2)
	assert.InDelta(t, 3.0, result.TotalCost, 0.0001)

	unassigned := 0
	for _, col := range result.RowAssignment {
		if col == assignmentsolver.Unassigned {
			unassigned++
		}
	}
	assert.Equal(t, 1, unassigned, "one row must go unmatched when columns run out")
	assertValidAssignment(t, result, 3, 2)
}

// A forbidden pairing describes something that cannot legally happen — a driver who is
// out of hours, say. The solver must leave both sides free rather than commit to it.
func TestSolve_NeverChoosesAForbiddenPair(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{assignmentsolver.Forbidden, 4},
		{assignmentsolver.Forbidden, assignmentsolver.Forbidden},
	}

	result := assignmentsolver.Solve(cost)

	assert.Equal(t, 1, result.RowAssignment[0])
	assert.Equal(t, assignmentsolver.Unassigned, result.RowAssignment[1])
	assert.Equal(t, assignmentsolver.Unassigned, result.ColAssignment[0])
	assert.InDelta(t, 4.0, result.TotalCost, 0.0001)
}

func TestSolve_AllPairsForbiddenAssignsNothing(t *testing.T) {
	t.Parallel()

	cost := [][]float64{
		{assignmentsolver.Forbidden, assignmentsolver.Forbidden},
		{assignmentsolver.Forbidden, assignmentsolver.Forbidden},
	}

	result := assignmentsolver.Solve(cost)

	for _, col := range result.RowAssignment {
		assert.Equal(t, assignmentsolver.Unassigned, col)
	}
	assert.InDelta(t, 0.0, result.TotalCost, 0.0001)
}

func TestSolve_DegenerateInputs(t *testing.T) {
	t.Parallel()

	assert.Empty(t, assignmentsolver.Solve(nil).RowAssignment)
	assert.Empty(t, assignmentsolver.Solve([][]float64{}).RowAssignment)
	assert.Empty(t, assignmentsolver.Solve([][]float64{{}}).RowAssignment)

	ragged := assignmentsolver.Solve([][]float64{{1, 2}, {3}})
	assert.Empty(t, ragged.RowAssignment, "a ragged matrix yields nothing rather than panicking")
}

func TestSolve_SingleCell(t *testing.T) {
	t.Parallel()

	result := assignmentsolver.Solve([][]float64{{7}})

	assert.Equal(t, 0, result.RowAssignment[0])
	assert.InDelta(t, 7.0, result.TotalCost, 0.0001)
}

// Cross-checks the solver against exhaustive search on inputs small enough to enumerate.
func TestSolve_MatchesBruteForceOnSmallMatrices(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(20260812))

	for trial := range 200 {
		rows := 1 + rng.Intn(5)
		cols := 1 + rng.Intn(5)

		cost := make([][]float64, rows)
		for i := range cost {
			cost[i] = make([]float64, cols)
			for j := range cost[i] {
				cost[i][j] = float64(rng.Intn(50))
			}
		}

		result := assignmentsolver.Solve(cost)
		expected := bruteForceMinCost(cost, rows, cols)

		assert.InDeltaf(t, expected, result.TotalCost, 0.0001,
			"trial %d: %dx%d matrix %v", trial, rows, cols, cost)
		assertValidAssignment(t, result, rows, cols)
	}
}

// bruteForceMinCost enumerates every way of matching min(rows, cols) pairs and returns the
// cheapest total. Only usable for tiny matrices, which is the point.
func bruteForceMinCost(cost [][]float64, rows, cols int) float64 {
	best := math.Inf(1)
	usedCols := make([]bool, cols)

	var walk func(row int, total float64, matched int)
	walk = func(row int, total float64, matched int) {
		if row == rows {
			if matched == min(rows, cols) && total < best {
				best = total
			}
			return
		}
		for j := range cols {
			if usedCols[j] {
				continue
			}
			usedCols[j] = true
			walk(row+1, total+cost[row][j], matched+1)
			usedCols[j] = false
		}
		// Skipping a row is only legal when there are not enough columns for every row.
		if rows > cols {
			walk(row+1, total, matched)
		}
	}

	walk(0, 0, 0)
	return best
}

func assertValidAssignment(t *testing.T, result assignmentsolver.Result, rows, cols int) {
	t.Helper()

	seenCols := make(map[int]int, cols)
	for row, col := range result.RowAssignment {
		if col == assignmentsolver.Unassigned {
			continue
		}
		require.GreaterOrEqual(t, col, 0)
		require.Less(t, col, cols)

		previous, duplicate := seenCols[col]
		require.Falsef(t, duplicate, "column %d matched to both row %d and row %d",
			col, previous, row)
		seenCols[col] = row

		require.Equalf(t, row, result.ColAssignment[col],
			"row and column assignments disagree for column %d", col)
	}
	require.Len(t, result.RowAssignment, rows)
	require.Len(t, result.ColAssignment, cols)
}
