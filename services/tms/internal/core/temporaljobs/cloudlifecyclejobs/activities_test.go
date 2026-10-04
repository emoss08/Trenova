package cloudlifecyclejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/cloudlifecycleservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scriptedPurge struct {
	passes  []*repositories.PurgeTenantRowsResult
	calls   int
	members []*repositories.TenantMember
}

func (s *scriptedPurge) ListMembers(
	context.Context,
	pagination.TenantInfo,
) ([]*repositories.TenantMember, error) {
	return s.members, nil
}

func (s *scriptedPurge) OrganizationProfile(
	context.Context,
	pagination.TenantInfo,
) (*repositories.TenantProfile, error) {
	return &repositories.TenantProfile{}, nil
}

func (s *scriptedPurge) PurgeRows(
	context.Context,
	*repositories.PurgeTenantRowsRequest,
) (*repositories.PurgeTenantRowsResult, error) {
	pass := s.passes[min(s.calls, len(s.passes)-1)]
	s.calls++
	return pass, nil
}

func (s *scriptedPurge) PurgeUser(
	context.Context,
	*repositories.PurgeTenantUserRequest,
) (*repositories.PurgeTenantUserResult, error) {
	return &repositories.PurgeTenantUserResult{}, nil
}

func (s *scriptedPurge) DeleteTenant(
	context.Context,
	pagination.TenantInfo,
) (*repositories.DeleteTenantResult, error) {
	return &repositories.DeleteTenantResult{}, nil
}

func newActivities(
	t *testing.T,
	purge *scriptedPurge,
	subs *mocks.MockSubscriptionRepository,
) *Activities {
	t.Helper()

	return NewActivities(ActivitiesParams{
		Lifecycle: cloudlifecycleservice.New(cloudlifecycleservice.Params{
			Subscriptions: subs,
			Purge:         purge,
			Plans:         mocks.NewMockPlanService(t),
			Logger:        zap.NewNop(),
		}),
		Logger: zap.NewNop(),
	})
}

func purgePayload() *PurgePayload {
	return &PurgePayload{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: pulid.MustNew("bu_")}
}

func TestPurgeRowsActivityRepeatsUntilTheRowsAreGone(t *testing.T) {
	t.Parallel()

	purge := &scriptedPurge{passes: []*repositories.PurgeTenantRowsResult{
		{Deleted: 500},
		{Deleted: 120, Complete: true, Retained: []string{"ai_logs"}},
	}}
	a := newActivities(t, purge, mocks.NewMockSubscriptionRepository(t))

	result, err := a.PurgeCloudTenantRowsActivity(t.Context(), purgePayload())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Passes)
	assert.Equal(t, int64(620), result.Deleted)
	assert.True(t, result.Complete)
	assert.Equal(t, []string{"ai_logs"}, result.Retained)
}

func TestPurgeRowsActivityStopsWhenAPassMakesNoProgress(t *testing.T) {
	t.Parallel()

	purge := &scriptedPurge{passes: []*repositories.PurgeTenantRowsResult{
		{Deleted: 10},
		{Deleted: 0, Blocked: []string{"customers"}},
	}}
	a := newActivities(t, purge, mocks.NewMockSubscriptionRepository(t))

	result, err := a.PurgeCloudTenantRowsActivity(t.Context(), purgePayload())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Passes)
	assert.False(t, result.Complete)
	assert.Equal(t, []string{"customers"}, result.Blocked)
}

func TestCheckActivityRefusesAnOrganizationThatHasNotExpired(t *testing.T) {
	t.Parallel()

	subs := mocks.NewMockSubscriptionRepository(t)
	subs.EXPECT().GetByOrganization(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Subscription"))
	a := newActivities(t, &scriptedPurge{}, subs)

	eligibility, err := a.CheckCloudTenantPurgeActivity(t.Context(), purgePayload())

	require.NoError(t, err)
	assert.False(t, eligibility.Eligible)
	assert.Empty(t, eligibility.Members)
}

func TestCheckActivityListsTheMembersOfAnExpiredOrganization(t *testing.T) {
	t.Parallel()

	payload := purgePayload()
	subs := mocks.NewMockSubscriptionRepository(t)
	subs.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(&subscription.Subscription{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		Status:         subscription.StatusExpired,
	}, nil)
	member := &repositories.TenantMember{UserID: pulid.MustNew("usr_")}
	a := newActivities(t, &scriptedPurge{members: []*repositories.TenantMember{member}}, subs)

	eligibility, err := a.CheckCloudTenantPurgeActivity(t.Context(), payload)

	require.NoError(t, err)
	assert.True(t, eligibility.Eligible)
	assert.Equal(t, []*repositories.TenantMember{member}, eligibility.Members)
}

func TestStorageActivitySkipsAnUnsupportedBackend(t *testing.T) {
	t.Parallel()

	a := newActivities(t, &scriptedPurge{}, mocks.NewMockSubscriptionRepository(t))

	result, err := a.PurgeCloudTenantStorageActivity(t.Context(), purgePayload())

	require.NoError(t, err)
	assert.True(t, result.Skipped)
}
