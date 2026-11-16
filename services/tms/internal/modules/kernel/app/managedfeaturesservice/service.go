package managedfeaturesservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
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
	stateTTL        = 30 * time.Second
	stateKey        = managedfeatures.CurrentStateID
	unknownLogEvery = time.Hour
)

type Params struct {
	fx.In

	Repo   managedfeaturesports.StateRepository
	Clock  timeutils.Clock
	Logger *zap.Logger
}

type Service struct {
	repo    managedfeaturesports.StateRepository
	clock   timeutils.Clock
	l       *zap.Logger
	cache   *otter.Cache[string, cached]
	unknown *rate.Sometimes
}

type cached struct {
	state *managedfeatures.State
}

var _ platformcatalogports.FeatureGate = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		repo:  p.Repo,
		clock: p.Clock,
		l:     p.Logger.Named("managed-features"),
		cache: otter.Must(&otter.Options[string, cached]{
			MaximumSize:      1,
			ExpiryCalculator: otter.ExpiryWriting[string, cached](stateTTL),
		}),
		unknown: &rate.Sometimes{First: 1, Interval: unknownLogEvery},
	}
}

func (s *Service) State(ctx context.Context) (*managedfeatures.State, error) {
	c, err := s.cache.Get(ctx, stateKey, otter.LoaderFunc[string, cached](
		func(ctx context.Context, _ string) (cached, error) {
			state, err := s.repo.Get(ctx)
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
	if known.IsBase() {
		return true, nil
	}
	state, err := s.State(ctx)
	if err != nil {
		return false, err
	}
	return state.Has(feature), nil
}

func (s *Service) SetFeatures(
	ctx context.Context,
	received *managedfeatures.Received,
) (*managedfeatures.Replacement, error) {
	now := s.clock()
	if err := received.Validate(now); err != nil {
		return nil, err
	}
	replacement, err := s.repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
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
	s.cache.Invalidate(stateKey)
	s.logChange(replacement)
	return replacement, nil
}

func (s *Service) RemindPlanEnd(ctx context.Context) (*managedfeatures.Replacement, error) {
	replacement, err := s.repo.RemindPlanEnd(ctx, s.clock())
	if err != nil {
		return nil, err
	}
	if !replacement.Updated() {
		return replacement, nil
	}
	s.cache.Invalidate(stateKey)
	s.l.Info("reminding the administrators that the plan ends in three days",
		zap.Int64("planEndsAt", replacement.Current.PlanEndsAt),
	)
	return replacement, nil
}

func (s *Service) logChange(replacement *managedfeatures.Replacement) {
	fields := []zap.Field{
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
