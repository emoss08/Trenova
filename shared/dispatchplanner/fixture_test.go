package dispatchplanner

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/stretchr/testify/require"
)

type pair struct {
	from int
	to   int
}

type tableTimeline struct {
	first  map[pair]int64
	next   map[pair]int64
	last   []int
	calls  int
	resets int
}

func (t *tableTimeline) Extend(ext Extension) Step {
	t.calls++
	for len(t.last) <= int(ext.Node) {
		t.last = append(t.last, Unassigned)
	}

	var (
		cost int64
		ok   bool
	)
	if ext.Parent == Root {
		cost, ok = t.first[pair{from: ext.Resource, to: ext.Task}]
	} else {
		cost, ok = t.next[pair{from: t.last[ext.Parent], to: ext.Task}]
	}
	if !ok {
		return Step{}
	}
	t.last[ext.Node] = ext.Task

	return Step{Cost: cost, Feasible: true}
}

func (t *tableTimeline) Reset() {
	t.last = t.last[:0]
	t.resets++
}

type tableSpec struct {
	resources int
	coverage  []int64
	opens     []int64
	first     map[pair]int64
	next      map[pair]int64
	incumbent []int
	options   Options
}

func tableProblem(spec *tableSpec) (*Problem, *tableTimeline) {
	timeline := &tableTimeline{first: spec.first, next: spec.next}
	tasks := len(spec.coverage)
	arcs := make([][]assignmentsolver.Arc, tasks)
	for _, key := range sortedPairs(spec.first) {
		arcs[key.to] = append(arcs[key.to], assignmentsolver.Arc{
			Col:  key.from,
			Cost: spec.first[key],
		})
	}
	successors := make([][]int, tasks)
	for _, key := range sortedPairs(spec.next) {
		successors[key.from] = append(successors[key.from], key.to)
	}

	return &Problem{
		Resources:  spec.resources,
		Arcs:       arcs,
		Coverage:   spec.coverage,
		Opens:      spec.opens,
		Successors: successors,
		Incumbent:  spec.incumbent,
		Timeline:   timeline,
		Options:    spec.options,
	}, timeline
}

func sortedPairs(m map[pair]int64) []pair {
	keys := make([]pair, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b pair) int {
		if c := cmp.Compare(a.from, b.from); c != 0 {
			return c
		}
		return cmp.Compare(a.to, b.to)
	})

	return keys
}

func searchWith(t testing.TB, p *Problem, chains map[int][]int) *search {
	t.Helper()

	s := newSearch(p)
	resources := make([]int, 0, len(chains))
	for resource := range chains {
		resources = append(resources, resource)
	}
	slices.Sort(resources)

	changes := make([]change, 0, len(chains))
	for _, resource := range resources {
		chain := chains[resource]
		cost, ok := s.cache.chain(resource, chain)
		require.True(t, ok, "chain %v on resource %d must be feasible", chain, resource)
		changes = append(changes, change{resource: resource, chain: chain, cost: cost})
	}
	s.commit(changes, 0)
	s.cur.objective = s.objective()
	s.bestObjective = s.cur.objective
	s.curIsBest = true

	return s
}

func requireConsistent(t testing.TB, s *search) {
	t.Helper()

	require.Equal(t, s.objective(), s.cur.objective, "incremental objective drifted")
	seen := make([]bool, s.tasks)
	assigned := 0
	for resource, chain := range s.cur.chains {
		if s.opt.MaxPerResource > 0 {
			require.LessOrEqual(t, len(chain), s.opt.MaxPerResource)
		}
		cost, ok := s.cache.chain(resource, chain)
		require.True(t, ok, "chain %v on resource %d became infeasible", chain, resource)
		require.Equal(t, cost, s.cur.cost[resource])
		for at, task := range chain {
			require.False(t, seen[task], "task %d is on two chains", task)
			seen[task] = true
			assigned++
			require.Equal(t, resource, s.cur.owner[task])
			require.Equal(t, at, s.cur.pos[task])
		}
	}
	require.Equal(t, assigned, s.cur.assigned)
	for i, task := range s.cur.order {
		require.Equal(t, i, s.cur.slot[task])
		require.Equal(t, i < s.cur.assigned, s.cur.owner[task] != Unassigned)
	}
}

