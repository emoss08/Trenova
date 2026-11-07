package dispatchplanner

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/shared/assignmentsolver"
)

const (
	Unassigned = -1
	Root       = int32(-1)

	defaultNeighbors  = 8
	defaultCacheNodes = 1 << 18
	minCacheNodes     = 1 << 10
	maxCacheNodes     = 1 << 24
)

var (
	ErrShape = errors.New(
		"dispatchplanner: problem inputs disagree on the number of tasks",
	)
	ErrMissingTimeline = errors.New(
		"dispatchplanner: a problem with tasks and resources needs a timeline",
	)
)

type Step struct {
	Cost     int64
	Feasible bool
}

type Extension struct {
	Resource int
	Task     int
	Parent   int32
	Node     int32
}

type Timeline interface {
	Extend(ext Extension) Step
	Reset()
}

type Options struct {
	MaxPerResource int
	Iterations     int
	Budget         time.Duration
	RandomSeed     uint64
	StabilityCents int64
	Neighbors      int
	Alternatives   int
	CacheNodes     int
	Clock          func() time.Time
}

type Problem struct {
	Resources  int
	Arcs       [][]assignmentsolver.Arc
	Coverage   []int64
	Opens      []int64
	Successors [][]int
	Incumbent  []int
	Timeline   Timeline
	Options    Options
}

type Alternative struct {
	Resource  int
	ExtraCost int64
}

type Assignment struct {
	Task         int
	Resource     int
	Sequence     int
	Cost         int64
	Alternatives []Alternative
}

type Stats struct {
	Iterations   int
	Accepted     int
	Improved     int
	Restarts     int
	CacheHits    int64
	CacheMisses  int64
	CacheFlushes int
	Elapsed      time.Duration
}

type Result struct {
	Assignments   []Assignment
	Unassigned    []int
	ResourceValue []int64
	Objective     int64
	SeedObjective int64
	Stats         Stats
}

func Solve(ctx context.Context, p *Problem) (Result, error) {
	if err := p.validate(); err != nil {
		return Result{}, err
	}

	s := newSearch(p)
	s.seed()
	seeded := s.cur.objective
	s.construct(ctx)
	s.anneal(ctx)
	s.settle()
	s.price(ctx)

	return s.result(seeded), nil
}

func (p *Problem) validate() error {
	tasks := len(p.Arcs)
	switch {
	case p.Resources < 0, len(p.Coverage) != tasks:
		return ErrShape
	case p.Opens != nil && len(p.Opens) != tasks:
		return ErrShape
	case p.Successors != nil && len(p.Successors) != tasks:
		return ErrShape
	case p.Incumbent != nil && len(p.Incumbent) != tasks:
		return ErrShape
	case tasks > 0 && p.Resources > 0 && p.Timeline == nil:
		return ErrMissingTimeline
	default:
		return nil
	}
}

func (o Options) normalized() Options {
	if o.Neighbors <= 0 {
		o.Neighbors = defaultNeighbors
	}
	if o.CacheNodes <= 0 {
		o.CacheNodes = defaultCacheNodes
	}
	o.CacheNodes = min(max(o.CacheNodes, minCacheNodes), maxCacheNodes)
	o.Alternatives = max(o.Alternatives, 0)
	o.StabilityCents = max(o.StabilityCents, 0)
	if o.Clock == nil {
		o.Clock = time.Now
	}

	return o
}
