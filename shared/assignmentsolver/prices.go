package assignmentsolver

import "container/heap"

const (
	noPath = int64(-1)

	overrideSettleBudget = 4096
)

type pathScratch struct {
	dist     []int64
	seen     []uint32
	done     []uint32
	epoch    uint32
	frontier columnHeap
}

func (s *solver) scratch() *pathScratch {
	if pooled, ok := s.scratchPool.Get().(*pathScratch); ok && len(pooled.dist) == s.total {
		return pooled
	}

	return &pathScratch{
		dist: make([]int64, s.total),
		seen: make([]uint32, s.total),
		done: make([]uint32, s.total),
	}
}

func (p *pathScratch) reset() {
	p.epoch++
	if p.epoch == 0 {
		clear(p.seen)
		clear(p.done)
		p.epoch = 1
	}
	p.frontier = p.frontier[:0]
}

func (p *pathScratch) offer(col int, d int64) {
	if p.done[col] == p.epoch || (p.seen[col] == p.epoch && d >= p.dist[col]) {
		return
	}
	p.seen[col] = p.epoch
	p.dist[col] = d
	heap.Push(&p.frontier, columnEntry{col: col, dist: d})
}

func (p *pathScratch) next() (columnEntry, bool) {
	for p.frontier.Len() > 0 {
		top, ok := p.frontier.pop()
		if !ok {
			return columnEntry{}, false
		}
		if p.done[top.col] == p.epoch || top.dist != p.dist[top.col] {
			continue
		}
		p.done[top.col] = p.epoch

		return top, true
	}

	return columnEntry{}, false
}

func (sol Solution) ColumnValue(col int) int64 {
	s := sol.s
	if s == nil || !s.validCol(col) || s.colOwner[col] == Unassigned {
		return 0
	}
	path := s.cheapestPath(&pathQuery{
		start:    s.colOwner[col],
		fixedRow: Unassigned,
		takenCol: col,
		freedCol: Unassigned,
	})
	if path < 0 {
		return 0
	}

	return path - s.v[col]
}

func (sol Solution) OverrideCost(row, col int) (int64, bool) {
	s := sol.s
	if s == nil || row < 0 || row >= len(s.rows) || !s.validCol(col) {
		return 0, false
	}
	cost, ok := arcTo(s.rows[row], col)
	if !ok {
		return 0, false
	}
	freed := s.rowCol[row]
	if freed == col {
		return 0, true
	}

	reduced := cost - s.u[row] - s.v[col]
	owner := s.colOwner[col]
	release, known := s.releaseValue(freed, row, owner, col)
	if !known {
		return 0, false
	}
	if owner == Unassigned {
		return reduced + release, true
	}

	path := s.cheapestPath(&pathQuery{
		start:    owner,
		fixedRow: row,
		takenCol: col,
		freedCol: freed,
		release:  release,
		budget:   overrideSettleBudget,
	})
	if path < 0 {
		return 0, false
	}

	return reduced + path, true
}

type pathQuery struct {
	start    int
	fixedRow int
	takenCol int
	freedCol int
	release  int64
	budget   int
}

func (s *solver) cheapestPath(q *pathQuery) int64 {
	p := s.scratch()
	defer s.scratchPool.Put(p)
	p.reset()

	relax := func(row int, base int64) {
		for _, arc := range s.rows[row] {
			if s.validCol(arc.Col) && arc.Col != q.takenCol {
				p.offer(arc.Col, base+arc.Cost-s.u[row]-s.v[arc.Col])
			}
		}
		dummy := s.cols + row
		p.offer(dummy, base+s.uncoveredCost(row)-s.u[row]-s.v[dummy])
	}

	best := noPath
	slack := min(q.release, 0)
	relax(q.start, 0)
	for settled := 1; ; settled++ {
		top, ok := p.next()
		if !ok || (best != noPath && best <= top.dist+slack) {
			return best
		}
		if q.budget > 0 && settled > q.budget {
			return noPath
		}

		owner := s.colOwner[top.col]
		switch {
		case top.col == q.freedCol:
			best = cheaper(best, top.dist)
		case owner == Unassigned:
			best = cheaper(best, top.dist+q.release)
		case owner != q.fixedRow:
			relax(owner, top.dist)
		}
	}
}

