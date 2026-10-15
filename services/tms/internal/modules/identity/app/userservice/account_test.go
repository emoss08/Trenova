package userservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordedEvents struct {
	records []*iamports.AuthEventRecord
}

func (r *recordedEvents) Record(_ context.Context, rec *iamports.AuthEventRecord) {
	r.records = append(r.records, rec)
}

func tenantOf(user *tenant.User) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}
}

func expectSelfLoad(deps *testDeps, user *tenant.User) {
	deps.repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req tenantports.GetUserByIDRequest) bool {
		return req.TenantInfo.UserID == user.ID
	})).Return(user, nil)
}

func capturedColumns(deps *testDeps) *[]string {
	columns := new([]string)
	deps.repo.On("UpdateColumns", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			*columns = args.Get(1).(tenantports.UpdateUserColumnsRequest).Columns
		}).
		Return(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) *tenant.User {
			return req.Entity
		}, nil)
	return columns
}

func TestUpdateMyProfileWritesOnlyTheFieldsSentAndCanClearThem(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	user.PhoneNumber = "312 555 0142"
	expectSelfLoad(deps, user)
	deps.expectAuditLog()
	columns := capturedColumns(deps)

	title := "  Dispatch lead  "
	empty := ""
	_, err := deps.svc.UpdateMyProfile(t.Context(), tenantOf(user), tenant.ProfileUpdate{
		JobTitle:    &title,
		PhoneNumber: &empty,
	})

	require.NoError(t, err)
	cols := tenantcol.UserColumns
	assert.ElementsMatch(t, []string{cols.JobTitle.String(), cols.PhoneNumber.String()}, *columns)
	assert.Equal(t, "Dispatch lead", user.JobTitle)
	assert.Empty(t, user.PhoneNumber, "an empty phone must be written, not skipped as a zero value")
}