func steppingClock(step time.Duration) func() time.Time {
	now := time.Unix(1_700_000_000, 0)
	return func() time.Time {
		now = now.Add(step)
		return now
	}
}

const (
	synthMilesPerMinute  = 50.0 / 60.0
	synthEmptyMileCents  = 200.0
	synthLateMinuteCents = int64(100)
	synthLateLimit       = int64(240)
	synthDriveCap        = int64(660)
	synthDwell           = int64(30)
	synthWindow          = int64(120)
	synthDay             = 1_440
)

type synthDriver struct {
	x, y float64
	free int64
}

type synthMove struct {
	ox, oy, dx, dy float64
	opens, closes  int64
}

type synthState struct {
	x, y      float64
	at, drive int64
}

type synthLeg struct {
	empty float64
	late  int64
	cost  int64
}

type synthBoard struct {
	drivers  []synthDriver
	moves    []synthMove
	coverage []int64
}

func newSynthBoard(seed uint64, moves, drivers int) *synthBoard {
	rng := rand.New(rand.NewPCG(seed, seed^seedMixer))
	side := 40 * math.Sqrt(float64(max(moves, drivers)))
	board := &synthBoard{
		drivers:  make([]synthDriver, drivers),
		moves:    make([]synthMove, moves),
		coverage: make([]int64, moves),
	}
	for i := range board.drivers {
		board.drivers[i] = synthDriver{
			x:    rng.Float64() * side,
			y:    rng.Float64() * side,
			free: rng.Int64N(240),
		}
	}
	for i := range board.moves {
		ox, oy := rng.Float64()*side, rng.Float64()*side
		angle := rng.Float64() * 2 * math.Pi
		length := 30 + rng.Float64()*370
		opens := rng.Int64N(synthDay)
		board.moves[i] = synthMove{
			ox:     ox,
			oy:     oy,
			dx:     math.Min(math.Max(ox+math.Cos(angle)*length, 0), side),
			dy:     math.Min(math.Max(oy+math.Sin(angle)*length, 0), side),
			opens:  opens,
			closes: opens + synthWindow,
		}
		board.coverage[i] = 30_000 + rng.Int64N(60_000)
	}

	return board
}

func (b *synthBoard) start(resource int) synthState {
	d := b.drivers[resource]
	return synthState{x: d.x, y: d.y, at: d.free}
}

func minutesFor(miles float64) int64 {
	return int64(math.Ceil(miles / synthMilesPerMinute))
}

func (b *synthBoard) step(from synthState, task int) (synthState, synthLeg, bool) {
	m := b.moves[task]
	empty := math.Hypot(m.ox-from.x, m.oy-from.y)
	emptyMinutes := minutesFor(empty)
	arrive := from.at + emptyMinutes
	late := max(arrive-m.closes, 0)
	if late > synthLateLimit {
		return synthState{}, synthLeg{}, false
	}
	loadedMinutes := minutesFor(math.Hypot(m.dx-m.ox, m.dy-m.oy))
	drive := from.drive + emptyMinutes + loadedMinutes
	if drive > synthDriveCap {
		return synthState{}, synthLeg{}, false
	}

	next := synthState{
		x:     m.dx,
		y:     m.dy,
		at:    max(arrive, m.opens) + synthDwell + loadedMinutes + synthDwell,
		drive: drive,
	}
	leg := synthLeg{
		empty: empty,
		late:  late,
		cost:  int64(math.Round(empty*synthEmptyMileCents)) + late*synthLateMinuteCents,
	}

	return next, leg, true
}

