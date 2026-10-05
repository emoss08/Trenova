package cloudquota_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/cloudquota"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var fixedNow = time.Date(2026, time.October, 1, 3, 0, 0, 0, time.UTC)

type harness struct {
	plans    *mocks.MockPlanService
	counters *mocks.MockQuotaCounterRepository
	guard    *cloudquota.CloudGuard
	tenant   pagination.TenantInfo
	inTx     bool
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		plans:    mocks.NewMockPlanService(t),
		counters: mocks.NewMockQuotaCounterRepository(t),
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		inTx: true,
	}
	h.guard = cloudquota.NewCloud(cloudquota.CloudConfig{
		Plans:         h.plans,
		Counters:      h.counters,
		Logger:        zap.NewNop(),
		Clock:         func() time.Time { return fixedNow },
		InTransaction: func(context.Context) bool { return h.inTx },
	})

	return h
}

func newHarnessWithSubscriptions(t *testing.T) (*harness, *mocks.MockSubscriptionRepository) {
	t.Helper()

	h := newHarness(t)
	subs := mocks.NewMockSubscriptionRepository(t)
	h.guard = cloudquota.NewCloud(cloudquota.CloudConfig{
		Plans:         h.plans,
		Counters:      h.counters,
		Subscriptions: subs,
		Logger:        zap.NewNop(),
		Clock:         func() time.Time { return fixedNow },
		InTransaction: func(context.Context) bool { return h.inTx },
	})

	return h, subs
}

func (h *harness) expectShipmentCount(used int64) {
	h.counters.EXPECT().Supports(platformcatalog.MeterShipmentsTotal).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, h.tenant, platformcatalog.MeterShipmentsTotal).Return(nil).Once()
	h.counters.EXPECT().Count(mock.Anything, mock.Anything).Return(used, nil).Once()
}

func (h *harness) freeDemo(t *testing.T, status subscription.Status) *platformplan.ResolvedPlan {
	t.Helper()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	sub := &subscription.Subscription{
		ID:             pulid.MustNew("osub_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         status,
		TrialEndsAt:    fixedNow.Add(24 * time.Hour).Unix(),
		ReadOnlyUntil:  fixedNow.Add(48 * time.Hour).Unix(),
		CreatedAt:      fixedNow.Add(-24 * time.Hour).Unix(),
	}

	return platformplan.NewManaged(plan, sub, fixedNow.Unix())
}

func (h *harness) resolves(plan *platformplan.ResolvedPlan) {
	h.plans.EXPECT().Resolve(mock.Anything, h.tenant.OrgID, h.tenant.BuID).Return(plan, nil)
}

func (h *harness) request(meter platformcatalog.MeterKey, quantity int64) *services.QuotaRequest {
	return &services.QuotaRequest{TenantInfo: h.tenant, Meter: meter, Quantity: quantity}
}

func TestEnforce_AllowsUnderTheLimit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterShipmentsTotal).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, h.tenant, platformcatalog.MeterShipmentsTotal).Return(nil).Once()
	h.counters.EXPECT().
		Count(mock.Anything, &repositories.QuotaCountRequest{
			TenantInfo: h.tenant,
			Meter:      platformcatalog.MeterShipmentsTotal,
		}).
		Return(int64(10), nil).
		Once()

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 2)))
}

func TestEnforce_RefusesTheSlotPastTheLimit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterShipmentsTotal).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, h.tenant, platformcatalog.MeterShipmentsTotal).Return(nil).Once()
	h.counters.EXPECT().Count(mock.Anything, mock.Anything).Return(int64(11), nil).Once()

	err := h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 2))

	var exceeded *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &exceeded)
	assert.Equal(t, "shipments.total", exceeded.Meter)
	assert.Equal(t, int64(12), exceeded.Limit)
	assert.Equal(t, int64(11), exceeded.Used)
	assert.Equal(t, "free_demo", exceeded.Plan)
}

func TestEnforce_RequiresTheWriteTransaction(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.inTx = false
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterCustomersTotal).Return(true)

	err := h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterCustomersTotal, 1))
	require.ErrorIs(t, err, quotaservice.ErrTransactionRequired)
}

func TestEnforce_PerItemLimitNeedsNoCountOrLock(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.inTx = false
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))

	require.NoError(t, h.guard.Enforce(
		t.Context(),
		h.request(platformcatalog.MeterDocumentFileBytes, platformplan.FreeDemoDocumentFileBytes),
	))

	err := h.guard.Enforce(
		t.Context(),
		h.request(platformcatalog.MeterDocumentFileBytes, platformplan.FreeDemoDocumentFileBytes+1),
	)
	var exceeded *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &exceeded)
	assert.Equal(t, platformplan.FreeDemoDocumentFileBytes+1, exceeded.Used)
}

