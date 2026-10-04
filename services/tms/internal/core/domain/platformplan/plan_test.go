package platformplan_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeDemoDefaults(t *testing.T) {
	t.Parallel()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	assert.Equal(t, platformplan.PlanKeyFreeDemo, plan.Key)
	assert.Len(t, plan.Limits, 13)

	expected := map[platformcatalog.MeterKey]platformplan.Limit{
		platformcatalog.MeterShipmentsTotal:          {Max: 12, Window: platformplan.WindowLifetime},
		platformcatalog.MeterRecurringShipmentSeries: {Max: 1, Window: platformplan.WindowLifetime},
		platformcatalog.MeterCustomersTotal:          {Max: 8, Window: platformplan.WindowLifetime},
		platformcatalog.MeterLocationsTotal:          {Max: 25, Window: platformplan.WindowLifetime},
		platformcatalog.MeterWorkersTotal:            {Max: 3, Window: platformplan.WindowLifetime},
		platformcatalog.MeterTractorsTotal:           {Max: 3, Window: platformplan.WindowLifetime},
		platformcatalog.MeterTrailersTotal:           {Max: 3, Window: platformplan.WindowLifetime},
		platformcatalog.MeterUserSeats:               {Max: 1, Window: platformplan.WindowLifetime},
		platformcatalog.MeterDocumentUploads:         {Max: 25, Window: platformplan.WindowLifetime},
		platformcatalog.MeterDocumentStorageBytes:    {Max: 104857600, Window: platformplan.WindowLifetime},
		platformcatalog.MeterDocumentFileBytes:       {Max: 10485760, Window: platformplan.WindowPerItem},
		platformcatalog.MeterAIAssistantMessages:     {Max: 25, Window: platformplan.WindowMonthly},
		platformcatalog.MeterAISpendCents:            {Max: 150, Window: platformplan.WindowMonthly},
	}
	assert.Equal(t, expected, plan.Limits)

	for _, capability := range platformplan.AllCapabilities() {
		assert.True(t, plan.Restricts(capability), "free demo must restrict %s", capability)
		assert.False(t, plan.Allows(capability))
	}
	assert.False(t, plan.IsUnlimited())
}

func TestFreeDemoAppliesOverrides(t *testing.T) {
	t.Parallel()

	plan, err := platformplan.FreeDemo(map[platformcatalog.MeterKey]int64{
		platformcatalog.MeterShipmentsTotal: 40,
		platformcatalog.MeterUserSeats:      0,
	})
	require.NoError(t, err)

	shipments, ok := plan.Limit(platformcatalog.MeterShipmentsTotal)
	require.True(t, ok)
	assert.Equal(t, int64(40), shipments.Max)
	assert.Equal(t, platformplan.WindowLifetime, shipments.Window)

	seats, ok := plan.Limit(platformcatalog.MeterUserSeats)
	require.True(t, ok)
	assert.Zero(t, seats.Max)

	defaults, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)
	untouched, _ := defaults.Limit(platformcatalog.MeterShipmentsTotal)
	assert.Equal(t, int64(12), untouched.Max, "overrides must not leak into other plans")
}

func TestFreeDemoRejectsBadOverrides(t *testing.T) {
	t.Parallel()

	_, err := platformplan.FreeDemo(map[platformcatalog.MeterKey]int64{
		platformcatalog.MeterAPIRequests: 10,
	})
	require.ErrorIs(t, err, platformplan.ErrUnknownMeterOverride)

	_, err = platformplan.FreeDemo(map[platformcatalog.MeterKey]int64{
		platformcatalog.MeterShipmentsTotal: -1,
	})
	require.ErrorIs(t, err, platformplan.ErrNegativeLimit)
}

func TestParseLimitOverrides(t *testing.T) {
	t.Parallel()

	overrides := platformplan.ParseLimitOverrides(map[string]int64{"shipments.total": 4})
	assert.Equal(t, map[platformcatalog.MeterKey]int64{platformcatalog.MeterShipmentsTotal: 4}, overrides)
}

