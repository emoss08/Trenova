package planservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func requireReason(t *testing.T, err error, reason, capability string) {
	t.Helper()

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, reason, restriction.Reason)
	assert.Equal(t, capability, restriction.Capability)
}

func TestAdmitIgnoresNonCloudAndUnscopedRequests(t *testing.T) {
	t.Parallel()

	require.NoError(t, planservice.Admit(t.Context(), nil, &planservice.AdmissionRequest{Write: true}))
	require.NoError(t, planservice.Admit(
		t.Context(),
		planservice.NewUnlimited(),
		&planservice.AdmissionRequest{TenantInfo: plantest.Tenant(), Write: true},
	))

	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true)
	require.NoError(t, planservice.Admit(
		t.Context(),
		plans,
		&planservice.AdmissionRequest{TenantInfo: pagination.TenantInfo{}, Write: true},
	))
}

func TestAdmitLetsUnmanagedOrganizationsThrough(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.Unmanaged(tenant))

	require.NoError(t, planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		Write:      true,
		APIKey:     true,
	}))
}

func TestAdmitRefusesWritesWhileReadOnly(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly))

	err := planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		Write:      true,
	})
	requireReason(t, err, platformplan.ReasonSubscriptionReadOnly, "")

	require.NoError(t, planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
	}))
}

func TestAdmitRefusesEverythingOnceExpiredUnlessAllowed(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusExpired))

	err := planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{TenantInfo: tenant})
	requireReason(t, err, platformplan.ReasonSubscriptionExpired, "")

	require.NoError(t, planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		ExpiredOK:  true,
	}))

	err = planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		ExpiredOK:  true,
		Write:      true,
	})
	requireReason(t, err, platformplan.ReasonSubscriptionExpired, "")
}

func TestAdmitRefusesAPIKeysWhenThePlanRestrictsThem(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing))

	err := planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		APIKey:     true,
	})
	requireReason(
		t,
		err,
		platformplan.ReasonPlanRestricted,
		platformplan.CapabilityAPIKeys.String(),
	)

	require.NoError(t, planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{
		TenantInfo: tenant,
		Write:      true,
	}))
}

func TestAdmitReturnsResolutionErrors(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	boom := errors.New("database unavailable")
	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true)
	plans.EXPECT().Resolve(mock.Anything, tenant.OrgID, tenant.BuID).Return(nil, boom)

	err := planservice.Admit(t.Context(), plans, &planservice.AdmissionRequest{TenantInfo: tenant})
	require.ErrorIs(t, err, boom)
}

func TestRequireCapabilityHelperToleratesNoPlanService(t *testing.T) {
	t.Parallel()

	require.NoError(t, planservice.RequireCapability(
		t.Context(),
		nil,
		plantest.Tenant(),
		platformplan.CapabilitySMS,
	))

	plans := plantest.Restricting(t, platformplan.CapabilitySMS)
	err := planservice.RequireCapability(
		t.Context(),
		plans,
		plantest.Tenant(),
		platformplan.CapabilitySMS,
	)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestAllowsSeparatesRestrictionsFromFailures(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()

	allowed, err := planservice.Allows(t.Context(), nil, tenant, platformplan.CapabilitySSO)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = planservice.Allows(
		t.Context(),
		plantest.Restricting(t, platformplan.CapabilitySSO),
		tenant,
		platformplan.CapabilitySSO,
	)
	require.NoError(t, err)
	assert.False(t, allowed)

	boom := errors.New("resolve failed")
	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().
		RequireCapability(mock.Anything, tenant, platformplan.CapabilitySSO).
		Return(boom)
	allowed, err = planservice.Allows(t.Context(), plans, tenant, platformplan.CapabilitySSO)
	require.ErrorIs(t, err, boom)
	assert.False(t, allowed)
}

func TestRequireWritableHelperDelegates(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	require.NoError(t, planservice.RequireWritable(t.Context(), nil, tenant))

	refusal := errortypes.NewPlanRestrictionError("", platformplan.ReasonSubscriptionReadOnly, "free_demo")
	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().RequireWritable(mock.Anything, tenant).Return(refusal)
	require.ErrorIs(t, planservice.RequireWritable(t.Context(), plans, tenant), refusal)
}

func TestOrUnlimitedFallsBackToUnlimitedPlans(t *testing.T) {
	t.Parallel()

	assert.IsType(t, &planservice.UnlimitedPlanService{}, planservice.OrUnlimited(nil))

	plans := mocks.NewMockPlanService(t)
	assert.Same(t, plans, planservice.OrUnlimited(plans))
}
