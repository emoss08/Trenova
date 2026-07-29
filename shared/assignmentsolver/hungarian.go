package assignmentsolver

import "math"

const Unassigned = -1

const Forbidden = math.MaxFloat64 / 1e6

type Result struct {
	RowAssignment []int
	ColAssignment []int
	TotalCost     float64
}

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
	u := make([]float64, rows+1)
	v := make([]float64, cols+1)
	colToRow := make([]int, cols+1)
	way := make([]int, cols+1)
	minDelta := make([]float64, cols+1)
	used := make([]bool, cols+1)

	for i := 1; i <= rows; i++ {
		colToRow[0] = i
		for j := range minDelta {
			minDelta[j] = math.Inf(1)
			used[j] = false
		}

		j0 := 0
		stuck := false
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
				stuck = true
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

		if stuck || j0 == 0 {
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
