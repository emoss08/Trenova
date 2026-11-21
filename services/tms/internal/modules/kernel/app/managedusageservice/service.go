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
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	hour         = int64(time.Hour / time.Second)
	relayMargin  = int64(2 * time.Minute / time.Second)
	MaxLookback  = 35 * 24 * time.Hour
	stateTTL     = 30 * time.Second
	cachedScopes = 10_000
)

var (
	ErrSinceNotOnTheHour = errors.New("since must be on the hour")
	ErrSinceTooOld       = fmt.Errorf("since must be within %s", MaxLookback)
	ErrSinceInFuture     = errors.New("since cannot be after the current hour")
)

type Reading = managedusageports.MeterReading

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
	m, named := stateMeter(state, meter)
	if !named {
		return managedusage.Decide(state, meter, requested), nil
	}
	if reader, isLevel := s.levelReader(meter); isLevel {
		level, lerr := reader.Level(ctx, scope)
		if lerr != nil {
			return managedusage.Decision{}, fmt.Errorf("read the level of %s: %w", meter, lerr)
		}
		return managedusage.DecideAt(state, meter, level, requested), nil
	}
	counter, counted := s.counter(meter)
	if !counted || !limited(state, &m) {
		return managedusage.Decide(state, meter, requested), nil
	}
	recent, err := s.countedSince(ctx, scope, counter, state)
	if err != nil {
		return managedusage.Decision{}, err
	}
	return managedusage.DecideAt(state, meter, m.Quantity+recent, requested), nil
}

func limited(state *managedusage.State, m *managedusage.Meter) bool {
	return state.HasWindow() && (m.Ceiling > 0 || m.HardCap || !state.Overage)
}

func (s *Service) countedSince(
	ctx context.Context,
	scope managedscope.Scope,
	counter managedusageports.HourlyCounter,
	state *managedusage.State,
) (int64, error) {
	now := s.clock()
	since := max(state.ReceivedAt-relayMargin, state.WindowStart)
	if since >= now+1 {
		return 0, nil
	}
	counts, err := counter.CountByHour(ctx, &managedusageports.HourCountRequest{
		Scope: scope,
		Since: since,
		Until: now + 1,
	})
	if err != nil {
		return 0, fmt.Errorf("count %s since the last usage answer: %w", counter.Meter(), err)
	}
	var total int64
	for _, c := range counts {
		total += c.Quantity
	}
	return total, nil
}

func (s *Service) counter(
	meter platformcatalog.MeterKey,
) (managedusageports.HourlyCounter, bool) {
	for _, c := range s.counters {
		if c.Meter() == meter {
			return c, true
		}
	}
	return nil, false
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

func (s *Service) currentHour(since int64) (int64, error) {
	now := s.clock()
	current := now - now%hour
	switch {
	case since%hour != 0:
		return 0, ErrSinceNotOnTheHour
	case since > current:
		return 0, ErrSinceInFuture
	case since < current-int64(MaxLookback/time.Second):
		return 0, ErrSinceTooOld
	}
	return current, nil
}

func (s *Service) ReadingsForOrganizations(
	ctx context.Context,
	organizationIDs []pulid.ID,
	since int64,
) (map[pulid.ID][]Reading, error) {
	if !s.scopes.Shared() {
		return nil, managedscope.ErrWrongTenancy
	}
	current, err := s.currentHour(since)
	if err != nil {
		return nil, err
	}
	hours := int((current-since)/hour) + 1
	out := make(map[pulid.ID][]Reading, len(organizationIDs))
	for _, id := range organizationIDs {
		out[id] = make([]Reading, 0, len(s.counters)*hours+len(s.levels))
	}
	if len(organizationIDs) == 0 {
		return out, nil
	}
	for _, c := range s.counters {
		counts, cerr := c.CountByHourForOrganizations(
			ctx,
			&managedusageports.OrganizationsHourCountRequest{
				OrganizationIDs: organizationIDs,
				Since:           since,
				Until:           current + hour,
			},
		)
		if cerr != nil {
			return nil, fmt.Errorf("count %s by hour and organization: %w", c.Meter(), cerr)
		}
		byOrgHour := make(map[pulid.ID]map[int64]int64, len(organizationIDs))
		for _, hc := range counts {
			perHour, ok := byOrgHour[hc.OrganizationID]
			if !ok {
				perHour = make(map[int64]int64)
				byOrgHour[hc.OrganizationID] = perHour
			}
			perHour[hc.Hour-hc.Hour%hour] += hc.Quantity
		}
		for _, id := range organizationIDs {
			for h := since; h <= current; h += hour {
				out[id] = append(out[id], Reading{
					Meter:    c.Meter(),
					Hour:     h,
					Quantity: byOrgHour[id][h],
				})
			}
		}
	}
	for _, lr := range s.levels {
		levels, lerr := lr.LevelsForOrganizations(ctx, organizationIDs)
		if lerr != nil {
			return nil, fmt.Errorf("read the level of %s by organization: %w", lr.Meter(), lerr)
		}
		byOrg := make(map[pulid.ID]int64, len(levels))
		for _, l := range levels {
			byOrg[l.OrganizationID] += l.Quantity
		}
		for _, id := range organizationIDs {
			out[id] = append(
				out[id],
				Reading{Meter: lr.Meter(), Hour: current, Quantity: byOrg[id]},
			)
		}
	}
	return out, nil
}

func (s *Service) Readings(
	ctx context.Context,
	scope managedscope.Scope,
	since int64,
) ([]Reading, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	current, err := s.currentHour(since)
	if err != nil {
		return nil, err
	}
	hours := int((current-since)/hour) + 1
	readings := make([]Reading, 0, len(s.counters)*hours+len(s.levels))
	for _, c := range s.counters {
		counts, cerr := c.CountByHour(ctx, &managedusageports.HourCountRequest{
			Scope: scope,
			Since: since,
			Until: current + hour,
		})
		if cerr != nil {
			return nil, fmt.Errorf("count %s by hour: %w", c.Meter(), cerr)
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
		level, lerr := lr.Level(ctx, scope)
		if lerr != nil {
			return nil, fmt.Errorf("read the level of %s: %w", lr.Meter(), lerr)
		}
		readings = append(readings, Reading{Meter: lr.Meter(), Hour: current, Quantity: level})
	}
	return readings, nil
}
