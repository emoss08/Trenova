package cloudlifecycleservice_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/lifecycle/cloudlifecycleservice"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePurge struct {
	mu          sync.Mutex
	members     map[pulid.ID][]*repositories.TenantMember
	purgedUsers []pulid.ID
	userResults map[pulid.ID]*repositories.PurgeTenantUserResult
	deleted     []pulid.ID
	rowsResult  *repositories.PurgeTenantRowsResult
}

func (f *fakePurge) ListMembers(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*repositories.TenantMember, error) {
	return f.members[tenantInfo.OrgID], nil
}

func (f *fakePurge) OrganizationProfile(
	context.Context,
	pagination.TenantInfo,
) (*repositories.TenantProfile, error) {
	return &repositories.TenantProfile{Name: "Acme Freight", Timezone: "America/Chicago"}, nil
}

func (f *fakePurge) PurgeRows(
	context.Context,
	*repositories.PurgeTenantRowsRequest,
) (*repositories.PurgeTenantRowsResult, error) {
	return f.rowsResult, nil
}

func (f *fakePurge) PurgeUser(
	_ context.Context,
	req *repositories.PurgeTenantUserRequest,
) (*repositories.PurgeTenantUserResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.purgedUsers = append(f.purgedUsers, req.UserID)
	return f.userResults[req.UserID], nil
}

func (f *fakePurge) DeleteTenant(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.DeleteTenantResult, error) {
	f.deleted = append(f.deleted, tenantInfo.OrgID)
	return &repositories.DeleteTenantResult{OrganizationDeleted: true}, nil
}

type recordingEmails struct {
	services.PlatformEmailService

	mu         sync.Mutex
	trialEnded []*services.TrialEndedEmail
	purged     []*services.AccountPurgedEmail
}

func (e *recordingEmails) SendTrialEnded(_ context.Context, msg *services.TrialEndedEmail) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.trialEnded = append(e.trialEnded, msg)
	return nil
}

func (e *recordingEmails) SendAccountPurged(_ context.Context, msg *services.AccountPurgedEmail) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.purged = append(e.purged, msg)
	return nil
}

type prefixStorage struct {
	*mocks.MockClient

	prefixes []string
}

func (s *prefixStorage) DeletePrefix(_ context.Context, prefix string) (int64, error) {
	s.prefixes = append(s.prefixes, prefix)
	return 7, nil
}

type harness struct {
	subs     *mocks.MockSubscriptionRepository
	plans    *mocks.MockPlanService
	sessions *mocks.MockSessionRepository
	purge    *fakePurge
	emails   *recordingEmails
	storage  *prefixStorage
	svc      *cloudlifecycleservice.Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		subs:     mocks.NewMockSubscriptionRepository(t),
		plans:    mocks.NewMockPlanService(t),
		sessions: mocks.NewMockSessionRepository(t),
		purge: &fakePurge{
			members:     map[pulid.ID][]*repositories.TenantMember{},
			userResults: map[pulid.ID]*repositories.PurgeTenantUserResult{},
		},
		emails:  &recordingEmails{},
		storage: &prefixStorage{MockClient: mocks.NewMockClient(t)},
	}
	h.svc = cloudlifecycleservice.New(cloudlifecycleservice.Params{
		Subscriptions: h.subs,
		Purge:         h.purge,
		Plans:         h.plans,
		Sessions:      h.sessions,
		Storage:       h.storage,
		Emails:        h.emails,
		Logger:        zap.NewNop(),
	})

	return h
}

func newSubscription(status subscription.Status, trialEnds, readOnlyUntil int64) *subscription.Subscription {
	return &subscription.Subscription{
		ID:             pulid.MustNew("osub_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PlanKey:        "free_demo",
		Status:         status,
		TrialEndsAt:    trialEnds,
		ReadOnlyUntil:  readOnlyUntil,
		Version:        3,
	}
}

func (h *harness) expectTransition(sub *subscription.Subscription, status subscription.Status) {
	h.subs.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *repositories.UpdateSubscriptionStatusRequest) bool {
			return req.ID == sub.ID && req.Status == status
		})).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.UpdateSubscriptionStatusRequest,
		) (*subscription.Subscription, error) {
			updated := *sub
			updated.Status = req.Status
			updated.Version = req.Version + 1
			return &updated, nil
		}).
		Once()
	h.plans.EXPECT().Invalidate(sub.OrganizationID).Once()
}

