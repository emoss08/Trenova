package plantest

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func FreeDemo(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	status subscription.Status,
) *platformplan.ResolvedPlan {
	t.Helper()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	now := time.Now()
	sub := &subscription.Subscription{
		ID:             pulid.MustNew("osub_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         status,
		TrialEndsAt:    now.Add(24 * time.Hour).Unix(),
		ReadOnlyUntil:  now.Add(48 * time.Hour).Unix(),
		CreatedAt:      now.Add(-24 * time.Hour).Unix(),
	}

	return platformplan.NewManaged(plan, sub, now.Unix())
}

func Unmanaged(tenantInfo pagination.TenantInfo) *platformplan.ResolvedPlan {
	return platformplan.NewUnmanaged(
		platformplan.Unlimited(),
		platformplan.OriginInternal,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		time.Now().Unix(),
	)
}

func Restriction(capability platformplan.Capability) error {
	return errortypes.NewPlanRestrictionError(
		capability.String(),
		platformplan.ReasonPlanRestricted,
		platformplan.PlanKeyFreeDemo.String(),
	)
}

func Restricting(t *testing.T, capability platformplan.Capability) *mocks.MockPlanService {
	t.Helper()

	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().
		RequireCapability(mock.Anything, mock.Anything, capability).
		Return(Restriction(capability))

	return plans
}

func Allowing(t *testing.T, capability platformplan.Capability) *mocks.MockPlanService {
	t.Helper()

	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().
		RequireCapability(mock.Anything, mock.Anything, capability).
		Return(nil)

	return plans
}

func Cloud(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	resolved *platformplan.ResolvedPlan,
) *mocks.MockPlanService {
	t.Helper()

	plans := mocks.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true).Maybe()
	plans.EXPECT().
		Resolve(mock.Anything, tenantInfo.OrgID, tenantInfo.BuID).
		Return(resolved, nil).
		Maybe()

	return plans
}
