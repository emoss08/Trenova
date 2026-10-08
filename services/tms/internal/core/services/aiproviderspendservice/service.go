package aiproviderspendservice

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const (
	cacheTTL       = time.Minute
	maxCachedSpend = 4096
)

type Params struct {
	fx.In

	Usage repositories.AIUsageRepository
}

type entry struct {
	spend     decimal.Decimal
	month     int64
	fetchedAt time.Time
}

type Service struct {
	usage  repositories.AIUsageRepository
	clock  func() time.Time
	ttl    time.Duration
	mu     sync.Mutex
	cached map[pulid.ID]entry
}

var _ services.AIProviderSpendService = (*Service)(nil)

func New(p Params) *Service {
	return newService(p.Usage, time.Now)
}

func newService(usage repositories.AIUsageRepository, clock func() time.Time) *Service {
	return &Service{
		usage:  usage,
		clock:  clock,
		ttl:    cacheTTL,
		cached: make(map[pulid.ID]entry),
	}
}

func (s *Service) MonthSpend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerIDs []pulid.ID,
) (map[pulid.ID]decimal.Decimal, error) {
	out := make(map[pulid.ID]decimal.Decimal, len(providerIDs))
	if len(providerIDs) == 0 {
		return out, nil
	}

	now := s.clock()
	month := timeutils.MonthStartUTC(now.Unix())
	missing := s.fromCache(out, providerIDs, month, now)
	if len(missing) == 0 {
		return out, nil
	}

	fetched, err := s.usage.SpendByProvider(ctx, repositories.AIUsageProviderSpendRequest{
		TenantInfo:  tenantInfo,
		ProviderIDs: missing,
		Since:       month,
	})
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.cached)+len(missing) > maxCachedSpend {
		s.evictLocked(now)
	}
	for _, id := range missing {
		spend := fetched[id]
		out[id] = spend
		s.cached[id] = entry{spend: spend, month: month, fetchedAt: now}
	}

	return out, nil
}

func (s *Service) fromCache(
	out map[pulid.ID]decimal.Decimal,
	providerIDs []pulid.ID,
	month int64,
	now time.Time,
) []pulid.ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	missing := make([]pulid.ID, 0, len(providerIDs))
	for _, id := range providerIDs {
		if _, seen := out[id]; seen {
			continue
		}
		cached, ok := s.cached[id]
		if ok && cached.month == month && now.Sub(cached.fetchedAt) < s.ttl {
			out[id] = cached.spend
			continue
		}
		out[id] = decimal.Zero
		missing = append(missing, id)
	}

	return missing
}

func (s *Service) evictLocked(now time.Time) {
	for id, cached := range s.cached {
		if now.Sub(cached.fetchedAt) >= s.ttl {
			delete(s.cached, id)
		}
	}
	if len(s.cached) >= maxCachedSpend {
		clear(s.cached)
	}
}