func TestSweepMovesAnEndedTrialToReadOnlyAndTellsTheOwner(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := time.Now().Unix()
	sub := newSubscription(subscription.StatusTrialing, now-60, now+3_600)
	owner := &repositories.TenantMember{
		UserID:       pulid.MustNew("usr_"),
		Name:         "Owner",
		EmailAddress: "owner@example.com",
	}
	system := &repositories.TenantMember{
		UserID:       pulid.MustNew("usr_"),
		Username:     tenant.SystemUsername,
		EmailAddress: "system@example.com",
	}
	h.purge.members[sub.OrganizationID] = []*repositories.TenantMember{owner, system}

	h.subs.EXPECT().ListDue(mock.Anything, mock.Anything).Return([]*subscription.Subscription{sub}, nil)
	h.subs.EXPECT().ListExpired(mock.Anything, mock.Anything).Return(nil, nil)
	h.expectTransition(sub, subscription.StatusReadOnly)

	result, err := h.svc.Sweep(t.Context(), now)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ReadOnly)
	assert.Zero(t, result.Expired)
	assert.Empty(t, result.PurgeTargets)
	require.Len(t, h.emails.trialEnded, 1)
	email := h.emails.trialEnded[0]
	assert.Equal(t, owner.EmailAddress, email.To)
	assert.Equal(t, "Acme Freight", email.CompanyName)
	assert.Equal(t, "America/Chicago", email.Timezone)
	assert.Equal(t, sub.ReadOnlyUntil, email.ReadOnlyUntil)
	assert.Empty(t, h.emails.purged)
}

func TestSweepExpiresAPastGraceOrganizationEndsSessionsAndQueuesItsPurge(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := time.Now().Unix()
	sub := newSubscription(subscription.StatusTrialing, now-7_200, now-60)
	solo := &repositories.TenantMember{UserID: pulid.MustNew("usr_"), EmailAddress: "solo@example.com"}
	shared := &repositories.TenantMember{
		UserID:           pulid.MustNew("usr_"),
		EmailAddress:     "shared@example.com",
		OtherMemberships: 2,
	}
	h.purge.members[sub.OrganizationID] = []*repositories.TenantMember{solo, shared}

	h.subs.EXPECT().ListDue(mock.Anything, mock.Anything).Return([]*subscription.Subscription{sub}, nil)
	h.subs.EXPECT().ListExpired(mock.Anything, mock.Anything).Return(nil, nil)
	h.expectTransition(sub, subscription.StatusReadOnly)
	h.expectTransition(sub, subscription.StatusExpired)
	h.sessions.EXPECT().DeleteAllForUser(mock.Anything, solo.UserID).Return(nil).Once()

	result, err := h.svc.Sweep(t.Context(), now)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ReadOnly)
	assert.Equal(t, 1, result.Expired)
	require.Len(t, result.PurgeTargets, 1)
	assert.Equal(t, sub.OrganizationID, result.PurgeTargets[0].OrganizationID)
	assert.Len(t, h.emails.trialEnded, 2)
	assert.Len(t, h.emails.purged, 2)
}

func TestSweepCollectsExpiredOrganizationsStillAwaitingPurgeOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := time.Now().Unix()
	due := newSubscription(subscription.StatusReadOnly, now-7_200, now-60)
	waiting := newSubscription(subscription.StatusExpired, now-9_000, now-3_600)
	duplicate := *due
	duplicate.Status = subscription.StatusExpired

	h.subs.EXPECT().ListDue(mock.Anything, mock.Anything).Return([]*subscription.Subscription{due}, nil)
	h.subs.EXPECT().ListExpired(mock.Anything, mock.Anything).
		Return([]*subscription.Subscription{&duplicate, waiting}, nil)
	h.expectTransition(due, subscription.StatusExpired)

	result, err := h.svc.Sweep(t.Context(), now)

	require.NoError(t, err)
	require.Len(t, result.PurgeTargets, 2)
	assert.Equal(t, due.OrganizationID, result.PurgeTargets[0].OrganizationID)
	assert.Equal(t, waiting.OrganizationID, result.PurgeTargets[1].OrganizationID)
}

