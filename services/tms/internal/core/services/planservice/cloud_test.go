package planservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

func newCloud(
	t *testing.T,
	repo *mocks.MockSubscriptionRepository,
	clock *fakeClock,
) *planservice.CloudService {
	t.Helper()

	catalog, err := platformplan.NewCatalog(nil)
	require.NoError(t, err)

	return planservice.NewCloud(planservice.CloudConfig{
		Catalog:       catalog,
		Subscriptions: repo,
		Logger:        zap.NewNop(),
		Clock:         clock.Now,
	})
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func trialSubscription(ti pagination.TenantInfo, now time.Time) *subscription.Subscription {
	return &subscription.Subscription{
		ID:             pulid.MustNew("osub_"),
		OrganizationID: ti.OrgID,
		BusinessUnitID: ti.BuID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    now.Add(30 * 24 * time.Hour).Unix(),
		ReadOnlyUntil:  now.Add(44 * 24 * time.Hour).Unix(),
	}
}

func TestCloudResolve_SubscriptionGivesItsPlan(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().
		GetByOrganization(mock.Anything, repositories.GetSubscriptionRequest{
			TenantInfo: pagination.TenantInfo{OrgID: ti.OrgID, BuID: ti.BuID},
		}).
		Return(trialSubscription(ti, clock.now), nil).
		Once()

	svc := newCloud(t, repo, clock)
	assert.True(t, svc.IsCloud())

	resolved, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
	assert.True(t, resolved.IsManaged())
	assert.Equal(t, platformplan.PlanKeyFreeDemo, resolved.Key())
	assert.Equal(t, subscription.StatusTrialing, resolved.Status)

	again, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
	assert.Same(t, resolved, again, "a second resolve inside the TTL is served from the cache")
}

func TestCloudResolve_MissingSubscriptionIsUnlimitedInternal(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().
		GetByOrganization(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Subscription not found")).
		Once()

	svc := newCloud(t, repo, clock)
	resolved, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
	assert.False(t, resolved.IsManaged())
	assert.Equal(t, platformplan.OriginInternal, resolved.Origin)
	assert.True(t, resolved.Plan.IsUnlimited())
	require.NoError(t, svc.RequireCapability(t.Context(), ti, platformplan.CapabilityAPIKeys))
	require.NoError(t, svc.RequireWritable(t.Context(), ti))
}

func TestCloudResolve_BindsTheTenantScopeForTheLookup(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()
	other := tenantInfo()
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().
		GetByOrganization(mock.MatchedBy(func(ctx context.Context) bool {
			tenant, ok := dbscope.TenantFrom(ctx)
			return ok && tenant.OrganizationID == ti.OrgID && tenant.BusinessUnitID == ti.BuID
		}), mock.Anything).
		Return(trialSubscription(ti, clock.now), nil).
		Once()

	svc := newCloud(t, repo, clock)
	ctx := dbscope.WithTenant(t.Context(), other.DBTenant())
	_, err := svc.Resolve(ctx, ti.OrgID, ti.BuID)
	require.NoError(t, err)
}

func TestCloudResolve_CacheExpiresAndInvalidates(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().
		GetByOrganization(mock.Anything, mock.Anything).
		Return(trialSubscription(ti, clock.now), nil).
		Times(3)

	svc := newCloud(t, repo, clock)

	_, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)

	clock.now = clock.now.Add(planservice.DefaultCacheTTL)
	_, err = svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)

	svc.Invalidate(ti.OrgID)
	_, err = svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
}

func TestCloudResolve_UnknownPlanAndRepositoryErrorsFail(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()

	repo := mocks.NewMockSubscriptionRepository(t)
	sub := trialSubscription(ti, clock.now)
	sub.PlanKey = "enterprise_gold"
	repo.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(sub, nil).Once()

	_, err := newCloud(t, repo, clock).Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.ErrorIs(t, err, platformplan.ErrUnknownPlan)

	failing := mocks.NewMockSubscriptionRepository(t)
	boom := errors.New("connection reset")
	failing.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(nil, boom).Once()

	svc := newCloud(t, failing, clock)
	_, err = svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.ErrorIs(t, err, boom)

	_, err = svc.Resolve(t.Context(), pulid.Nil, ti.BuID)
	require.ErrorIs(t, err, planservice.ErrTenantRequired)
}

