package planbilling

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/catalog"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRegistry(t *testing.T) *catalog.Registry {
	t.Helper()

	registry, err := catalog.NewRegistry(catalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{catalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	return registry
}

func TestLocalPlanBillingProvider_ReportsTheTrial(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	subID := pulid.MustNew("osub_")

	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().
		Usage(t.Context(), pagination.TenantInfo{OrgID: orgID, BuID: buID}).
		Return(&services.QuotaUsageSummary{
			OrganizationID:         orgID,
			BusinessUnitID:         buID,
			Plan:                   platformplan.PlanKeyFreeDemo,
			PlanName:               "Free demo",
			Origin:                 platformplan.OriginSubscription,
			SubscriptionID:         subID,
			Status:                 subscription.StatusTrialing,
			TrialEndsAt:            2_000,
			ReadOnlyUntil:          3_000,
			SubscribedAt:           1_000,
			RestrictedCapabilities: []platformplan.Capability{platformplan.CapabilityAPIKeys},
			Meters: []services.QuotaMeterUsage{{
				Meter:     platformcatalog.MeterShipmentsTotal,
				Window:    platformplan.WindowLifetime,
				Limit:     12,
				Used:      4,
				Remaining: 8,
			}},
		}, nil).
		Once()

	provider := NewLocalPlanBillingProvider(LocalPlanBillingProviderParams{
		Registry: newRegistry(t),
		Quota:    quota,
	})

	result, err := provider.GetBillingSummary(t.Context(), &services.BillingSummaryRequest{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CheckedAt:      1_500,
	})
	require.NoError(t, err)

	assert.True(t, result.Active)
	assert.Equal(t, "cloud_plan", result.Reason)
	assert.Equal(t, "free_demo", result.Plan.Key)
	assert.Equal(t, "Free demo", result.Plan.Name)
	assert.Equal(t, "trialing", result.Plan.Status)
	assert.Equal(t, subID.String(), result.Subscription.ID)
	assert.Equal(t, int64(1_000), result.Subscription.CurrentPeriodStart)
	assert.Equal(t, int64(2_000), result.Subscription.CurrentPeriodEnd)
	assert.Equal(t, int64(2_000), result.Subscription.TrialEndsAt)
	assert.Equal(t, int64(3_000), result.Subscription.ReadOnlyUntil)
	assert.Equal(t, []string{"api_keys"}, result.Restrictions)
	require.Len(t, result.Usage, 1)
	assert.Equal(t, int64(12), result.Usage[0].Limit)
	assert.Equal(t, int64(4), result.Usage[0].Used)
	assert.Equal(t, "shipment", result.Usage[0].Unit)
	assert.Equal(t, "lifetime", result.Usage[0].Window)
	assert.NotEmpty(t, result.Features)
	assert.Equal(t, int64(1_500), result.CheckedAt)
}

func TestLocalPlanBillingProvider_InternalOrganizationIsUnlimited(t *testing.T) {
	t.Parallel()

	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().
		Usage(t.Context(), pagination.TenantInfo{}).
		Return(&services.QuotaUsageSummary{
			Plan:      platformplan.PlanKeyUnlimited,
			PlanName:  "Unlimited",
			Origin:    platformplan.OriginInternal,
			Unlimited: true,
		}, nil).
		Once()

	registry := newRegistry(t)
	provider := NewLocalPlanBillingProvider(LocalPlanBillingProviderParams{Registry: registry, Quota: quota})

	result, err := provider.GetBillingSummary(t.Context(), &services.BillingSummaryRequest{})
	require.NoError(t, err)
	assert.True(t, result.Active)
	assert.Equal(t, "cloud_internal", result.Reason)
	assert.Equal(t, "active", result.Plan.Status)
	assert.Equal(t, "internal", result.Subscription.ID)
	assert.Len(t, result.Usage, len(registry.ListMeters()))
	assert.Empty(t, result.Restrictions)
}

func TestLocalPlanBillingProvider_ReadOnlyIsInactive(t *testing.T) {
	t.Parallel()

	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().
		Usage(t.Context(), pagination.TenantInfo{}).
		Return(&services.QuotaUsageSummary{
			Plan:   platformplan.PlanKeyFreeDemo,
			Origin: platformplan.OriginSubscription,
			Status: subscription.StatusReadOnly,
		}, nil).
		Once()

	provider := NewLocalPlanBillingProvider(LocalPlanBillingProviderParams{
		Registry: newRegistry(t),
		Quota:    quota,
	})

	result, err := provider.GetBillingSummary(t.Context(), &services.BillingSummaryRequest{})
	require.NoError(t, err)
	assert.False(t, result.Active)
	assert.Equal(t, "read_only", result.Subscription.Status)
}
