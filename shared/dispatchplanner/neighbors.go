package dispatchplanner

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/assignmentsolver"
)

type neighborhood struct {
	arcs  [][]assignmentsolver.Arc
	cols  [][]int
	preds [][]int
	limit int
	stamp []int
	epoch int
}

type rankedTask struct {
	task int
	rank int
}

func newNeighborhood(p *Problem, limit int) *neighborhood {
	tasks := len(p.Arcs)
	h := &neighborhood{
		arcs:  make([][]assignmentsolver.Arc, tasks),
		cols:  make([][]int, tasks),
		preds: make([][]int, tasks),
		limit: limit,
		stamp: make([]int, p.Resources),
	}

	for task, arcs := range p.Arcs {
		valid := make([]assignmentsolver.Arc, 0, len(arcs))
		for _, arc := range arcs {
			if arc.Col >= 0 && arc.Col < p.Resources {
				valid = append(valid, arc)
			}
		}
		slices.SortStableFunc(valid, func(a, b assignmentsolver.Arc) int {
			if c := cmp.Compare(a.Cost, b.Cost); c != 0 {
				return c
			}
			return cmp.Compare(a.Col, b.Col)
		})
		h.arcs[task] = valid
		h.cols[task] = make([]int, len(valid))
		for i, arc := range valid {
			h.cols[task][i] = arc.Col
		}
		slices.Sort(h.cols[task])
	}

	ranked := make([][]rankedTask, tasks)
	for from, next := range p.Successors {
		for rank, task := range next {
			if task < 0 || task >= tasks || task == from {
				continue
			}
			ranked[task] = append(ranked[task], rankedTask{task: from, rank: rank})
		}
	}
	for task, preds := range ranked {
		slices.SortStableFunc(preds, func(a, b rankedTask) int {
			if c := cmp.Compare(a.rank, b.rank); c != 0 {
				return c
			}
			return cmp.Compare(a.task, b.task)
		})
		kept := min(len(preds), limit)
		h.preds[task] = make([]int, kept)
		for i := range kept {
			h.preds[task][i] = preds[i].task
		}
	}

	return h
}

func (h *neighborhood) starts(task, resource int) bool {
	_, found := slices.BinarySearch(h.cols[task], resource)
	return found
}

func (h *neighborhood) nearest(task int) []assignmentsolver.Arc {
	arcs := h.arcs[task]
	return arcs[:min(len(arcs), h.limit)]
}

func (h *neighborhood) predecessors(task int) []int {
	return h.preds[task]
}

func (h *neighborhood) resources(task int, owner []int, incumbent int, dst []int) []int {
	h.epoch++
	for _, arc := range h.nearest(task) {
		dst = h.add(dst, arc.Col)
	}
	for _, pred := range h.preds[task] {
		dst = h.add(dst, owner[pred])
	}

	return h.add(dst, incumbent)
}

func (h *neighborhood) add(dst []int, resource int) []int {
	if resource < 0 || resource >= len(h.stamp) || h.stamp[resource] == h.epoch {
		return dst
	}
	h.stamp[resource] = h.epoch

	return append(dst, resource)
}

func (h *neighborhood) reaches(task, resource int, owner []int, incumbent int) bool {
	if resource == incumbent {
		return true
	}
	for _, arc := range h.nearest(task) {
		if arc.Col == resource {
			return true
		}
	}
	for _, pred := range h.preds[task] {
		if owner[pred] == resource {
			return true
		}
	}

	return false
}