func TestCloudRequireCapability(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1_800_000_000, 0)}
	ti := tenantInfo()
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().
		GetByOrganization(mock.Anything, mock.Anything).
		Return(trialSubscription(ti, clock.now), nil).
		Once()

	svc := newCloud(t, repo, clock)

	err := svc.RequireCapability(t.Context(), ti, platformplan.CapabilityEmailOutbound)
	var restricted *errortypes.PlanRestrictionError
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, "email.outbound", restricted.Capability)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restricted.Reason)
	assert.Equal(t, "free_demo", restricted.Plan)

	require.NoError(t, svc.RequireWritable(t.Context(), ti))
}

func TestCloudRequireWritable_ReadOnlyAndExpired(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_800_000_000, 0)
	ti := tenantInfo()
	sub := trialSubscription(ti, start)

	clock := &fakeClock{now: time.Unix(sub.TrialEndsAt, 0)}
	repo := mocks.NewMockSubscriptionRepository(t)
	repo.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(sub, nil).Times(2)
	svc := newCloud(t, repo, clock)

	err := svc.RequireWritable(t.Context(), ti)
	var restricted *errortypes.PlanRestrictionError
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.PlanRestrictionReasonReadOnly, restricted.Reason)

	clock.now = time.Unix(sub.ReadOnlyUntil, 0)
	svc.Invalidate(ti.OrgID)

	err = svc.RequireWritable(t.Context(), ti)
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.PlanRestrictionReasonExpired, restricted.Reason)

	err = svc.RequireCapability(t.Context(), ti, platformplan.CapabilitySSO)
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.PlanRestrictionReasonExpired, restricted.Reason)
}

func TestUnlimitedPlanService(t *testing.T) {
	t.Parallel()

	svc := planservice.NewUnlimited()
	ti := tenantInfo()

	assert.False(t, svc.IsCloud())
	resolved, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
	assert.Equal(t, platformplan.OriginSelfHosted, resolved.Origin)
	assert.True(t, resolved.Plan.IsUnlimited())
	require.NoError(t, svc.RequireCapability(t.Context(), ti, platformplan.CapabilitySMS))
	require.NoError(t, svc.RequireWritable(t.Context(), ti))
	svc.Invalidate(ti.OrgID)
}

func TestNewSelectsByPlatformMode(t *testing.T) {
	t.Parallel()

	catalog, err := planservice.NewCatalog(planservice.CatalogParams{Config: &config.Config{}})
	require.NoError(t, err)

	selfHosted := planservice.New(planservice.Params{
		Config:  &config.Config{},
		Catalog: catalog,
		Logger:  zap.NewNop(),
	})
	assert.IsType(t, &planservice.UnlimitedPlanService{}, selfHosted)

	cloud := planservice.New(planservice.Params{
		Config:        &config.Config{Platform: config.PlatformConfig{Mode: config.PlatformModeCloud}},
		Catalog:       catalog,
		Subscriptions: mocks.NewMockSubscriptionRepository(t),
		Logger:        zap.NewNop(),
	})
	assert.IsType(t, &planservice.CloudService{}, cloud)
}

func TestNewCatalogAppliesConfiguredOverrides(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Platform.Cloud.FreePlan.Limits = map[string]map[string]int64{
		"shipments": {"total": 50},
	}

	catalog, err := planservice.NewCatalog(planservice.CatalogParams{Config: cfg})
	require.NoError(t, err)

	limit, ok := catalog.FreeDemo().Limit("shipments.total")
	require.True(t, ok)
	assert.Equal(t, int64(50), limit.Max)

	cfg.Platform.Cloud.FreePlan.Limits = map[string]map[string]int64{"bogus": {"meter": 1}}
	_, err = planservice.NewCatalog(planservice.CatalogParams{Config: cfg})
	require.ErrorIs(t, err, platformplan.ErrUnknownMeterOverride)
}
