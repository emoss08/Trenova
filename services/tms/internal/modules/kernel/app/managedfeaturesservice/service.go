package managedfeaturesservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	stateTTL            = 30 * time.Second
	cachedScopes        = 10_000
	unknownLogEvery     = time.Hour
	baseAnywhere        = platformcatalog.FeatureKey("")
	remindersPerPage    = 500
	reminderPagesPerRun = 40
)

var ErrInstanceNotSuspended = errors.New(
	"a dedicated instance is always active; only a shared server's organization scopes are suspended",
)

type Params struct {
	fx.In

	Repo   managedfeaturesports.StateRepository
	Scopes *managedscope.Resolver
	Clock  timeutils.Clock
	Logger *zap.Logger
}

type Service struct {
	repo     managedfeaturesports.StateRepository
	scopes   *managedscope.Resolver
	clock    timeutils.Clock
	l        *zap.Logger
	cache    *otter.Cache[string, cached]
	anywhere *otter.Cache[platformcatalog.FeatureKey, bool]
	unknown  *rate.Sometimes
}

type cached struct {
	state *managedfeatures.State
}

var _ platformcatalogports.FeatureGate = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		repo:   p.Repo,
		scopes: p.Scopes,
		clock:  p.Clock,
		l:      p.Logger.Named("managed-features"),
		cache: otter.Must(&otter.Options[string, cached]{
			MaximumSize:      cachedScopes,
			ExpiryCalculator: otter.ExpiryWriting[string, cached](stateTTL),
		}),
		anywhere: otter.Must(&otter.Options[platformcatalog.FeatureKey, bool]{
			MaximumSize:      len(platformcatalog.Features()) + 1,
			ExpiryCalculator: otter.ExpiryWriting[platformcatalog.FeatureKey, bool](stateTTL),
		}),
		unknown: &rate.Sometimes{First: 1, Interval: unknownLogEvery},
	}
}

func (s *Service) State(
	ctx context.Context,
	scope managedscope.Scope,
) (*managedfeatures.State, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	c, err := s.cache.Get(ctx, scope.Key(), otter.LoaderFunc[string, cached](
		func(ctx context.Context, _ string) (cached, error) {
			state, err := s.repo.Get(ctx, scope)
			if errors.Is(err, managedfeatures.ErrNoState) {
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

func (s *Service) Enabled(ctx context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	known, ok := platformcatalog.Lookup(feature)
	if !ok {
		return false, nil
	}
	scope, err := s.scopes.Scope(ctx)
	if err != nil {
		return false, err
	}
	if known.IsBase() && !s.scopes.Shared() {
		return true, nil
	}
	state, err := s.State(ctx, scope)
	if err != nil {
		return false, err
	}
	if known.IsBase() {
		return state.HasList() && !state.Suspended(), nil
	}
	return state.Has(feature), nil
}

func (s *Service) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	known, ok := platformcatalog.Lookup(feature)
	if !ok {
		return false, nil
	}
	if !s.scopes.Shared() {
		if known.IsBase() {
			return true, nil
		}
		state, err := s.State(ctx, managedscope.Instance())
		if err != nil {
			return false, err
		}
		return state.Has(feature), nil
	}

	key := feature
	if known.IsBase() {
		key = baseAnywhere
	}
	return s.anywhere.Get(ctx, key, otter.LoaderFunc[platformcatalog.FeatureKey, bool](
		func(ctx context.Context, key platformcatalog.FeatureKey) (bool, error) {
			if key == baseAnywhere {
				return s.repo.AnyActiveScope(ctx, nil)
			}
			return s.repo.AnyActiveScope(ctx, &key)
		},
	))
}

func (s *Service) SetFeatures(
	ctx context.Context,
	scope managedscope.Scope,
	received *managedfeatures.Received,
) (*managedfeatures.Replacement, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	now := s.clock()
	if err := received.Validate(now); err != nil {
		return nil, err
	}
	replacement, err := s.repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Scope:    scope,
		Received: received,
		Now:      now,
	})
	if err != nil {
		return nil, err
	}
	if !replacement.Updated() {
		s.logUnknown(replacement)
		return replacement, nil
	}
	s.forget(scope)
	s.logChange(replacement)
	return replacement, nil
}

func (s *Service) SetLifecycle(
	ctx context.Context,
	scope managedscope.Scope,
	lifecycle managedfeatures.Lifecycle,
) (*managedfeatures.Replacement, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, err
	}
	if scope.IsInstance() {
		return nil, ErrInstanceNotSuspended
	}
	if !lifecycle.IsValid() {
		return nil, fmt.Errorf("%w: %q", managedfeatures.ErrUnknownLifecycle, lifecycle)
	}
	replacement, err := s.repo.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     scope,
		Lifecycle: lifecycle,
	})
	if err != nil {
		return nil, err
	}
	if !replacement.Updated() {
		return replacement, nil
	}
	s.forget(scope)
	s.l.Info("the control plane set a scope's lifecycle",
		zap.String("scope", scope.Key()),
		zap.String("lifecycle", string(lifecycle)),
	)
	return replacement, nil
}

