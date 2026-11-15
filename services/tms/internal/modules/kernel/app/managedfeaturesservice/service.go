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
)

const (
	stateTTL = 30 * time.Second
	stateKey = managedfeatures.CurrentStateID
)

type Params struct {
	fx.In

	Repo   managedfeaturesports.StateRepository
	Clock  timeutils.Clock
	Logger *zap.Logger
}

type Service struct {
	repo  managedfeaturesports.StateRepository
	clock timeutils.Clock
	l     *zap.Logger
	cache *otter.Cache[string, cached]
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
	features []platformcatalog.FeatureKey,
) (*managedfeatures.Replacement, error) {
	replacement, err := s.repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Features: features,
		Now:      s.clock(),
	})
	if err != nil {
		return nil, err
	}
	if !replacement.Changed() {
		return replacement, nil
	}
	s.cache.Invalidate(stateKey)
	s.logChange(replacement)
	return replacement, nil
}

func (s *Service) logChange(replacement *managedfeatures.Replacement) {
	fields := []zap.Field{
		zap.Int("features", len(replacement.Current.Features)),
		zap.Strings("added", keyStrings(replacement.Change.Added)),
		zap.Strings("removed", keyStrings(replacement.Change.Removed)),
	}
	if len(replacement.Unknown) == 0 {
		s.l.Info("the control plane set the feature list", fields...)
		return
	}
	s.l.Warn(
		"the control plane set the feature list and named features this instance does not know",
		append(fields, zap.Strings("ignored", keyStrings(replacement.Unknown)))...,
	)
}

func keyStrings(keys []platformcatalog.FeatureKey) []string {
	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = string(key)
	}
	return out
}
