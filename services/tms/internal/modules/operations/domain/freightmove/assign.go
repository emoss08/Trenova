package freightmove

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
)

const stayReward = int64(1_000_000_000_000)

type seat struct {
	target    *target
	departure pulid.ID
}

type board struct {
	options []option
	seats   []seat
	movers  []pulid.ID
	column  map[pulid.ID]int
	banned  map[int]bool
}

func (p *planner) assign(options []option) []option {
	b := &board{options: leadingOptions(options), banned: map[int]bool{}}
	b.seat()
	chosen := b.solve(p)
	for range MaxRepairs {
		clash, ok := conflict(b.options, chosen)
		if !ok {
			break
		}
		b.banned[clash] = true
		chosen = b.solve(p)
	}
	for {
		clash, ok := conflict(b.options, chosen)
		if !ok {
			break
		}
		chosen = slices.DeleteFunc(chosen, func(index int) bool { return index == clash })
	}

	out := make([]option, 0, len(chosen))
	for _, index := range chosen {
		out = append(out, b.options[index])
	}
	slices.SortStableFunc(out, byNet)

	return out
}

func byNet(a, b option) int {
	if c := cmp.Compare(b.step.NetCents, a.step.NetCents); c != 0 {
		return c
	}
	if c := cmp.Compare(a.worker, b.worker); c != 0 {
		return c
	}

	return cmp.Compare(a.partner, b.partner)
}

func leadingOptions(options []option) []option {
	slices.SortStableFunc(options, byNet)
	taken := map[*target]int{}
	out := make([]option, 0, len(options))
	for _, opt := range options {
		if taken[opt.target] >= opt.target.room+MaxTargetOptions {
			continue
		}
		taken[opt.target]++
		out = append(out, opt)
	}

	return out
}

func (b *board) seat() {
	b.column = map[pulid.ID]int{}
	departures := map[*target][]pulid.ID{}
	targets := []*target{}
	seen := map[*target]bool{}
	for i := range b.options {
		opt := &b.options[i]
		if _, ok := b.column[opt.mover()]; !ok {
			b.column[opt.mover()] = len(b.movers)
			b.movers = append(b.movers, opt.mover())
		}
		if !seen[opt.target] {
			seen[opt.target] = true
			targets = append(targets, opt.target)
		}
		if !opt.departure.IsNil() && !slices.Contains(departures[opt.target], opt.departure) {
			departures[opt.target] = append(departures[opt.target], opt.departure)
		}
	}
	for _, t := range targets {
		kept := departures[t]
		for index := range t.room {
			s := seat{target: t}
			if index < len(kept) {
				s.departure = kept[index]
			}
			b.seats = append(b.seats, s)
		}
	}
}

func (b *board) solve(p *planner) []int {
	rows := make([][]assignmentsolver.Arc, len(b.seats))
	picked := make([]map[int]int, len(b.seats))
	for row, s := range b.seats {
		best := map[int]int{}
		for index := range b.options {
			opt := &b.options[index]
			if b.banned[index] || opt.target != s.target ||
				(!opt.departure.IsNil() && opt.departure != s.departure) {
				continue
			}
			col := b.column[opt.mover()]
			if current, ok := best[col]; !ok ||
				opt.step.NetCents > b.options[current].step.NetCents {
				best[col] = index
			}
		}
		picked[row] = best
		for col, index := range best {
			rows[row] = append(rows[row], assignmentsolver.Arc{
				Col:  col,
				Cost: -b.options[index].step.NetCents,
			})
		}
		slices.SortFunc(rows[row], func(a, c assignmentsolver.Arc) int {
			return cmp.Compare(a.Col, c.Col)
		})
	}
	rows = append(rows, b.stays(p)...)

	result := assignmentsolver.Solve(assignmentsolver.Problem{Cols: len(b.movers), Rows: rows})
	chosen := []int{}
	for row := range b.seats {
		if col := result.RowAssignment[row]; col != assignmentsolver.Unassigned {
			chosen = append(chosen, picked[row][col])
		}
	}

	return chosen
}

func (b *board) stays(p *planner) [][]assignmentsolver.Arc {
	type origin struct {
		cols  []int
		spare int
	}
	origins := map[int64]*origin{}
	cells := []int64{}
	for col, mover := range b.movers {
		d := p.drivers[mover]
		o := origins[d.Cell]
		if o == nil {
			o = &origin{}
			origins[d.Cell] = o
			cells = append(cells, d.Cell)
		}
		o.cols = append(o.cols, col)
		o.spare = max(o.spare, p.spare(d))
	}
	slices.Sort(cells)

	rows := [][]assignmentsolver.Arc{}
	for _, cell := range cells {
		o := origins[cell]
		for range len(o.cols) - o.spare {
			arcs := make([]assignmentsolver.Arc, 0, len(o.cols))
			for _, col := range o.cols {
				arcs = append(arcs, assignmentsolver.Arc{Col: col, Cost: -stayReward})
			}
			rows = append(rows, arcs)
		}
	}

	return rows
}

func conflict(options []option, chosen []int) (int, bool) {
	moving := map[pulid.ID]int{}
	for _, index := range chosen {
		moving[options[index].mover()] = index
	}
	clash, found := 0, false
	for _, index := range chosen {
		opt := &options[index]
		if opt.partner.IsNil() {
			continue
		}
		other, ok := moving[opt.worker]
		if !ok {
			continue
		}
		weaker := index
		if options[other].step.NetCents < opt.step.NetCents {
			weaker = other
		}
		if !found || options[weaker].step.NetCents < options[clash].step.NetCents {
			clash, found = weaker, true
		}
	}

	return clash, found
}