func (b *synthBoard) problem(k int, options Options) (*Problem, *synthTimeline) {
	tasks := len(b.moves)
	arcs := make([][]assignmentsolver.Arc, tasks)
	opens := make([]int64, tasks)
	successors := make([][]int, tasks)
	for task := range b.moves {
		opens[task] = b.moves[task].opens
		for resource := range b.drivers {
			if _, leg, ok := b.step(b.start(resource), task); ok {
				arcs[task] = append(arcs[task], assignmentsolver.Arc{Col: resource, Cost: leg.cost})
			}
		}
		slices.SortFunc(arcs[task], func(x, y assignmentsolver.Arc) int {
			if c := cmp.Compare(x.Cost, y.Cost); c != 0 {
				return c
			}
			return cmp.Compare(x.Col, y.Col)
		})
		arcs[task] = arcs[task][:min(len(arcs[task]), k)]
		successors[task] = b.successors(task, k)
	}
	timeline := &synthTimeline{board: b}

	return &Problem{
		Resources:  len(b.drivers),
		Arcs:       arcs,
		Coverage:   b.coverage,
		Opens:      opens,
		Successors: successors,
		Timeline:   timeline,
		Options:    options,
	}, timeline
}

func (b *synthBoard) successors(from, k int) []int {
	type near struct {
		task  int
		miles float64
	}
	m := b.moves[from]
	ready := m.opens + 2*synthDwell + minutesFor(math.Hypot(m.dx-m.ox, m.dy-m.oy))
	found := make([]near, 0, k*4)
	for task, next := range b.moves {
		if task == from {
			continue
		}
		miles := math.Hypot(next.ox-m.dx, next.oy-m.dy)
		if ready+minutesFor(miles) > next.closes {
			continue
		}
		found = append(found, near{task: task, miles: miles})
	}
	slices.SortFunc(found, func(x, y near) int {
		if c := cmp.Compare(x.miles, y.miles); c != 0 {
			return c
		}
		return cmp.Compare(x.task, y.task)
	})

	out := make([]int, 0, min(len(found), k))
	for _, n := range found[:min(len(found), k)] {
		out = append(out, n.task)
	}

	return out
}

type synthTimeline struct {
	board  *synthBoard
	states []synthState
	calls  int
}

func (t *synthTimeline) Extend(ext Extension) Step {
	t.calls++
	from := t.board.start(ext.Resource)
	if ext.Parent != Root {
		from = t.states[ext.Parent]
	}
	next, leg, ok := t.board.step(from, ext.Task)
	if !ok {
		return Step{}
	}
	for len(t.states) <= int(ext.Node) {
		t.states = append(t.states, synthState{})
	}
	t.states[ext.Node] = next

	return Step{Cost: leg.cost, Feasible: true}
}

func (t *synthTimeline) Reset() {
	t.states = t.states[:0]
}

type synthMetrics struct {
	covered     int
	emptyMiles  float64
	lateMinutes int64
	objective   int64
	feasible    bool
}

func (b *synthBoard) measure(result *Result, maxPerResource int) synthMetrics {
	metrics := synthMetrics{feasible: true}
	chains := make(map[int][]int)
	for _, assignment := range result.Assignments {
		chains[assignment.Resource] = append(chains[assignment.Resource], assignment.Task)
	}
	for resource, chain := range chains {
		if maxPerResource > 0 && len(chain) > maxPerResource {
			metrics.feasible = false
		}
		state := b.start(resource)
		for _, task := range chain {
			next, leg, ok := b.step(state, task)
			if !ok {
				metrics.feasible = false
				break
			}
			metrics.covered++
			metrics.emptyMiles += leg.empty
			metrics.lateMinutes += leg.late
			metrics.objective += leg.cost
			state = next
		}
	}
	for _, task := range result.Unassigned {
		metrics.objective += b.coverage[task]
	}

	return metrics
}