func (s *Service) History(
	ctx context.Context,
	scope managedscope.Scope,
	limit int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	if err := s.scopes.Accepts(scope); err != nil {
		return nil, nil, err
	}
	return s.repo.History(ctx, scope, limit)
}

func (s *Service) RemindPlanEnds(ctx context.Context) ([]*managedfeatures.Replacement, error) {
	now := s.clock()
	reminded := make([]*managedfeatures.Replacement, 0)
	var errs []error
	var after *managedfeaturesports.DueReminder
	for range reminderPagesPerRun {
		due, err := s.repo.DuePlanEndReminders(ctx, &managedfeaturesports.DueRemindersRequest{
			Now:   now,
			Limit: remindersPerPage,
			After: after,
		})
		if err != nil {
			return reminded, errors.Join(append(errs, err)...)
		}
		for _, item := range due {
			if err = ctx.Err(); err != nil {
				return reminded, errors.Join(append(errs, err)...)
			}
			replacement, rerr := s.repo.RemindPlanEnd(ctx, item.Scope, now)
			if rerr != nil {
				s.l.Warn("could not record a plan end reminder; it is tried again next run",
					zap.String("scope", item.Scope.Key()),
					zap.Error(rerr),
				)
				errs = append(errs, fmt.Errorf("remind %s: %w", item.Scope.Key(), rerr))
				continue
			}
			if !replacement.Updated() {
				continue
			}
			s.forget(item.Scope)
			s.l.Info("reminding the administrators that the plan ends in three days",
				zap.String("scope", item.Scope.Key()),
				zap.Int64("planEndsAt", replacement.Current.PlanEndsAt),
			)
			reminded = append(reminded, replacement)
		}
		if len(due) < remindersPerPage {
			return reminded, errors.Join(errs...)
		}
		after = &due[len(due)-1]
	}
	s.l.Info("plan end reminders reached the bound of one run; the next run continues",
		zap.Int("reminded", len(reminded)),
	)
	return reminded, errors.Join(errs...)
}

func (s *Service) forget(scope managedscope.Scope) {
	s.cache.Invalidate(scope.Key())
	s.anywhere.InvalidateAll()
}

func (s *Service) logChange(replacement *managedfeatures.Replacement) {
	fields := []zap.Field{
		zap.String("scope", replacement.Scope.Key()),
		zap.Int("features", len(replacement.Current.Features)),
		zap.Int("addOnEndings", len(replacement.Current.AddOnEndings)),
		zap.Bool("planUrl", replacement.Current.PlanURL != ""),
		zap.Int64("planEndsAt", replacement.Current.PlanEndsAt),
	}
	if replacement.PlanEnd != nil {
		fields = append(fields, zap.String("planEndNotice", string(replacement.PlanEnd.Kind)))
	}
	if replacement.Changed() {
		fields = append(fields,
			zap.Strings("added", keyStrings(replacement.Change.Added)),
			zap.Strings("removed", keyStrings(replacement.Change.Removed)),
			zap.Strings("ending", endingStrings(replacement.Change.Ending)),
			zap.Strings("kept", keyStrings(replacement.Change.Kept)),
		)
	}
	if !hasUnknown(replacement) {
		s.l.Info("the control plane set the feature list", fields...)
		return
	}
	s.l.Warn(
		"the control plane set the feature list and named features or add-ons this instance does not know",
		append(fields, unknownFields(replacement)...)...,
	)
}

func (s *Service) logUnknown(replacement *managedfeatures.Replacement) {
	if !hasUnknown(replacement) {
		return
	}
	s.unknown.Do(func() {
		s.l.Warn(
			"the control plane names features or add-ons this instance does not know; they are ignored",
			unknownFields(replacement)...,
		)
	})
}

func hasUnknown(replacement *managedfeatures.Replacement) bool {
	return len(replacement.Unknown) > 0 || len(replacement.UnknownAddOns) > 0
}

func unknownFields(replacement *managedfeatures.Replacement) []zap.Field {
	return []zap.Field{
		zap.Strings("ignored", keyStrings(replacement.Unknown)),
		zap.Strings("ignoredAddOns", keyStrings(replacement.UnknownAddOns)),
	}
}

func keyStrings[K ~string](keys []K) []string {
	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = string(key)
	}
	return out
}

func endingStrings(endings []managedfeatures.AddOnEnding) []string {
	out := make([]string, len(endings))
	for i, ending := range endings {
		out[i] = string(ending.AddOn) + "@" + time.Unix(ending.EndsAt, 0).UTC().Format(time.RFC3339)
	}
	return out
}