func TestSweepRecordsAFailedTransitionAndCarriesOn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := time.Now().Unix()
	failing := newSubscription(subscription.StatusTrialing, now-60, now+3_600)
	healthy := newSubscription(subscription.StatusTrialing, now-60, now+3_600)

	h.subs.EXPECT().ListDue(mock.Anything, mock.Anything).
		Return([]*subscription.Subscription{failing, healthy}, nil)
	h.subs.EXPECT().ListExpired(mock.Anything, mock.Anything).Return(nil, nil)
	h.subs.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *repositories.UpdateSubscriptionStatusRequest) bool {
			return req.ID == failing.ID
		})).
		Return(nil, errortypes.NewNotFoundError("Subscription")).
		Once()
	h.expectTransition(healthy, subscription.StatusReadOnly)

	result, err := h.svc.Sweep(t.Context(), now)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, []string{failing.OrganizationID.String()}, result.Failures)
	assert.Equal(t, 1, result.ReadOnly)
}

func TestSweepReturnsListingErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	boom := errors.New("database unavailable")
	h.subs.EXPECT().ListDue(mock.Anything, mock.Anything).Return(nil, boom)

	_, err := h.svc.Sweep(t.Context(), 0)
	require.ErrorIs(t, err, boom)
}

func TestRequireExpiredOnlyAdmitsExpiredSubscriptions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	expired := newSubscription(subscription.StatusExpired, 1, 2)
	readOnly := newSubscription(subscription.StatusReadOnly, 1, 2)
	missing := cloudlifecycleservice.TenantRef{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	h.subs.EXPECT().
		GetByOrganization(mock.Anything, mock.MatchedBy(func(req repositories.GetSubscriptionRequest) bool {
			return req.TenantInfo.OrgID == expired.OrganizationID
		})).
		Return(expired, nil)
	h.subs.EXPECT().
		GetByOrganization(mock.Anything, mock.MatchedBy(func(req repositories.GetSubscriptionRequest) bool {
			return req.TenantInfo.OrgID == readOnly.OrganizationID
		})).
		Return(readOnly, nil)
	h.subs.EXPECT().
		GetByOrganization(mock.Anything, mock.MatchedBy(func(req repositories.GetSubscriptionRequest) bool {
			return req.TenantInfo.OrgID == missing.OrganizationID
		})).
		Return(nil, errortypes.NewNotFoundError("Subscription"))

	require.NoError(t, h.svc.RequireExpired(t.Context(), cloudlifecycleservice.TenantRef{
		OrganizationID: expired.OrganizationID,
		BusinessUnitID: expired.BusinessUnitID,
	}))
	require.ErrorIs(t, h.svc.RequireExpired(t.Context(), cloudlifecycleservice.TenantRef{
		OrganizationID: readOnly.OrganizationID,
		BusinessUnitID: readOnly.BusinessUnitID,
	}), cloudlifecycleservice.ErrNotExpired)
	require.ErrorIs(t, h.svc.RequireExpired(t.Context(), missing), cloudlifecycleservice.ErrNotExpired)
}

func TestPurgeStorageDeletesTheOrganizationPrefix(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	ref := cloudlifecycleservice.TenantRef{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	sub := newSubscription(subscription.StatusExpired, 1, 2)
	sub.OrganizationID = ref.OrganizationID
	sub.BusinessUnitID = ref.BusinessUnitID
	h.subs.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(sub, nil)

	deleted, err := h.svc.PurgeStorage(t.Context(), ref)

	require.NoError(t, err)
	assert.Equal(t, int64(7), deleted)
	assert.Equal(t, []string{ref.OrganizationID.String() + "/"}, h.storage.prefixes)
}

func TestPurgeStorageRefusesAnOrganizationThatHasNotExpired(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sub := newSubscription(subscription.StatusReadOnly, 1, 2)
	h.subs.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(sub, nil)

	_, err := h.svc.PurgeStorage(t.Context(), cloudlifecycleservice.TenantRef{
		OrganizationID: sub.OrganizationID,
		BusinessUnitID: sub.BusinessUnitID,
	})

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	assert.Empty(t, h.storage.prefixes)
}

func TestPurgeStorageRefusesAnEmptyTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.PurgeStorage(t.Context(), cloudlifecycleservice.TenantRef{})

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	assert.Empty(t, h.storage.prefixes)
}