func TestEnforce_MonthlyWindowUsesTheOrganizationTimezone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterAIAssistantMessages).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, h.tenant, platformcatalog.MeterAIAssistantMessages).Return(nil).Once()
	h.counters.EXPECT().OrganizationTimezone(mock.Anything, h.tenant).Return("America/Chicago", nil).Once()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	h.counters.EXPECT().
		Count(mock.Anything, &repositories.QuotaCountRequest{
			TenantInfo:  h.tenant,
			Meter:       platformcatalog.MeterAIAssistantMessages,
			WindowStart: time.Date(2026, time.September, 1, 0, 0, 0, 0, chicago).Unix(),
			WindowEnd:   time.Date(2026, time.October, 1, 0, 0, 0, 0, chicago).Unix(),
		}).
		Return(int64(24), nil).
		Once()

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterAIAssistantMessages, 1)))
}

func TestEnforce_UnlimitedOrganizationsSkipEverything(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.inTx = false
	h.resolves(platformplan.NewUnmanaged(
		platformplan.Unlimited(),
		platformplan.OriginInternal,
		h.tenant.OrgID,
		h.tenant.BuID,
		fixedNow.Unix(),
	))

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 500)))
}

func TestEnforce_UnmeteredMeterAndZeroQuantityPass(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.inTx = false
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterShipmentsTotal).Return(true)

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterAPIRequests, 5)))
	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 0)))
}

func TestEnforce_RefusesReadOnlyAndExpiredOrganizations(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusReadOnly))

	err := h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 1))
	var restricted *errortypes.PlanRestrictionError
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.PlanRestrictionReasonReadOnly, restricted.Reason)

	expired := newHarness(t)
	expired.resolves(expired.freeDemo(t, subscription.StatusExpired))
	err = expired.guard.Enforce(t.Context(), expired.request(platformcatalog.MeterShipmentsTotal, 1))
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.PlanRestrictionReasonExpired, restricted.Reason)
}

func TestEnforce_ValidatesTheRequest(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	require.ErrorIs(t, h.guard.Enforce(t.Context(), nil), quotaservice.ErrRequestRequired)
	require.ErrorIs(
		t,
		h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, -1)),
		quotaservice.ErrInvalidQuantity,
	)
	require.ErrorIs(
		t,
		h.guard.Enforce(t.Context(), &services.QuotaRequest{Meter: platformcatalog.MeterShipmentsTotal}),
		quotaservice.ErrTenantRequired,
	)
}

func TestEnforce_MissingCounterIsAnError(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterTrailersTotal).Return(false)

	err := h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterTrailersTotal, 1))
	require.ErrorIs(t, err, quotaservice.ErrCounterMissing)
}

func TestEnforce_PropagatesFailures(t *testing.T) {
	t.Parallel()

	boom := errors.New("lock timeout")

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterWorkersTotal).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, mock.Anything, mock.Anything).Return(boom).Once()
	require.ErrorIs(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterWorkersTotal, 1)), boom)

	failing := newHarness(t)
	failing.plans.EXPECT().Resolve(mock.Anything, mock.Anything, mock.Anything).Return(nil, boom)
	require.ErrorIs(
		t,
		failing.guard.Enforce(t.Context(), failing.request(platformcatalog.MeterWorkersTotal, 1)),
		boom,
	)
}

func TestCheck_ReportsTheDecisionWithoutLocking(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.inTx = false
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterDocumentUploads).Return(true)
	h.counters.EXPECT().Count(mock.Anything, mock.Anything).Return(int64(25), nil).Once()

	decision, err := h.guard.Check(t.Context(), h.request(platformcatalog.MeterDocumentUploads, 1))
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.False(t, decision.Unlimited)
	assert.Equal(t, int64(25), decision.Limit)
	assert.Equal(t, int64(25), decision.Used)
	assert.Zero(t, decision.Remaining)
	assert.Equal(t, platformplan.PlanKeyFreeDemo, decision.Plan)
	assert.Equal(t, platformplan.WindowLifetime, decision.Window)
}

func TestCheck_PerItemAndUnlimited(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))

	decision, err := h.guard.Check(t.Context(), h.request(platformcatalog.MeterDocumentFileBytes, 1024))
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, platformplan.FreeDemoDocumentFileBytes-1024, decision.Remaining)

	decision, err = h.guard.Check(t.Context(), h.request(platformcatalog.MeterAPIRequests, 1))
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.True(t, decision.Unlimited)
}

