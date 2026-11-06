package assignmentsolver

import (
	"container/heap"
	"sync"
)

const Unassigned = -1

type Arc struct {
	Col  int
	Cost int64
}

type Problem struct {
	Cols      int
	Rows      [][]Arc
	Uncovered []int64
}

type Result struct {
	RowAssignment []int
	ColAssignment []int
	TotalCost     int64
	Objective     int64
}

type solver struct {
	rows      [][]Arc
	uncovered []int64
	cols      int
	total     int

	u           []int64
	v           []int64
	colOwner    []int
	rowCol      []int
	matchedCost []int64

	dist     []int64
	pred     []int
	predCost []int64
	stamp    []int
	done     []int
	epoch    int
	visited  []int
	frontier columnHeap

	byCol       [][]rowArc
	scratchPool sync.Pool
}

func Solve(p Problem) Result {
	return SolveWithPrices(p).Result
}

func newSolver(p Problem) *solver {
	rows := len(p.Rows)
	cols := max(p.Cols, 0)
	total := cols + rows

	s := &solver{
		rows:        p.Rows,
		uncovered:   p.Uncovered,
		cols:        cols,
		total:       total,
		u:           make([]int64, rows),
		v:           make([]int64, total),
		colOwner:    make([]int, total),
		rowCol:      make([]int, rows),
		matchedCost: make([]int64, rows),
		dist:        make([]int64, total),
		pred:        make([]int, total),
		predCost:    make([]int64, total),
		stamp:       make([]int, total),
		done:        make([]int, total),
		visited:     make([]int, 0, 64),
	}
	for j := range s.colOwner {
		s.colOwner[j] = Unassigned
	}
	for i := range s.rowCol {
		s.rowCol[i] = Unassigned
	}

	return s
}

func (s *solver) uncoveredCost(row int) int64 {
	if row < len(s.uncovered) {
		return s.uncovered[row]
	}

	return 0
}

func (s *solver) reduceRows() {
	for row, arcs := range s.rows {
		best := s.uncoveredCost(row)
		bestCol := s.cols + row
		for _, arc := range arcs {
			if !s.validCol(arc.Col) {
				continue
			}
			if arc.Cost < best || (arc.Cost == best && arc.Col < bestCol) {
				best = arc.Cost
				bestCol = arc.Col
			}
		}
		s.u[row] = best
		if s.colOwner[bestCol] == Unassigned {
			s.colOwner[bestCol] = row
			s.rowCol[row] = bestCol
			s.matchedCost[row] = best
		}
	}
}

func (s *solver) validCol(col int) bool {
	return col >= 0 && col < s.cols
}

func (s *solver) augment(start int) {
	s.epoch++
	s.visited = s.visited[:0]
	s.frontier = s.frontier[:0]

	s.relaxRow(start, 0)

	for s.frontier.Len() > 0 {
		top, ok := s.frontier.pop()
		if !ok {
			break
		}
		col := top.col
		if s.done[col] == s.epoch || top.dist != s.dist[col] {
			continue
		}
		s.done[col] = s.epoch
		s.visited = append(s.visited, col)

		owner := s.colOwner[col]
		if owner == Unassigned {
			s.finish(start, col)
			return
		}
		s.relaxRow(owner, s.dist[col])
	}
}

func (s *solver) reduced(row, col int, cost int64) int64 {
	return cost - s.u[row] - s.v[col]
}

func (s *solver) relaxRow(row int, base int64) {
	for _, arc := range s.rows[row] {
		if !s.validCol(arc.Col) {
			continue
		}
		s.relax(row, arc.Col, arc.Cost, base+s.reduced(row, arc.Col, arc.Cost))
	}
	dummy := s.cols + row
	cost := s.uncoveredCost(row)
	s.relax(row, dummy, cost, base+s.reduced(row, dummy, cost))
}

func (s *solver) relax(row, col int, cost, candidate int64) {
	if s.done[col] == s.epoch {
		return
	}
	if s.stamp[col] == s.epoch && candidate >= s.dist[col] {
		return
	}
	s.stamp[col] = s.epoch
	s.dist[col] = candidate
	s.pred[col] = row
	s.predCost[col] = cost
	heap.Push(&s.frontier, columnEntry{col: col, dist: candidate})
}

func (s *solver) finish(start, end int) {
	final := s.dist[end]

	for _, col := range s.visited {
		if col == end {
			continue
		}
		s.v[col] += s.dist[col] - final
	}
	for _, col := range s.visited {
		if owner := s.colOwner[col]; owner != Unassigned && col != end {
			s.u[owner] = s.matchedCost[owner] - s.v[col]
		}
	}

	col := end
	for {
		row := s.pred[col]
		previous := s.rowCol[row]
		s.rowCol[row] = col
		s.colOwner[col] = row
		s.matchedCost[row] = s.predCost[col]
		if row == start {
			break
		}
		col = previous
	}
	s.u[start] = s.matchedCost[start] - s.v[s.rowCol[start]]
}

type columnEntry struct {
	col  int
	dist int64
}

type columnHeap []columnEntry

func (h columnHeap) Len() int { return len(h) }

func (h columnHeap) Less(i, j int) bool {
	if h[i].dist != h[j].dist {
		return h[i].dist < h[j].dist
	}
	return h[i].col < h[j].col
}

func (h columnHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *columnHeap) Push(x any) {
	if entry, ok := x.(columnEntry); ok {
		*h = append(*h, entry)
	}
}

func (h *columnHeap) pop() (columnEntry, bool) {
	entry, ok := heap.Pop(h).(columnEntry)
	return entry, ok
}

func (h *columnHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

type Solution struct {
	Result

	s *solver
}

func SolveWithPrices(p Problem) Solution {
	rows := len(p.Rows)
	s := newSolver(p)
	s.reduceRows()
	for row := range rows {
		if s.rowCol[row] == Unassigned {
			s.augment(row)
		}
	}

	s.indexColumns()

	return Solution{Result: s.result(), s: s}
}

func (s *solver) result() Result {
	result := Result{
		RowAssignment: make([]int, len(s.rowCol)),
		ColAssignment: make([]int, s.cols),
	}
	for j := range result.ColAssignment {
		result.ColAssignment[j] = Unassigned
	}
	for row, col := range s.rowCol {
		result.RowAssignment[row] = Unassigned
		result.Objective += s.matchedCost[row]
		if col >= s.cols {
			continue
		}
		result.RowAssignment[row] = col
		result.ColAssignment[col] = row
		result.TotalCost += s.matchedCost[row]
	}

	return result
}

func arcTo(arcs []Arc, col int) (int64, bool) {
	for _, arc := range arcs {
		if arc.Col == col {
			return arc.Cost, true
		}
	}

	return 0, false
}

func (s *solver) indexColumns() {
	s.byCol = make([][]rowArc, s.cols)
	for row, arcs := range s.rows {
		for _, arc := range arcs {
			if s.validCol(arc.Col) {
				s.byCol[arc.Col] = append(s.byCol[arc.Col], rowArc{row: row, cost: arc.Cost})
			}
		}
	}
}

type rowArc struct {
	row  int
	cost int64
}
