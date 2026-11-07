package dispatchplanner

import (
	"cmp"
	"context"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/emoss08/trenova/shared/assignmentsolver"
)

const seedMixer = uint64(0x9e3779b97f4a7c15)

type search struct {
	p       *Problem
	opt     Options
	tasks   int
	rng     *rand.Rand
	cache   *prefixCache
	hood    *neighborhood
	cur     *plan
	started time.Time
	stats   Stats

	bestChains    [][]int
	bestCost      []int64
	bestObjective int64
	curIsBest     bool
	dirty         []int
	isDirty       []bool

	trial   proposal
	choice  proposal
	cands   []int
	at      []int
	work    []int
	mark    []int
	markTo  []int
	touched []int
	epoch   int
	steps   []int64

	values       []int64
	alternatives [][]Alternative
}

func newSearch(p *Problem) *search {
	opt := p.Options.normalized()
	tasks := len(p.Arcs)
	//nolint:gosec // reproducible search, not cryptography
	rng := rand.New(rand.NewPCG(opt.RandomSeed, opt.RandomSeed^seedMixer))

	return &search{
		p:            p,
		opt:          opt,
		tasks:        tasks,
		rng:          rng,
		cache:        newPrefixCache(p.Timeline, opt.CacheNodes),
		hood:         newNeighborhood(p, opt.Neighbors),
		cur:          newPlan(tasks, p.Resources),
		started:      opt.Clock(),
		bestChains:   make([][]int, p.Resources),
		bestCost:     make([]int64, p.Resources),
		isDirty:      make([]bool, p.Resources),
		mark:         make([]int, tasks),
		markTo:       make([]int, tasks),
		values:       make([]int64, p.Resources),
		alternatives: make([][]Alternative, tasks),
	}
}

func (s *search) opens(task int) int64 {
	if s.p.Opens == nil {
		return 0
	}

	return s.p.Opens[task]
}

func (s *search) incumbent(task int) int {
	if s.p.Incumbent == nil {
		return Unassigned
	}
	if r := s.p.Incumbent[task]; r >= 0 && r < s.p.Resources {
		return r
	}

	return Unassigned
}

func (s *search) term(task, resource int) int64 {
	cents := int64(0)
	if resource == Unassigned {
		cents = s.p.Coverage[task]
	}
	if inc := s.incumbent(task); inc != Unassigned && inc != resource {
		cents += s.opt.StabilityCents
	}

	return cents
}

func (s *search) objective() int64 {
	total := int64(0)
	for _, cost := range s.cur.cost {
		total += cost
	}
	for task, owner := range s.cur.owner {
		total += s.term(task, owner)
	}

	return total
}

func (s *search) full(chain []int) bool {
	return s.opt.MaxPerResource > 0 && len(chain) >= s.opt.MaxPerResource
}

func (s *search) candidates(task int) []int {
	s.cands = s.hood.resources(task, s.cur.owner, s.incumbent(task), s.cands[:0])
	return s.cands
}

func (s *search) reaches(task, resource int) bool {
	return s.hood.reaches(task, resource, s.cur.owner, s.incumbent(task))
}

func (s *search) slots(task int, chain []int) []int {
	opens := s.opens(task)
	at := len(chain)
	for i, other := range chain {
		if s.opens(other) > opens {
			at = i
			break
		}
	}
	s.at = append(s.at[:0], at)

	preds := s.hood.predecessors(task)
	for i, other := range chain {
		if slices.Contains(preds, other) {
			if i+1 != at {
				s.at = append(s.at, i+1)
			}
			break
		}
	}

	return s.at
}

func (s *search) nearestIndex(chain []int, opens int64) int {
	best, gap := 0, int64(-1)
	for i, task := range chain {
		d := s.opens(task) - opens
		if d < 0 {
			d = -d
		}
		if gap < 0 || d < gap {
			best, gap = i, d
		}
	}

	return best
}

func (s *search) score(pr *proposal) bool {
	pr.valid = false
	delta := int64(0)
	for i := range pr.n {
		c := &pr.changes[i]
		cost, ok := s.chainCost(c.resource, c.chain)
		if !ok {
			return false
		}
		c.cost = cost
		delta += cost - s.cur.cost[c.resource]
	}
	pr.delta = delta + s.assignmentDelta(pr)
	pr.valid = true

	return true
}

func (s *search) chainCost(resource int, chain []int) (int64, bool) {
	if s.opt.MaxPerResource > 0 && len(chain) > s.opt.MaxPerResource {
		return 0, false
	}
	if len(chain) > 0 && !s.hood.starts(chain[0], resource) {
		return 0, false
	}

	return s.cache.chain(resource, chain)
}

func (s *search) assignmentDelta(pr *proposal) int64 {
	s.epoch++
	s.touched = s.touched[:0]
	for i := range pr.n {
		c := pr.changes[i]
		for _, task := range c.chain {
			s.stamp(task, c.resource)
		}
	}
	for i := range pr.n {
		for _, task := range s.cur.chains[pr.changes[i].resource] {
			if s.mark[task] != s.epoch {
				s.stamp(task, Unassigned)
			}
		}
	}

	delta := int64(0)
	for _, task := range s.touched {
		if to, from := s.markTo[task], s.cur.owner[task]; to != from {
			delta += s.term(task, to) - s.term(task, from)
		}
	}

	return delta
}