func TestUpdateMyProfileRefusesAnAvatarStyleTrenovaDoesNotOffer(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	style := tenant.AvatarStyle("neon")

	_, err := deps.svc.UpdateMyProfile(t.Context(), tenantOf(user), tenant.ProfileUpdate{
		AvatarStyle: &style,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	deps.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestSetMyAssistantHiddenWritesOnlyTheAssistantColumn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	expectSelfLoad(deps, user)
	deps.expectAuditLog()
	columns := capturedColumns(deps)

	updated, err := deps.svc.SetMyAssistantHidden(t.Context(), tenantOf(user), true)

	require.NoError(t, err)
	assert.Equal(t, []string{tenantcol.UserColumns.AssistantHidden.String()}, *columns)
	assert.True(t, updated.AssistantHidden)
}

func TestSetMyAssistantHiddenBringsTheAssistantBack(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	user.AssistantHidden = true
	expectSelfLoad(deps, user)
	deps.expectAuditLog()
	columns := capturedColumns(deps)

	updated, err := deps.svc.SetMyAssistantHidden(t.Context(), tenantOf(user), false)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{tenantcol.UserColumns.AssistantHidden.String()},
		*columns,
		"showing the assistant again must write false, not skip it as a zero value",
	)
	assert.False(t, updated.AssistantHidden)
}

func TestSetMyAssistantHiddenWritesNothingWhenTheChoiceIsUnchanged(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	user.AssistantHidden = true
	expectSelfLoad(deps, user)

	updated, err := deps.svc.SetMyAssistantHidden(t.Context(), tenantOf(user), true)

	require.NoError(t, err)
	assert.True(t, updated.AssistantHidden)
	deps.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestUpdateMyAvailabilityRefusesAHandOffOutsideTheOrganization(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	outsider := pulid.MustNew("usr_")
	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*tenant.User{}, nil)

	_, err := deps.svc.UpdateMyAvailability(t.Context(), tenantOf(user), tenant.AvailabilityUpdate{
		Status:         tenant.AvailabilityAway,
		DelegateUserID: &outsider,
	})

	require.Error(t, err)
	deps.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestUpdateMyAvailabilityClearsWhatAFocusDoesNotUse(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	delegate := pulid.MustNew("usr_")
	backOn := time.Now().Add(48 * time.Hour).Unix()
	user.AvailabilityStatus = tenant.AvailabilityAway
	user.AwayDelegateUserID = &delegate
	user.AwayBackOn = &backOn
	user.AwayNote = "Out until Monday"
	expectSelfLoad(deps, user)
	deps.expectAuditLog()
	columns := capturedColumns(deps)

	until := time.Now().Add(2 * time.Hour).Unix()
	_, err := deps.svc.UpdateMyAvailability(t.Context(), tenantOf(user), tenant.AvailabilityUpdate{
		Status:         tenant.AvailabilityFocus,
		FocusUntil:     &until,
		DelegateUserID: &delegate,
		AwayNote:       "ignored",
	})

	require.NoError(t, err)
	assert.Equal(t, tenant.AvailabilityFocus, user.AvailabilityStatus)
	assert.Equal(t, &until, user.FocusUntil)
	assert.Nil(t, user.AwayDelegateUserID)
	assert.Nil(t, user.AwayBackOn)
	assert.Empty(t, user.AwayNote)
	assert.Len(t, *columns, 5)
}

func TestUpdateMyAvailabilityRequiresAFocusThatEnds(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()

	_, err := deps.svc.UpdateMyAvailability(t.Context(), tenantOf(user), tenant.AvailabilityUpdate{
		Status: tenant.AvailabilityFocus,
	})

	require.Error(t, err)
}

func TestDeleteProfilePictureWritesTheEmptiedColumns(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	user.ProfilePicURL = "user/profile-picture/a.webp"
	user.ThumbnailURL = "user/profile-picture/a.webp"
	expectSelfLoad(deps, user)
	deps.expectAuditLog()
	columns := capturedColumns(deps)
	deps.storage.On("Delete", mock.Anything, mock.Anything).Return(nil).Maybe()

	_, err := deps.svc.DeleteProfilePicture(t.Context(), tenantOf(user))

	require.NoError(t, err)
	cols := tenantcol.UserColumns
	assert.ElementsMatch(t, []string{cols.ProfilePicURL.String(), cols.ThumbnailURL.String()}, *columns)
	assert.Empty(t, user.ProfilePicURL)
}

func sessionFor(userID pulid.ID, lastSeen int64, ua string) *session.Session {
	return &session.Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         userID,
		UserAgent:      ua,
		IPAddress:      "73.12.88.4",
		LastAccessedAt: lastSeen,
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
	}
}

func TestListMySessionsPutsThisDeviceFirstThenMostRecent(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	now := time.Now().Unix()
	current := sessionFor(user.ID, now-600, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/129.0 Safari/537.36")
	recent := sessionFor(user.ID, now-60, "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile/15E148 Safari/604.1")
	stale := sessionFor(user.ID, now-86400, "")
	deps.sessionRepo.On("ListForUser", mock.Anything, user.ID).
		Return([]*session.Session{stale, current, recent}, nil)

	got, err := deps.svc.ListMySessions(t.Context(), SessionScope{
		TenantInfo:       tenantOf(user),
		CurrentSessionID: current.ID,
	})

	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, current.ID, got[0].ID)
	assert.True(t, got[0].Current)
	assert.Equal(t, "Chrome on macOS", got[0].Label)
	assert.Equal(t, recent.ID, got[1].ID)
	assert.Equal(t, "Safari on iOS", got[1].Label)
	assert.Equal(t, stale.ID, got[2].ID)
}

func TestRevokeMySessionRefusesTheSessionInUse(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	current := pulid.MustNew("ses_")

	err := deps.svc.RevokeMySession(t.Context(), SessionScope{
		TenantInfo:       tenantOf(user),
		CurrentSessionID: current,
	}, current)

	require.Error(t, err)
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestRevokeMySessionOnlyEndsTheCallersOwnSessions(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	user := newTestUser()
	mine := sessionFor(user.ID, time.Now().Unix(), "")
	someoneElses := pulid.MustNew("ses_")
	deps.sessionRepo.On("ListForUser", mock.Anything, user.ID).Return([]*session.Session{mine}, nil)

	err := deps.svc.RevokeMySession(t.Context(), SessionScope{TenantInfo: tenantOf(user)}, someoneElses)

	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestRevokeMySessionEndsItAndRecordsTheSignOut(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	events := &recordedEvents{}
	deps.svc.authEvents = events
	user := newTestUser()
	other := sessionFor(user.ID, time.Now().Unix(), "")
	deps.sessionRepo.On("ListForUser", mock.Anything, user.ID).Return([]*session.Session{other}, nil)
	deps.sessionRepo.On("Delete", mock.Anything, other.ID).Return(nil)

	err := deps.svc.RevokeMySession(t.Context(), SessionScope{TenantInfo: tenantOf(user)}, other.ID)

	require.NoError(t, err)
	require.Len(t, events.records, 1)
	assert.Equal(t, iamports.AuthEventProviderSessionRevoked, events.records[0].Provider)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, events.records[0].Outcome)
}

func TestRevokeMyOtherSessionsKeepsTheCurrentOne(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.authEvents = &recordedEvents{}
	user := newTestUser()
	current := sessionFor(user.ID, time.Now().Unix(), "")
	first := sessionFor(user.ID, time.Now().Unix(), "")
	second := sessionFor(user.ID, time.Now().Unix(), "")
	deps.sessionRepo.On("ListForUser", mock.Anything, user.ID).
		Return([]*session.Session{current, first, second}, nil)
	deps.sessionRepo.On("Delete", mock.Anything, first.ID).Return(nil)
	deps.sessionRepo.On("Delete", mock.Anything, second.ID).Return(nil)

	revoked, err := deps.svc.RevokeMyOtherSessions(t.Context(), SessionScope{
		TenantInfo:       tenantOf(user),
		CurrentSessionID: current.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, revoked)
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, current.ID)
}

func TestListAvatarsShowsAnAwayNoteOnlyWhileAway(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	caller := newTestUser()
	away := newTestUser()
	away.AvailabilityStatus = tenant.AvailabilityAway
	away.AwayNote = "Jordan has my loads"
	away.AvatarStyle = tenant.AvatarStyleDusk
	returned := newTestUser()
	past := time.Now().Add(-time.Hour).Unix()
	returned.AvailabilityStatus = tenant.AvailabilityAway
	returned.AwayBackOn = &past
	returned.AwayNote = "Back soon"
	deps.repo.On("GetByIDs", mock.Anything, mock.MatchedBy(func(req tenantports.GetUsersByIDsRequest) bool {
		return len(req.UserIDs) == 2
	})).Return([]*tenant.User{away, returned}, nil)

	got, err := deps.svc.ListAvatars(t.Context(), tenantOf(caller), []pulid.ID{away.ID, returned.ID, away.ID})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, tenant.AvailabilityAway, got[0].Availability)
	assert.Equal(t, "Jordan has my loads", got[0].AwayNote)
	assert.Equal(t, tenant.AvatarStyleDusk, got[0].AvatarStyle)
	assert.Equal(t, tenant.AvailabilityAvailable, got[1].Availability)
	assert.Empty(t, got[1].AwayNote)
	assert.Equal(t, tenant.AvatarStyleAuto, got[1].AvatarStyle)
}
