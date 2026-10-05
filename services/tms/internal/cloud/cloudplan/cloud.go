package cloudplan

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	DefaultCacheTTL     = 30 * time.Second
	cachePruneThreshold = 4_096
)

type cacheEntry struct {
	plan      *platformplan.ResolvedPlan
	expiresAt time.Time
}

type CloudConfig struct {
	Catalog       *platformplan.Catalog
	Subscriptions repositories.SubscriptionRepository
	Logger        *zap.Logger
	CacheTTL      time.Duration
	Clock         func() time.Time
}

type CloudService struct {
	catalog       *platformplan.Catalog
	subscriptions repositories.SubscriptionRepository
	l             *zap.Logger
	ttl           time.Duration
	now           func() time.Time

	mu    sync.RWMutex
	cache map[pulid.ID]cacheEntry
}

var _ services.PlanService = (*CloudService)(nil)

func NewCloud(cfg CloudConfig) *CloudService {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CloudService{
		catalog:       cfg.Catalog,
		subscriptions: cfg.Subscriptions,
		l:             logger.Named("plan-service"),
		ttl:           ttl,
		now:           clock,
		cache:         make(map[pulid.ID]cacheEntry),
	}
}

func (s *CloudService) EnforcesPlans() bool {
	return true
}

func (s *CloudService) Resolve(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*platformplan.ResolvedPlan, error) {
	if orgID.IsNil() || buID.IsNil() {
		return nil, planservice.ErrTenantRequired
	}

	now := s.now()
	if cached, ok := s.cached(orgID, buID, now); ok {
		return cached, nil
	}

	resolved, err := s.load(ctx, orgID, buID, now.Unix())
	if err != nil {
		return nil, err
	}

	s.store(orgID, resolved, now)

	return resolved, nil
}

func (s *CloudService) RequireCapability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	capability platformplan.Capability,
) error {
	resolved, err := s.Resolve(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return err
	}

	if !resolved.IsManaged() {
		return nil
	}

	if !resolved.AllowsLogin() {
		return errortypes.NewPlanRestrictionError(
			capability.String(),
			platformplan.ReasonSubscriptionExpired,
			resolved.Key().String(),
		)
	}

	if !resolved.Allows(capability) {
		return errortypes.NewPlanRestrictionError(
			capability.String(),
			platformplan.ReasonPlanRestricted,
			resolved.Key().String(),
		)
	}

	return nil
}

func (s *CloudService) RequireWritable(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	resolved, err := s.Resolve(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return err
	}

	if resolved.AllowsWrites() {
		return nil
	}

	reason := platformplan.ReasonSubscriptionReadOnly
	if !resolved.AllowsLogin() {
		reason = platformplan.ReasonSubscriptionExpired
	}

	return errortypes.NewPlanRestrictionError("", reason, resolved.Key().String())
}

func (s *CloudService) Invalidate(orgID pulid.ID) {
	s.mu.Lock()
	delete(s.cache, orgID)
	s.mu.Unlock()
}

func (s *CloudService) load(
	ctx context.Context,
	orgID, buID pulid.ID,
	now int64,
) (*platformplan.ResolvedPlan, error) {
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	scoped := dbscope.EnsureTenant(ctx, dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID})

	sub, err := s.subscriptions.GetByOrganization(scoped, repositories.GetSubscriptionRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return platformplan.NewUnmanaged(
				s.catalog.Unlimited(),
				platformplan.OriginInternal,
				orgID,
				buID,
				now,
			), nil
		}

		return nil, fmt.Errorf("resolve plan for organization %s: %w", orgID, err)
	}

	plan, err := s.catalog.Get(platformplan.PlanKey(sub.PlanKey))
	if err != nil {
		s.l.Error("organization subscription names an unknown plan",
			zap.String("organizationId", orgID.String()),
			zap.String("planKey", sub.PlanKey),
			zap.Error(err),
		)
		return nil, fmt.Errorf("resolve plan for organization %s: %w", orgID, err)
	}

	return platformplan.NewManaged(plan, sub, now), nil
}

func (s *CloudService) cached(
	orgID, buID pulid.ID,
	now time.Time,
) (*platformplan.ResolvedPlan, bool) {
	s.mu.RLock()
	entry, ok := s.cache[orgID]
	s.mu.RUnlock()

	if !ok || !now.Before(entry.expiresAt) || entry.plan.BusinessUnitID != buID {
		return nil, false
	}

	return entry.plan, true
}

func (s *CloudService) store(orgID pulid.ID, resolved *platformplan.ResolvedPlan, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.cache) >= cachePruneThreshold {
		for key, entry := range s.cache {
			if !now.Before(entry.expiresAt) {
				delete(s.cache, key)
			}
		}
	}

	s.cache[orgID] = cacheEntry{plan: resolved, expiresAt: now.Add(s.ttl)}
}