func TestUnlimited(t *testing.T) {
	t.Parallel()

	plan := platformplan.Unlimited()
	assert.True(t, plan.IsUnlimited())
	assert.Empty(t, plan.MeterKeys())
	for _, capability := range platformplan.AllCapabilities() {
		assert.True(t, plan.Allows(capability))
	}
	_, ok := plan.Limit(platformcatalog.MeterShipmentsTotal)
	assert.False(t, ok)

	var nilPlan *platformplan.Plan
	assert.True(t, nilPlan.IsUnlimited())
	assert.True(t, nilPlan.Allows(platformplan.CapabilitySSO))
}

func TestCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := platformplan.NewCatalog(nil)
	require.NoError(t, err)

	freeDemo, err := catalog.Get(platformplan.PlanKeyFreeDemo)
	require.NoError(t, err)
	assert.Same(t, catalog.FreeDemo(), freeDemo)

	unlimited, err := catalog.Get(platformplan.PlanKeyUnlimited)
	require.NoError(t, err)
	assert.Same(t, catalog.Unlimited(), unlimited)

	_, err = catalog.Get(platformplan.PlanKey("enterprise_gold"))
	require.ErrorIs(t, err, platformplan.ErrUnknownPlan)

	_, err = platformplan.NewCatalog(map[platformcatalog.MeterKey]int64{"bogus.meter": 1})
	require.ErrorIs(t, err, platformplan.ErrUnknownMeterOverride)
}

func TestEnumsAreValid(t *testing.T) {
	t.Parallel()

	for _, capability := range platformplan.AllCapabilities() {
		assert.True(t, capability.IsValid())
	}
	for _, window := range platformplan.AllWindows() {
		assert.True(t, window.IsValid())
	}
	for _, key := range platformplan.AllPlanKeys() {
		assert.True(t, key.IsValid())
	}
	assert.False(t, platformplan.Capability("teleport").IsValid())
	assert.False(t, platformplan.Window("hourly").IsValid())
	assert.False(t, platformplan.PlanKey("gold").IsValid())
}

func TestResolvedPlan(t *testing.T) {
	t.Parallel()

	freeDemo, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	sub := &subscription.Subscription{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    1_000,
		ReadOnlyUntil:  2_000,
	}

	trialing := platformplan.NewManaged(freeDemo, sub, 500)
	assert.True(t, trialing.IsManaged())
	assert.Equal(t, platformplan.PlanKeyFreeDemo, trialing.Key())
	assert.Equal(t, subscription.StatusTrialing, trialing.Status)
	assert.True(t, trialing.AllowsWrites())
	assert.True(t, trialing.AllowsLogin())
	assert.False(t, trialing.Allows(platformplan.CapabilityAPIKeys))

	readOnly := platformplan.NewManaged(freeDemo, sub, 1_500)
	assert.Equal(t, subscription.StatusReadOnly, readOnly.Status)
	assert.False(t, readOnly.AllowsWrites())
	assert.True(t, readOnly.AllowsLogin())

	expired := platformplan.NewManaged(freeDemo, sub, 2_000)
	assert.Equal(t, subscription.StatusExpired, expired.Status)
	assert.False(t, expired.AllowsWrites())
	assert.False(t, expired.AllowsLogin())

	internal := platformplan.NewUnmanaged(
		platformplan.Unlimited(),
		platformplan.OriginInternal,
		orgID,
		buID,
		500,
	)
	assert.False(t, internal.IsManaged())
	assert.True(t, internal.AllowsWrites())
	assert.True(t, internal.AllowsLogin())
	assert.True(t, internal.Allows(platformplan.CapabilitySSO))
	assert.Equal(t, platformplan.PlanKeyUnlimited, internal.Key())

	var none *platformplan.ResolvedPlan
	assert.Equal(t, platformplan.PlanKeyUnlimited, none.Key())
	assert.True(t, none.Allows(platformplan.CapabilitySMS))
	assert.True(t, none.AllowsWrites())
}

func TestTrialEndingMeters(t *testing.T) {
	t.Parallel()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	assert.True(t, plan.EndsTrial(platformcatalog.MeterShipmentsTotal))
	assert.False(t, plan.EndsTrial(platformcatalog.MeterTrailersTotal))
	assert.False(t, platformplan.Unlimited().EndsTrial(platformcatalog.MeterShipmentsTotal))

	var nilPlan *platformplan.Plan
	assert.False(t, nilPlan.EndsTrial(platformcatalog.MeterShipmentsTotal))
}