func TestUsage_ReportsEveryMeteredLimit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	resolved := h.freeDemo(t, subscription.StatusTrialing)
	h.resolves(resolved)
	h.counters.EXPECT().OrganizationTimezone(mock.Anything, h.tenant).Return("UTC", nil).Once()
	h.counters.EXPECT().
		Count(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.QuotaCountRequest) (int64, error) {
			switch req.Meter {
			case platformcatalog.MeterShipmentsTotal:
				return 15, nil
			case platformcatalog.MeterAISpendCents:
				assert.Equal(t, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC).Unix(), req.WindowStart)
				assert.Equal(t, time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC).Unix(), req.WindowEnd)
				return 40, nil
			default:
				return 1, nil
			}
		})

	summary, err := h.guard.Usage(t.Context(), h.tenant)
	require.NoError(t, err)

	assert.Equal(t, platformplan.PlanKeyFreeDemo, summary.Plan)
	assert.Equal(t, "Free demo", summary.PlanName)
	assert.Equal(t, platformplan.OriginSubscription, summary.Origin)
	assert.False(t, summary.Unlimited)
	assert.Equal(t, subscription.StatusTrialing, summary.Status)
	assert.Equal(t, resolved.Subscription.TrialEndsAt, summary.TrialEndsAt)
	assert.Equal(t, resolved.Subscription.ReadOnlyUntil, summary.ReadOnlyUntil)
	assert.Len(t, summary.RestrictedCapabilities, len(platformplan.AllCapabilities()))
	assert.Len(t, summary.Meters, 13)

	byMeter := make(map[platformcatalog.MeterKey]services.QuotaMeterUsage, len(summary.Meters))
	for _, usage := range summary.Meters {
		byMeter[usage.Meter] = usage
	}

	assert.Equal(t, int64(15), byMeter[platformcatalog.MeterShipmentsTotal].Used)
	assert.Zero(t, byMeter[platformcatalog.MeterShipmentsTotal].Remaining, "over the limit never goes negative")
	assert.Equal(t, int64(110), byMeter[platformcatalog.MeterAISpendCents].Remaining)
	fileBytes := byMeter[platformcatalog.MeterDocumentFileBytes]
	assert.Equal(t, platformplan.WindowPerItem, fileBytes.Window)
	assert.Zero(t, fileBytes.Used)
	assert.Equal(t, platformplan.FreeDemoDocumentFileBytes, fileBytes.Remaining)
}

func TestDecorateReplacesTheDefaultOnlyInCloudMode(t *testing.T) {
	t.Parallel()

	fallback := quotaservice.NewUnlimited()
	assert.Same(t, fallback, cloudquota.Decorate(cloudquota.DecorateParams{
		Default: fallback,
		Config:  &config.Config{},
		Logger:  zap.NewNop(),
	}))
	assert.IsType(t, &cloudquota.CloudGuard{}, cloudquota.Decorate(cloudquota.DecorateParams{
		Default:  fallback,
		Config:   &config.Config{Platform: config.PlatformConfig{Mode: config.PlatformModeCloud}},
		Plans:    mocks.NewMockPlanService(t),
		Counters: mocks.NewMockQuotaCounterRepository(t),
		Logger:   zap.NewNop(),
	}))
}

func TestEnforce_UsingUpShipmentsEndsTheTrial(t *testing.T) {
	t.Parallel()

	h, subs := newHarnessWithSubscriptions(t)
	resolved := h.freeDemo(t, subscription.StatusTrialing)
	h.resolves(resolved)
	h.expectShipmentCount(11)
	subs.EXPECT().
		EndTrial(mock.Anything, &repositories.EndSubscriptionTrialRequest{
			TenantInfo: h.tenant,
			ID:         resolved.Subscription.ID,
			EndedAt:    fixedNow.Unix(),
		}).
		Return(true, nil).
		Once()
	h.plans.EXPECT().Invalidate(h.tenant.OrgID).Once()

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 1)))
}

func TestEnforce_ShipmentsBelowTheLimitKeepTheTrial(t *testing.T) {
	t.Parallel()

	h, _ := newHarnessWithSubscriptions(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.expectShipmentCount(10)

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 1)))
}

func TestEnforce_OtherMetersAtTheirLimitKeepTheTrial(t *testing.T) {
	t.Parallel()

	h, _ := newHarnessWithSubscriptions(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.counters.EXPECT().Supports(platformcatalog.MeterTrailersTotal).Return(true)
	h.counters.EXPECT().Lock(mock.Anything, h.tenant, platformcatalog.MeterTrailersTotal).Return(nil).Once()
	h.counters.EXPECT().Count(mock.Anything, mock.Anything).Return(int64(2), nil).Once()

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterTrailersTotal, 1)))
}

func TestEnforce_TrialAlreadyEndedIsLeftAlone(t *testing.T) {
	t.Parallel()

	h, subs := newHarnessWithSubscriptions(t)
	resolved := h.freeDemo(t, subscription.StatusTrialing)
	h.resolves(resolved)
	h.expectShipmentCount(11)
	subs.EXPECT().EndTrial(mock.Anything, mock.Anything).Return(false, nil).Once()

	require.NoError(t, h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 1)))
}

func TestEnforce_EndingTheTrialFailureFailsTheWrite(t *testing.T) {
	t.Parallel()

	h, subs := newHarnessWithSubscriptions(t)
	h.resolves(h.freeDemo(t, subscription.StatusTrialing))
	h.expectShipmentCount(11)
	subs.EXPECT().EndTrial(mock.Anything, mock.Anything).Return(false, errors.New("boom")).Once()

	err := h.guard.Enforce(t.Context(), h.request(platformcatalog.MeterShipmentsTotal, 1))
	require.ErrorContains(t, err, "end the trial when shipments.total was used up")
}
