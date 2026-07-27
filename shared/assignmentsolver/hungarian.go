// Package assignmentsolver solves the rectangular linear assignment problem: given a
// cost for pairing each agent with each task, choose the pairing set that minimizes total
// cost with at most one task per agent and one agent per task.
//
// Dispatch uses this to cover a board as a whole rather than one load at a time. Greedy
// coverage — repeatedly taking the best remaining pair — spends the strongest driver on
// whichever load happens to be considered first and leaves later loads uncoverable; the
// global optimum does not.
package assignmentsolver

import "math"

// Unassigned marks a row or column that the solution left without a partner, which
// happens whenever the matrix is rectangular or entries are forbidden.
const Unassigned = -1

// Forbidden is the cost of a pairing that must never be chosen. It is large enough to
// dominate any real cost yet far from overflow, so summing a full row stays finite.
const Forbidden = math.MaxFloat64 / 1e6

// Result is the outcome of a solve.
type Result struct {
	// RowAssignment[i] is the column matched to row i, or Unassigned.
	RowAssignment []int
	// ColAssignment[j] is the row matched to column j, or Unassigned.
	ColAssignment []int
	// TotalCost is the summed cost of the chosen pairs, excluding forbidden ones.
	TotalCost float64
}

// Solve finds the minimum-cost assignment for the given cost matrix using the
// Jonker-Volgenant shortest augmenting path method, which runs in O(n^3) and handles
// rectangular inputs directly.
//
// A cost of Forbidden (or greater) marks a pairing that may not be chosen; such pairs are
// left unassigned rather than being forced in. An empty or ragged matrix yields an empty
// result rather than an error, so callers can pass a filtered candidate set without
// pre-checking it.
func Solve(cost [][]float64) Result {
	rows := len(cost)
	if rows == 0 {
		return Result{RowAssignment: []int{}, ColAssignment: []int{}}
	}

	cols := len(cost[0])
	for i := range cost {
		if len(cost[i]) != cols {
			return Result{RowAssignment: []int{}, ColAssignment: []int{}}
		}
	}
	if cols == 0 {
		return Result{RowAssignment: make([]int, 0), ColAssignment: []int{}}
	}

	// The algorithm below assumes at least as many columns as rows. Transposing keeps one
	// implementation instead of two mirror-image ones.
	if rows > cols {
		transposed := transpose(cost)
		result := Solve(transposed)
		return Result{
			RowAssignment: result.ColAssignment,
			ColAssignment: result.RowAssignment,
			TotalCost:     result.TotalCost,
		}
	}

	return solveWide(cost, rows, cols)
}

//nolint:gocognit // the augmenting-path inner loop is a single cohesive algorithm
func solveWide(cost [][]float64, rows, cols int) Result {
	// u and v are the dual potentials; way records the alternating tree used to augment.
	u := make([]float64, rows+1)
	v := make([]float64, cols+1)
	// colToRow[j] is the row currently matched to column j, using 1-based row indices with
	// 0 meaning "free". The extra sentinel column simplifies the augmentation loop.
	colToRow := make([]int, cols+1)
	way := make([]int, cols+1)

	for i := 1; i <= rows; i++ {
		colToRow[0] = i
		minDelta := make([]float64, cols+1)
		used := make([]bool, cols+1)
		for j := range minDelta {
			minDelta[j] = math.Inf(1)
		}

		j0 := 0
		for {
			used[j0] = true
			i0 := colToRow[j0]
			delta := math.Inf(1)
			j1 := 0

			for j := 1; j <= cols; j++ {
				if used[j] {
					continue
				}
				current := cost[i0-1][j-1] - u[i0] - v[j]
				if current < minDelta[j] {
					minDelta[j] = current
					way[j] = j0
				}
				if minDelta[j] < delta {
					delta = minDelta[j]
					j1 = j
				}
			}

			if math.IsInf(delta, 1) {
				// No reachable column remains; this row stays unassigned.
				break
			}

			for j := 0; j <= cols; j++ {
				if used[j] {
					u[colToRow[j]] += delta
					v[j] -= delta
					continue
				}
				minDelta[j] -= delta
			}

			j0 = j1
			if colToRow[j0] == 0 {
				break
			}
		}

		if j0 == 0 {
			continue
		}

		for j0 != 0 {
			j1 := way[j0]
			colToRow[j0] = colToRow[j1]
			j0 = j1
		}
	}

	return buildResult(cost, colToRow, rows, cols)
}

func buildResult(cost [][]float64, colToRow []int, rows, cols int) Result {
	result := Result{
		RowAssignment: make([]int, rows),
		ColAssignment: make([]int, cols),
	}
	for i := range result.RowAssignment {
		result.RowAssignment[i] = Unassigned
	}
	for j := range result.ColAssignment {
		result.ColAssignment[j] = Unassigned
	}

	for j := 1; j <= cols; j++ {
		row := colToRow[j] - 1
		if row < 0 || row >= rows {
			continue
		}
		// A forbidden pair means the solver had nowhere better to go, not that the pair is
		// acceptable. Dropping it keeps both sides free rather than committing to a pairing
		// the caller declared impossible.
		if cost[row][j-1] >= Forbidden {
			continue
		}
		result.RowAssignment[row] = j - 1
		result.ColAssignment[j-1] = row
		result.TotalCost += cost[row][j-1]
	}

	return result
}

func transpose(matrix [][]float64) [][]float64 {
	rows := len(matrix)
	cols := len(matrix[0])

	out := make([][]float64, cols)
	for j := range out {
		out[j] = make([]float64, rows)
		for i := range matrix {
			out[j][i] = matrix[i][j]
		}
	}
	return out
}
