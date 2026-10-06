package authservice

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func withThrottle(t *testing.T, deps *testDeps) *mocks.MockLoginThrottleStore {
	t.Helper()
	store := mocks.NewMockLoginThrottleStore(t)
	deps.svc.throttle = store
	return store
}

func withPlans(t *testing.T, deps *testDeps) *mocks.MockPlanService {
	t.Helper()
	plans := mocks.NewMockPlanService(t)
	deps.svc.plans = plans
	return plans
}

func metaContext(t *testing.T) context.Context {
	t.Helper()
	return requestmeta.With(t.Context(), requestmeta.New("req-1", "203.0.113.7", "test-agent"))
}

func throttleKeyFor(email string) repositories.LoginThrottleKey {
	return repositories.LoginThrottleKey{Account: email, ClientIP: "203.0.113.7"}
}

func expectSuccessfulSession(deps *testDeps, usr *tenant.User) {
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)
}

func managedPlan(status subscription.Status, orgID, buID pulid.ID) *platformplan.ResolvedPlan {
	now := time.Now().Unix()
	sub := &subscription.Subscription{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         status,
		TrialEndsAt:    now + 3_600,
		ReadOnlyUntil:  now + 7_200,
	}
	return platformplan.NewManaged(platformplan.Unlimited(), sub, now)
}

func TestLoginRefusedWhileAccountIsThrottled(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)
	ctx := metaContext(t)

	store.EXPECT().Check(mock.Anything, throttleKeyFor("test@example.com")).Return(
		&repositories.LoginThrottleDecision{
			Blocked:    true,
			Scope:      repositories.LoginThrottleScopeAccount,
			RetryAfter: 90 * time.Second,
		}, nil,
	)

	_, err := deps.svc.Login(ctx, services.LoginRequest{
		EmailAddress: "Test@Example.com",
		Password:     "password123",
	})
	require.Error(t, err)
	assert.Equal(t, http.StatusTooManyRequests, errortypes.HTTPStatus(err))
	retryAfter, ok := errortypes.RetryAfterOf(err)
	require.True(t, ok)
	assert.Equal(t, 90*time.Second, retryAfter)

	rec := deps.authEvents.only(t)
	assert.Equal(t, services.AuthEventProviderLoginThrottled, rec.Provider)
	assert.Equal(t, iam.AuthEventOutcomeDenied, rec.Outcome)
	assert.Equal(t, authErrorThrottledAccount, rec.ErrorCode)
	assert.Equal(t, []string{riskSignalAccountThrottle}, rec.RiskSignals)
	deps.userRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything)
}

func TestLoginRefusedWhileIPIsThrottled(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)

	store.EXPECT().Check(mock.Anything, mock.Anything).Return(
		&repositories.LoginThrottleDecision{
			Blocked:    true,
			Scope:      repositories.LoginThrottleScopeIP,
			RetryAfter: time.Hour,
		}, nil,
	)

	_, err := deps.svc.Login(metaContext(t), services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	})
	require.True(t, errortypes.IsRateLimitError(err))
	assert.Equal(t, authErrorThrottledIP, deps.authEvents.only(t).ErrorCode)
}

func TestLoginFailureIsCountedAndTripsTheThrottle(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)
	usr := newTestUser(t)
	key := throttleKeyFor(usr.EmailAddress)

	store.EXPECT().Check(mock.Anything, key).Return(&repositories.LoginThrottleDecision{}, nil)
	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	store.EXPECT().RecordFailure(mock.Anything, key).Return(&repositories.LoginThrottleDecision{
		Blocked:         true,
		Scope:           repositories.LoginThrottleScopeAccount,
		RetryAfter:      30 * time.Second,
		AccountFailures: 5,
	}, nil)

	_, err := deps.svc.Login(metaContext(t), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "wrong-password",
	})
	require.ErrorIs(t, err, errInvalidCredentials)

	deps.authEvents.mu.Lock()
	defer deps.authEvents.mu.Unlock()
	require.Len(t, deps.authEvents.records, 2)
	assert.Equal(t, services.AuthEventProviderLoginThrottled, deps.authEvents.records[0].Provider)
	assert.Equal(t, authErrorRejectedLogin, deps.authEvents.records[1].ErrorCode)
}

func TestLoginUnknownAccountIsCounted(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)
	key := throttleKeyFor("ghost@example.com")

	store.EXPECT().Check(mock.Anything, key).Return(&repositories.LoginThrottleDecision{}, nil)
	deps.userRepo.On("FindByEmail", mock.Anything, "ghost@example.com").
		Return(nil, errors.New("not found"))
	store.EXPECT().RecordFailure(mock.Anything, key).
		Return(&repositories.LoginThrottleDecision{AccountFailures: 1}, nil)

	_, err := deps.svc.Login(metaContext(t), services.LoginRequest{
		EmailAddress: "ghost@example.com",
		Password:     "password123",
	})
	require.ErrorIs(t, err, errInvalidCredentials)
	assert.Equal(t, authErrorUnknownAccount, deps.authEvents.only(t).ErrorCode)
}

