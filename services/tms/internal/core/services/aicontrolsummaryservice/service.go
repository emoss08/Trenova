// Package aicontrolsummaryservice writes the sentence that heads each tab of
// AI control. The sentence is worked out from the records on every read,
// which is cheap; a model rewords it only when those records say something
// new, in the background, and what it wrote is kept until they move again.
// Reading a tab therefore never waits on a model, and navigating back and
// forth never asks one twice.
package aicontrolsummaryservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	// shadowWindow is how far back a shadow agent's recordings are counted.
	shadowWindow = 30 * 24 * time.Hour
	// failureWindow is how far back a provider's failed calls are read.
	failureWindow = 24 * time.Hour
	weekWindow    = 7 * 24 * time.Hour
	// narratedTTL keeps a model's sentence for as long as its facts hold.
	narratedTTL = 24 * time.Hour
	// plainTTL keeps the plain sentence after a model declined or failed,
	// so the same facts are not sent again for a while.
	plainTTL = time.Hour
	// narrationsPerDay bounds how often a tab is reworded in a day, however
	// often its facts move.
	narrationsPerDay = 24
	narrationTimeout = 45 * time.Second
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Facts      repositories.AIControlFactsRepository
	Providers  repositories.AIProviderRepository
	Controls   repositories.AgentControlRepository
	Cache      repositories.AIControlSummaryCache
	Dismissals repositories.AIProviderFailureDismissalRepository
	Usage      repositories.AIUsageRepository
	Completion services.CompletionService `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	facts      repositories.AIControlFactsRepository
	providers  repositories.AIProviderRepository
	controls   repositories.AgentControlRepository
	cache      repositories.AIControlSummaryCache
	dismissals repositories.AIProviderFailureDismissalRepository
	usage      repositories.AIUsageRepository
	completion services.CompletionService
	inflight   singleflight.Group
	now        func() time.Time
	// detach starts a rewording that outlives the request asking for it.
	detach func(func())
}

var _ services.AIControlSummaryService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.aicontrolsummary"),
		facts:      p.Facts,
		providers:  p.Providers,
		controls:   p.Controls,
		cache:      p.Cache,
		dismissals: p.Dismissals,
		usage:      p.Usage,
		completion: p.Completion,
		now:        time.Now,
		detach:     func(run func()) { go run() },
	}
}

// Summary is a tab's sentence for the facts as they are now.
func (s *Service) Summary(
	ctx context.Context,
	req *services.AIControlSummaryRequest,
) (*aicontrolsummary.Summary, error) {
	if !req.Tab.IsValid() {
		return nil, errortypes.NewValidationError("tab", errortypes.ErrInvalid, "Unknown tab")
	}

	facts, err := s.Facts(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	key := &repositories.AIControlSummaryKey{
		TenantInfo: pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID},
		Tab:        req.Tab,
		FactsHash:  facts.Hash(req.Tab),
	}

	var cached aicontrolsummary.Summary
	hit, err := s.cache.Get(ctx, key, &cached)
	if err != nil {
		s.l.Warn("ai control summary cache unavailable", zap.Error(err))
	}
	if hit {
		cached.Facts = *facts
		return &cached, nil
	}

	plain := &aicontrolsummary.Summary{
		Tab:         req.Tab,
		Segments:    aicontrolsummary.Plain(req.Tab, facts),
		FactsHash:   key.FactsHash,
		GeneratedAt: s.now().Unix(),
		Facts:       *facts,
	}
	if facts.NoProvider() || s.completion == nil || err != nil {
		return plain, nil
	}

	claimed, claimErr := s.cache.ClaimNarration(ctx, key, narrationsPerDay)
	if claimErr != nil {
		s.l.Warn("could not claim an ai control narration", zap.Error(claimErr))
		return plain, nil
	}
	if !claimed {
		return plain, nil
	}

	plain.Pending = true
	s.reword(key, req.TenantInfo, facts, plain)
	return plain, nil
}

// reword asks a model for the sentence once per key, however many readers
// are waiting on it, and keeps what comes back.
func (s *Service) reword(
	key *repositories.AIControlSummaryKey,
	tenantInfo pagination.TenantInfo,
	facts *aicontrolsummary.Facts,
	plain *aicontrolsummary.Summary,
) {
	flight := key.TenantInfo.OrgID.String() + ":" + key.TenantInfo.BuID.String() + ":" +
		string(key.Tab) + ":" + key.FactsHash
	s.detach(func() {
		_, _, _ = s.inflight.Do(flight, func() (any, error) {
			ctx, cancel := context.WithTimeout(context.Background(), narrationTimeout)
			defer cancel()

			kept := *plain
			kept.Pending = false
			ttl := plainTTL
			if segments, ok := s.narrate(ctx, tenantInfo, key.Tab, facts, plain.Segments); ok {
				kept.Segments = segments
				kept.Narrated = true
				kept.GeneratedAt = s.now().Unix()
				ttl = narratedTTL
			}
			if err := s.cache.Set(ctx, key, &kept, ttl); err != nil {
				s.l.Warn("could not keep an ai control summary", zap.Error(err))
			}
			return nil, nil
		})
	})
}

// Facts are what the tabs' sentences may say, read from the records.
func (s *Service) Facts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*aicontrolsummary.Facts, error) {
	now := s.now()
	counts, err := s.facts.AgentCounts(ctx, &repositories.AgentCountsRequest{
		TenantInfo:  tenantInfo,
		ShadowSince: now.Add(-shadowWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}

	enabled, err := s.providers.ListEnabled(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	failures, err := s.facts.ProviderFailures(ctx, &repositories.ProviderFailuresRequest{
		TenantInfo: tenantInfo,
		Since:      now.Add(-failureWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}

	control, err := s.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	all, err := s.providers.ListOrdered(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	week, err := s.usage.Summary(ctx, repositories.AIUsageSummaryRequest{
		TenantInfo: tenantInfo,
		Since:      now.Add(-weekWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}

	return &aicontrolsummary.Facts{
		Agents:         counts,
		ProvidersOn:    len(enabled),
		ProvidersTotal: len(all),
		WeekCalls:      week.Totals.Calls,
		AwaitingKey:    awaitingKey(all),
		Failing:        failingProviders(enabled, failures),
		Uncovered:      uncoveredTasks(enabled),
		Paused:         control.ShadowMode,
	}, nil
}

func awaitingKey(providers []*aiprovider.Provider) *aicontrolsummary.ProviderRef {
	for _, provider := range providers {
		if provider.Kind.RequiresAPIKey() && !provider.HasAPIKey {
			return &aicontrolsummary.ProviderRef{ProviderID: provider.ID, Name: provider.Name}
		}
	}
	return nil
}

// failingProviders names the enabled providers whose last call failed.
// A disabled provider's failures are its own business, not the overview's.
func failingProviders(
	enabled []*aiprovider.Provider,
	failures []repositories.ProviderFailureCount,
) []aicontrolsummary.ProviderFailure {
	names := make(map[string]string, len(enabled))
	for _, provider := range enabled {
		names[provider.ID.String()] = provider.Name
	}

	out := make([]aicontrolsummary.ProviderFailure, 0, len(failures))
	for _, failure := range failures {
		name, ok := names[failure.ProviderID.String()]
		if !ok {
			continue
		}
		out = append(out, aicontrolsummary.ProviderFailure{
			ProviderID:    failure.ProviderID,
			Name:          name,
			FailedCalls:   failure.FailedCalls,
			LastFailureAt: failure.LastFailureAt,
		})
	}
	return out
}

func uncoveredTasks(enabled []*aiprovider.Provider) int {
	uncovered := 0
	for _, task := range aiprovider.AllTasks() {
		if aiprovider.RouteFor(enabled, task) == nil {
			uncovered++
		}
	}
	return uncovered
}

