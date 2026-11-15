package planservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/subscription"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type featureGate struct {
	disabled map[platformcatalog.FeatureKey]bool
}

func (g featureGate) Enabled(_ context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	return !g.disabled[feature], nil
}

func allFeatures() featureGate {
	return featureGate{}
}

func withoutAPIKeys() featureGate {
	return featureGate{disabled: map[platformcatalog.FeatureKey]bool{
		platformcatalog.FeatureAPIKeys: true,
	}}
}

func requireRestriction(t *testing.T, err error, reason, feature, plan string) {
	t.Helper()

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, reason, restriction.Reason)
	assert.Equal(t, feature, restriction.Feature)
	assert.Empty(t, restriction.AddOn)
	assert.Equal(t, plan, restriction.Plan)
}

func TestAdmitIgnoresUnenforcedPlansAndUnscopedRequests(t *testing.T) {
	t.Parallel()

	require.NoError(t, planservice.Admit(
		t.Context(),
		planservice.NewUnlimited(),
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: plantest.Tenant(), Write: true, APIKey: true},
	))

	plans := platformplanportsmock.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true)
	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: pagination.TenantInfo{}, Write: true},
	))
}

func TestAdmitLetsUnmanagedOrganizationsThrough(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.Unmanaged(tenant))

	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant, Write: true, APIKey: true},
	))
}

func TestAdmitRefusesWritesWhileReadOnly(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly))

	err := planservice.Admit(t.Context(), plans, allFeatures(), &planservice.AdmissionRequest{
		TenantInfo: tenant,
		Write:      true,
	})
	requireRestriction(t, err, platformplan.ReasonSubscriptionReadOnly, "", "free_demo")
	assert.Equal(t, "This organization's trial has ended and it is now read-only", err.Error())

	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant},
	))
}

func TestAdmitRefusesEverythingOnceExpiredUnlessAllowed(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusExpired))

	err := planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant},
	)
	requireRestriction(t, err, platformplan.ReasonSubscriptionExpired, "", "free_demo")
	assert.Equal(t, "This organization's trial has expired", err.Error())

	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant, ExpiredOK: true},
	))

	err = planservice.Admit(t.Context(), plans, allFeatures(), &planservice.AdmissionRequest{
		TenantInfo: tenant,
		ExpiredOK:  true,
		Write:      true,
	})
	requireRestriction(t, err, platformplan.ReasonSubscriptionExpired, "", "free_demo")
}

func TestAdmitReportsExpiryBeforeAMissingAPIKeyFeature(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusExpired))

	err := planservice.Admit(t.Context(), plans, withoutAPIKeys(), &planservice.AdmissionRequest{
		TenantInfo: tenant,
		APIKey:     true,
	})
	requireRestriction(t, err, platformplan.ReasonSubscriptionExpired, "", "free_demo")
}

func TestAdmitRefusesAnAPIKeyWhenTheInstanceLacksTheFeature(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()

	err := planservice.Admit(
		t.Context(),
		planservice.NewUnlimited(),
		withoutAPIKeys(),
		&planservice.AdmissionRequest{TenantInfo: tenant, APIKey: true},
	)
	requireRestriction(t, err, errortypes.PlanRestrictionReasonPlan, "platform.api_keys", "")
	assert.Equal(t, "This feature is not available on your organization's plan", err.Error())

	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing))
	err = planservice.Admit(t.Context(), plans, withoutAPIKeys(), &planservice.AdmissionRequest{
		TenantInfo: tenant,
		APIKey:     true,
		Write:      true,
	})
	requireRestriction(t, err, errortypes.PlanRestrictionReasonPlan, "platform.api_keys", "")

	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		withoutAPIKeys(),
		&planservice.AdmissionRequest{TenantInfo: tenant, Write: true},
	))
}

func TestAdmitAllowsAnAPIKeyWhenTheInstanceHasTheFeature(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing))

	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant, APIKey: true, Write: true},
	))
}

func TestAdmitReturnsResolutionErrors(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	boom := errors.New("database unavailable")
	plans := platformplanportsmock.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true)
	plans.EXPECT().Resolve(mock.Anything, tenant.OrgID, tenant.BuID).Return(nil, boom)

	err := planservice.Admit(
		t.Context(),
		plans,
		allFeatures(),
		&planservice.AdmissionRequest{TenantInfo: tenant},
	)
	require.ErrorIs(t, err, boom)
}

func TestRequireWritableHelperDelegates(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	require.NoError(t, planservice.RequireWritable(t.Context(), nil, tenant))

	refusal := errortypes.NewPlanRestrictionError(
		platformplan.ReasonSubscriptionReadOnly,
		"free_demo",
	)
	plans := platformplanportsmock.NewMockPlanService(t)
	plans.EXPECT().RequireWritable(mock.Anything, tenant).Return(refusal)
	require.ErrorIs(t, planservice.RequireWritable(t.Context(), plans, tenant), refusal)
}

func TestOrUnlimitedFallsBackToUnlimitedPlans(t *testing.T) {
	t.Parallel()

	assert.IsType(t, &planservice.UnlimitedPlanService{}, planservice.OrUnlimited(nil))

	plans := platformplanportsmock.NewMockPlanService(t)
	assert.Same(t, plans, planservice.OrUnlimited(plans))
}