func TestPurgeStorageReportsAnUnsupportedBackend(t *testing.T) {
	t.Parallel()

	subs := mocks.NewMockSubscriptionRepository(t)
	sub := newSubscription(subscription.StatusExpired, 1, 2)
	subs.EXPECT().GetByOrganization(mock.Anything, mock.Anything).Return(sub, nil)
	svc := cloudlifecycleservice.New(cloudlifecycleservice.Params{
		Subscriptions: subs,
		Purge:         &fakePurge{},
		Plans:         mocks.NewMockPlanService(t),
		Storage:       mocks.NewMockClient(t),
		Logger:        zap.NewNop(),
	})

	_, err := svc.PurgeStorage(t.Context(), cloudlifecycleservice.TenantRef{
		OrganizationID: sub.OrganizationID,
		BusinessUnitID: sub.BusinessUnitID,
	})
	require.ErrorIs(t, err, cloudlifecycleservice.ErrStorageUnsupported)
}

func TestPurgeUsersEndsTheSessionsOfRemovedUsers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	deleted := pulid.MustNew("usr_")
	deactivated := pulid.MustNew("usr_")
	moved := pulid.MustNew("usr_")
	h.purge.userResults[deleted] = &repositories.PurgeTenantUserResult{Deleted: true}
	h.purge.userResults[deactivated] = &repositories.PurgeTenantUserResult{Deactivated: true}
	h.purge.userResults[moved] = &repositories.PurgeTenantUserResult{Reassigned: true}
	h.sessions.EXPECT().DeleteAllForUser(mock.Anything, deleted).Return(nil).Once()
	h.sessions.EXPECT().DeleteAllForUser(mock.Anything, deactivated).Return(nil).Once()

	result, err := h.svc.PurgeUsers(t.Context(), cloudlifecycleservice.TenantRef{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}, []pulid.ID{deleted, deactivated, moved})

	require.NoError(t, err)
	assert.Equal(t, &cloudlifecycleservice.PurgeUsersResult{Deleted: 1, Deactivated: 1, Reassigned: 1}, result)
}

func TestFinalizeDeletesTheTenantAndForgetsItsPlan(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	ref := cloudlifecycleservice.TenantRef{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	h.plans.EXPECT().Invalidate(ref.OrganizationID).Once()

	result, err := h.svc.Finalize(t.Context(), ref)

	require.NoError(t, err)
	assert.True(t, result.OrganizationDeleted)
	assert.Equal(t, []pulid.ID{ref.OrganizationID}, h.purge.deleted)
}

func TestSweepSkipsEmailWithoutAnEmailService(t *testing.T) {
	t.Parallel()

	subs := mocks.NewMockSubscriptionRepository(t)
	plans := mocks.NewMockPlanService(t)
	svc := cloudlifecycleservice.New(cloudlifecycleservice.Params{
		Subscriptions: subs,
		Purge:         &fakePurge{members: map[pulid.ID][]*repositories.TenantMember{}},
		Plans:         plans,
		Logger:        zap.NewNop(),
	})
	now := time.Now().Unix()
	sub := newSubscription(subscription.StatusTrialing, now-60, now+3_600)
	subs.EXPECT().ListDue(mock.Anything, mock.Anything).Return([]*subscription.Subscription{sub}, nil)
	subs.EXPECT().ListExpired(mock.Anything, mock.Anything).Return(nil, nil)
	subs.EXPECT().UpdateStatus(mock.Anything, mock.Anything).Return(sub, nil)
	plans.EXPECT().Invalidate(sub.OrganizationID)

	result, err := svc.Sweep(t.Context(), now)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ReadOnly)
}