func (s *search) stamp(task, to int) {
	s.mark[task] = s.epoch
	s.markTo[task] = to
	s.touched = append(s.touched, task)
}

func (s *search) consider() {
	if !s.score(&s.trial) {
		return
	}
	if !s.choice.valid || s.trial.delta < s.choice.delta {
		s.choice.copyFrom(&s.trial)
	}
}

func (s *search) apply(pr *proposal) {
	s.commit(pr.changes[:pr.n], s.cur.objective+pr.delta)
}

func (s *search) commit(changes []change, objective int64) {
	for _, c := range changes {
		for _, task := range s.cur.chains[c.resource] {
			s.cur.release(task)
		}
	}
	for _, c := range changes {
		s.cur.chains[c.resource] = append(s.cur.chains[c.resource][:0], c.chain...)
		s.cur.cost[c.resource] = c.cost
		for at, task := range c.chain {
			s.cur.take(task, c.resource, at)
		}
		if !s.isDirty[c.resource] {
			s.isDirty[c.resource] = true
			s.dirty = append(s.dirty, c.resource)
		}
	}
	s.cur.objective = objective
}

func (s *search) snapshot() {
	for _, r := range s.dirty {
		s.bestChains[r] = append(s.bestChains[r][:0], s.cur.chains[r]...)
		s.bestCost[r] = s.cur.cost[r]
	}
	s.clearDirty()
}

func (s *search) restore() {
	changes := make([]change, 0, len(s.dirty))
	for _, r := range s.dirty {
		changes = append(changes, change{resource: r, chain: s.bestChains[r], cost: s.bestCost[r]})
	}
	s.commit(changes, s.bestObjective)
	s.clearDirty()
}

func (s *search) clearDirty() {
	for _, r := range s.dirty {
		s.isDirty[r] = false
	}
	s.dirty = s.dirty[:0]
}

func (s *search) settle() {
	if !s.curIsBest {
		s.restore()
		s.curIsBest = true
	}
}

func (s *search) seed() {
	problem := assignmentsolver.Problem{
		Cols:      s.p.Resources,
		Rows:      make([][]assignmentsolver.Arc, s.tasks),
		Uncovered: make([]int64, s.tasks),
	}
	for task, arcs := range s.hood.arcs {
		row := make([]assignmentsolver.Arc, len(arcs))
		for i, arc := range arcs {
			row[i] = assignmentsolver.Arc{Col: arc.Col, Cost: arc.Cost + s.term(task, arc.Col)}
		}
		problem.Rows[task] = row
		problem.Uncovered[task] = s.term(task, Unassigned)
	}

	solved := assignmentsolver.Solve(problem)
	changes := make([]change, 0, len(solved.RowAssignment))
	for task, resource := range solved.RowAssignment {
		if resource == assignmentsolver.Unassigned {
			continue
		}
		chain := []int{task}
		cost, ok := s.cache.chain(resource, chain)
		if !ok {
			continue
		}
		changes = append(changes, change{resource: resource, chain: chain, cost: cost})
	}
	s.commit(changes, 0)
	s.cur.objective = s.objective()
	s.bestObjective = s.cur.objective
	s.curIsBest = true
}

func (s *search) construct(ctx context.Context) {
	free := slices.Clone(s.cur.order[s.cur.assigned:])
	slices.SortFunc(free, func(a, b int) int {
		if c := cmp.Compare(s.opens(a), s.opens(b)); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	for _, task := range free {
		if s.expired(ctx) {
			break
		}
		s.choice.reset()
		s.bestInsertion(task)
		if s.choice.valid && s.choice.delta < 0 {
			s.apply(&s.choice)
		}
	}
	s.bestObjective = s.cur.objective
}

func (s *search) expired(ctx context.Context) bool {
	if ctx.Err() != nil {
		return true
	}

	return s.opt.Budget > 0 && s.opt.Clock().Sub(s.started) >= s.opt.Budget
}

func (s *search) progress(iteration int) float64 {
	done := float64(iteration) / float64(max(s.opt.Iterations, 1))
	if s.opt.Budget > 0 {
		done = max(done, float64(s.opt.Clock().Sub(s.started))/float64(s.opt.Budget))
	}

	return min(done, 1)
}

func (s *search) result(seeded int64) Result {
	out := Result{
		Assignments:   make([]Assignment, 0, s.cur.assigned),
		Unassigned:    make([]int, 0, s.tasks-s.cur.assigned),
		ResourceValue: s.values,
		Objective:     s.cur.objective,
		SeedObjective: seeded,
	}

	for resource, chain := range s.cur.chains {
		if len(chain) == 0 {
			continue
		}
		s.steps, _ = s.cache.steps(resource, chain, s.steps[:0])
		for at, task := range chain {
			cost := int64(0)
			if at < len(s.steps) {
				cost = s.steps[at]
			}
			out.Assignments = append(out.Assignments, Assignment{
				Task:         task,
				Resource:     resource,
				Sequence:     at,
				Cost:         cost,
				Alternatives: s.alternatives[task],
			})
		}
	}
	for task, owner := range s.cur.owner {
		if owner == Unassigned {
			out.Unassigned = append(out.Unassigned, task)
		}
	}

	out.Stats = s.stats
	out.Stats.CacheHits = s.cache.hits
	out.Stats.CacheMisses = s.cache.misses
	out.Stats.CacheFlushes = s.cache.flushes
	out.Stats.Elapsed = s.opt.Clock().Sub(s.started)

	return out
}
