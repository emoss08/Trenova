package managedusageservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	hour         = int64(time.Hour / time.Second)
	MaxLookback  = 35 * 24 * time.Hour
	stateTTL     = 30 * time.Second
	cachedScopes = 10_000
)

var (
	ErrSinceNotOnTheHour = errors.New("since must be on the hour")
	ErrSinceTooOld       = fmt.Errorf("since must be within %s", MaxLookback)
	ErrSinceInFuture     = errors.New("since cannot be after the current hour")
)

type Reading struct {
	Meter    platformcatalog.MeterKey
	Hour     int64
	Quantity int64
}

type Params struct {
	fx.In

	Repo     managedusageports.StateRepository
	Scopes   *managedscope.Resolver
	Counters []managedusageports.HourlyCounter `group:"managed_hourly_counters"`
	Levels   []managedusageports.LevelReader   `group:"managed_level_readers"`
	Clock    timeutils.Clock
	Logger   *zap.Logger
}

type Service struct {
	repo     managedusageports.StateRepository
	scopes   *managedscope.Resolver
	counters []managedusageports.HourlyCounter
	levels   []managedusageports.LevelReader
	clock    timeutils.Clock
	l        *zap.Logger
	cache    *otter.Cache[string, cached]
}

type cached struct {
	state *managedusage.State
}

var (
	_ managedusageports.Gate        = (*Service)(nil)
	_ managedusageports.UsageReader = (*Service)(nil)
)

func New(p Params) *Service {
	counters := slices.Clone(p.Counters)
	slices.SortFunc(counters, func(a, b managedusageports.HourlyCounter) int {
		return cmp.Compare(a.Meter(), b.Meter())
	})
	levels := slices.Clone(p.Levels)
	slices.SortFunc(levels, func(a, b managedusageports.LevelReader) int {
		return cmp.Compare(a.Meter(), b.Meter())
	})
	return &Service{
		repo:     p.Repo,
		scopes:   p.Scopes,
		counters: counters,
		levels:   levels,
		clock:    p.Clock,
		l:        p.Logger.Named("managed-usage"),
		cache: otter.Must(&otter.Options[string, cached]{
			MaximumSize:      cachedScopes,
			ExpiryCalculator: otter.ExpiryWriting[string, cached](stateTTL),
		}),
	}
}

func (s *Service) State(
	ctx context.Context,
	scope managedscope.Scope,
) (*managedusage.State, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	c, err := s.cache.Get(ctx, scope.Key(), otter.LoaderFunc[string, cached](
		func(ctx context.Context, _ string) (cached, error) {
			state, err := s.repo.Get(ctx, scope)
			if errors.Is(err, managedusage.ErrNoState) {
				return cached{}, nil
			}
			if err != nil {
				return cached{}, err
			}
			return cached{state: state}, nil
		},
	))
	if err != nil {
		return nil, err
	}
	return c.state, nil
}

func (s *Service) Decide(
	ctx context.Context,
	scope managedscope.Scope,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	state, err := s.State(ctx, scope)
	if err != nil {
		return managedusage.Decision{}, err
	}
	reader, isLevel := s.levelReader(meter)
	if _, named := stateMeter(state, meter); !isLevel || !named {
		return managedusage.Decide(state, meter, requested), nil
	}
	level, err := reader.Level(ctx, scope)
	if err != nil {
		return managedusage.Decision{}, fmt.Errorf("read the level of %s: %w", meter, err)
	}
	return managedusage.DecideAt(state, meter, level, requested), nil
}

func (s *Service) Usage(
	ctx context.Context,
	scope managedscope.Scope,
) (*managedusage.Usage, error) {
	state, err := s.State(ctx, scope)
	if err != nil {
		return nil, err
	}
	if state == nil || !state.HasWindow() {
		return managedusage.UsageOf(state, nil), nil
	}
	levels := make(map[platformcatalog.MeterKey]int64, len(s.levels))
	for _, m := range state.Meters {
		reader, isLevel := s.levelReader(m.Meter)
		if !isLevel {
			continue
		}
		level, lerr := reader.Level(ctx, scope)
		if lerr != nil {
			return nil, fmt.Errorf("read the level of %s: %w", m.Meter, lerr)
		}
		levels[m.Meter] = level
	}
	return managedusage.UsageOf(state, levels), nil
}

func (s *Service) levelReader(
	meter platformcatalog.MeterKey,
) (managedusageports.LevelReader, bool) {
	for _, lr := range s.levels {
		if lr.Meter() == meter {
			return lr, true
		}
	}
	return nil, false
}

func stateMeter(
	state *managedusage.State,
	meter platformcatalog.MeterKey,
) (managedusage.Meter, bool) {
	if state == nil {
		return managedusage.Meter{}, false
	}
	return state.Meter(meter)
}

func (s *Service) SetState(
	ctx context.Context,
	scope managedscope.Scope,
	state *managedusage.State,
) (*managedusageports.StateChange, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	if err := state.Validate(); err != nil {
		return nil, err
	}
	state.ScopeKey = scope.Key()
	state.ReceivedAt = s.clock()
	previous, err := s.repo.Get(ctx, scope)
	if err != nil && !errors.Is(err, managedusage.ErrNoState) {
		return nil, err
	}
	if err = s.repo.Save(ctx, scope, state); err != nil {
		return nil, err
	}
	s.cache.Invalidate(scope.Key())
	s.l.Info("the control plane set the usage state",
		zap.String("scope", scope.Key()),
		zap.Bool("window", state.HasWindow()),
		zap.Bool("overage", state.Overage),
		zap.Int("meters", len(state.Meters)),
		zap.Int("stopped", stoppedCount(state)),
	)
	return &managedusageports.StateChange{Scope: scope, Previous: previous, Current: state}, nil
}

func stoppedCount(state *managedusage.State) int {
	n := 0
	for i := range state.Meters {
		if state.Meters[i].Stopped {
			n++
		}
	}
	return n
}

func (s *Service) Readings(
	ctx context.Context,
	scope managedscope.Scope,
	since int64,
) ([]Reading, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	now := s.clock()
	current := now - now%hour
	switch {
	case since%hour != 0:
		return nil, ErrSinceNotOnTheHour
	case since > current:
		return nil, ErrSinceInFuture
	case since < current-int64(MaxLookback/time.Second):
		return nil, ErrSinceTooOld
	}
	hours := int((current-since)/hour) + 1
	readings := make([]Reading, 0, len(s.counters)*hours+len(s.levels))
	for _, c := range s.counters {
		counts, err := c.CountByHour(ctx, &managedusageports.HourCountRequest{
			Scope: scope,
			Since: since,
			Until: current + hour,
		})
		if err != nil {
			return nil, fmt.Errorf("count %s by hour: %w", c.Meter(), err)
		}
		byHour := make(map[int64]int64, len(counts))
		for _, hc := range counts {
			byHour[hc.Hour-hc.Hour%hour] += hc.Quantity
		}
		for h := since; h <= current; h += hour {
			readings = append(readings, Reading{Meter: c.Meter(), Hour: h, Quantity: byHour[h]})
		}
	}
	for _, lr := range s.levels {
		level, err := lr.Level(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("read the level of %s: %w", lr.Meter(), err)
		}
		readings = append(readings, Reading{Meter: lr.Meter(), Hour: current, Quantity: level})
	}
	return readings, nil
}