func TestLoginSuccessClearsTheAccountCounter(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)
	usr := newTestUser(t)
	key := throttleKeyFor(usr.EmailAddress)

	store.EXPECT().Check(mock.Anything, key).Return(&repositories.LoginThrottleDecision{}, nil)
	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	store.EXPECT().Reset(mock.Anything, usr.EmailAddress).Return(nil)
	expectSuccessfulSession(deps, usr)

	_, err := deps.svc.Login(metaContext(t), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
}

func TestLoginThrottleFailsOpenWhenRedisIsDown(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	store := withThrottle(t, deps)
	usr := newTestUser(t)

	store.EXPECT().Check(mock.Anything, mock.Anything).Return(nil, errors.New("redis down"))
	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	store.EXPECT().Reset(mock.Anything, mock.Anything).Return(errors.New("redis down"))
	expectSuccessfulSession(deps, usr)

	_, err := deps.svc.Login(metaContext(t), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
}

func TestLoginRefusedWhenTheOnlySubscriptionExpired(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	plans := withPlans(t, deps)
	usr := newTestUser(t)

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	plans.EXPECT().EnforcesPlans().Return(true)
	plans.EXPECT().Resolve(mock.Anything, usr.CurrentOrganizationID, usr.BusinessUnitID).
		Return(managedPlan(subscription.StatusExpired, usr.CurrentOrganizationID, usr.BusinessUnitID), nil)
	deps.userRepo.On("GetOrganizations", mock.Anything, usr.ID).Return(
		[]*tenant.OrganizationMembership{{
			UserID:         usr.ID,
			OrganizationID: usr.CurrentOrganizationID,
			BusinessUnitID: usr.BusinessUnitID,
		}}, nil,
	)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.ErrorIs(t, err, errSubscriptionExpired)
	rec := deps.authEvents.only(t)
	assert.Equal(t, authErrorSubscriptionExpired, rec.ErrorCode)
	assert.Equal(t, iam.AuthEventOutcomeDenied, rec.Outcome)
}

func TestLoginSwitchesAwayFromAnExpiredOrganization(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	plans := withPlans(t, deps)
	usr := newTestUser(t)
	otherOrg := pulid.MustNew("org_")
	otherBU := pulid.MustNew("bu_")
	expiredOrg := usr.CurrentOrganizationID

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	plans.EXPECT().EnforcesPlans().Return(true)
	plans.EXPECT().Resolve(mock.Anything, expiredOrg, usr.BusinessUnitID).
		Return(managedPlan(subscription.StatusExpired, expiredOrg, usr.BusinessUnitID), nil)
	plans.EXPECT().Resolve(mock.Anything, otherOrg, otherBU).Return(
		platformplan.NewUnmanaged(
			platformplan.Unlimited(),
			platformplan.OriginInternal,
			otherOrg,
			otherBU,
			time.Now().Unix(),
		), nil,
	)
	deps.userRepo.On("GetOrganizations", mock.Anything, usr.ID).Return(
		[]*tenant.OrganizationMembership{
			{OrganizationID: expiredOrg, BusinessUnitID: usr.BusinessUnitID},
			{OrganizationID: otherOrg, BusinessUnitID: otherBU},
		}, nil,
	)
	deps.userRepo.On("UpdateCurrentOrganization", mock.Anything, usr.ID, otherOrg, otherBU).Return(nil)
	expectSuccessfulSession(deps, usr)

	resp, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, otherOrg, resp.User.CurrentOrganizationID)
	assert.Equal(t, otherBU, resp.User.BusinessUnitID)
}

func TestLoginAllowedForReadOnlySubscription(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	plans := withPlans(t, deps)
	usr := newTestUser(t)

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	plans.EXPECT().EnforcesPlans().Return(true)
	plans.EXPECT().Resolve(mock.Anything, usr.CurrentOrganizationID, usr.BusinessUnitID).
		Return(managedPlan(subscription.StatusReadOnly, usr.CurrentOrganizationID, usr.BusinessUnitID), nil)
	expectSuccessfulSession(deps, usr)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
}

func TestLoginSkipsPlanCheckOutsideCloud(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	plans := withPlans(t, deps)
	usr := newTestUser(t)

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	plans.EXPECT().EnforcesPlans().Return(false)
	expectSuccessfulSession(deps, usr)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
}

func TestCreateSessionForUser(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	expectSuccessfulSession(deps, usr)

	resp, err := deps.svc.CreateSessionForUser(t.Context(), &services.CreateSessionForUserRequest{
		User: usr,
	})
	require.NoError(t, err)
	assert.Equal(t, usr, resp.User)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.SessionToken)
	assert.Equal(t, services.AuthEventProviderPassword, resp.AuthProvider)

	rec := deps.authEvents.only(t)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, rec.Outcome)
	assert.Equal(t, usr.ID, rec.UserID)
}

func TestCreateSessionForUserRefusesInactiveUsers(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	usr.Status = domaintypes.StatusInactive

	_, err := deps.svc.CreateSessionForUser(t.Context(), &services.CreateSessionForUserRequest{
		User: usr,
	})
	require.Error(t, err)
	assert.Equal(t, authErrorAccountUnavailable, deps.authEvents.only(t).ErrorCode)

	_, err = deps.svc.CreateSessionForUser(t.Context(), nil)
	require.True(t, errortypes.IsAuthenticationError(err))
}