func (s *solver) releaseValue(freed, fixedRow, movingRow, takenCol int) (int64, bool) {
	best := -s.v[freed]
	if freed >= s.cols {
		return best, true
	}

	p := s.scratch()
	defer s.scratchPool.Put(p)
	p.reset()
	p.offer(freed, 0)
	for settled := 1; ; settled++ {
		top, ok := p.next()
		if !ok || top.dist >= best {
			return best, true
		}
		if settled > overrideSettleBudget {
			return 0, false
		}
		if top.col != freed {
			best = min(best, top.dist-s.v[top.col])
		}
		if top.col >= s.cols {
			continue
		}

		for _, taker := range s.byCol[top.col] {
			if taker.row == fixedRow || taker.row == movingRow {
				continue
			}
			held := s.rowCol[taker.row]
			if held == takenCol || held == Unassigned {
				continue
			}
			p.offer(held, top.dist+taker.cost-s.u[taker.row]-s.v[top.col])
		}
	}
}

func cheaper(best, candidate int64) int64 {
	if best == noPath || candidate < best {
		return candidate
	}

	return best
}

type rowEntry struct {
	row  int
	dist int64
}

type rowHeap []rowEntry

func (h rowHeap) Len() int { return len(h) }

func (h rowHeap) Less(i, j int) bool {
	if h[i].dist != h[j].dist {
		return h[i].dist < h[j].dist
	}
	return h[i].row < h[j].row
}

func (h rowHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *rowHeap) Push(x any) {
	if entry, ok := x.(rowEntry); ok {
		*h = append(*h, entry)
	}
}

func (h *rowHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

func (sol Solution) ColumnValues() []int64 {
	s := sol.s
	if s == nil {
		return nil
	}
	values := make([]int64, s.cols)
	exit := s.exitDistances()
	for col := range s.cols {
		owner := s.colOwner[col]
		if owner == Unassigned || exit[owner] < 0 {
			continue
		}
		values[col] = max(0, exit[owner]-s.v[col])
	}

	return values
}

func (s *solver) exitDistances() []int64 {
	rows := len(s.rows)
	dist := make([]int64, rows)
	done := make([]bool, rows)
	frontier := make(rowHeap, 0, rows)
	for row := range rows {
		dist[row] = noPath
		if held := s.rowCol[row]; held == Unassigned || held >= s.cols {
			continue
		}
		best := s.uncoveredCost(row) - s.u[row] - s.v[s.cols+row]
		for _, arc := range s.rows[row] {
			if s.validCol(arc.Col) && s.colOwner[arc.Col] == Unassigned {
				best = min(best, arc.Cost-s.u[row]-s.v[arc.Col])
			}
		}
		dist[row] = best
		frontier = append(frontier, rowEntry{row: row, dist: best})
	}
	heap.Init(&frontier)

	for frontier.Len() > 0 {
		top, ok := heap.Pop(&frontier).(rowEntry)
		if !ok {
			break
		}
		if done[top.row] || top.dist != dist[top.row] {
			continue
		}
		done[top.row] = true

		held := s.rowCol[top.row]
		for _, taker := range s.byCol[held] {
			if taker.row == top.row || done[taker.row] {
				continue
			}
			if via := s.rowCol[taker.row]; via == Unassigned || via >= s.cols {
				continue
			}
			candidate := top.dist + taker.cost - s.u[taker.row] - s.v[held]
			if dist[taker.row] == noPath || candidate < dist[taker.row] {
				dist[taker.row] = candidate
				heap.Push(&frontier, rowEntry{row: taker.row, dist: candidate})
			}
		}
	}

	return dist
}
