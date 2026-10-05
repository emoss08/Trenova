package supportaccessservice

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const validReason = "Customer reported a missing invoice on load 4411"

type harness struct {
	svc      *Service
	store    *memoryStore
	mfa      *fakeMFA
	auditor  *recordingAuditor
	limiter  *fakeLimiter
	users    *mocks.MockUserRepository
	clock    time.Time
	staff    StaffContext
	staffUsr *tenant.User
	customer pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	staffUser := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "Jordan Lee",
		EmailAddress:          "jordan@trenova.app",
		Timezone:              "America/Chicago",
		Locale:                "en",
	}

	store := newMemoryStore()
	store.staff[staffUser.ID] = &supportaccess.StaffMember{
		UserID: staffUser.ID,
		Role:   supportaccess.StaffRoleSupport,
		Active: true,
	}

	users := mocks.NewMockUserRepository(t)
	users.On("FindByIDForLogin", mock.Anything, staffUser.ID).Return(staffUser, nil).Maybe()
	users.On("FindByEmail", mock.Anything, staffUser.EmailAddress).Return(staffUser, nil).Maybe()

	cfg := &config.Config{Platform: config.PlatformConfig{Mode: config.PlatformModeCloud}}
	cloudconfig.Attach(cfg, &cloudconfig.Settings{})

	h := &harness{
		store:    store,
		mfa:      &fakeMFA{enrolled: true, password: "correct-horse", validCode: "123456"},
		auditor:  &recordingAuditor{},
		limiter:  &fakeLimiter{remaining: 100},
		users:    users,
		clock:    time.Unix(1_800_000_000, 0),
		staffUsr: staffUser,
		staff: StaffContext{
			UserID:             staffUser.ID,
			OrganizationID:     staffUser.CurrentOrganizationID,
			BusinessUnitID:     staffUser.BusinessUnitID,
			SessionID:          pulid.MustNew("ses_"),
			AuthenticatorAAL:   2,
			MFAAuthenticatedAt: 1_799_999_000,
		},
		customer: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	store.orgNames[h.customer.OrgID] = "Acme Freight"

	h.svc = &Service{
		repo:     store,
		users:    users,
		mfa:      h.mfa,
		auditor:  h.auditor,
		limiter:  h.limiter,
		platform: &cfg.Platform,
		settings: &cloudconfig.From(cfg).Cloud.SupportAccess,
		l:        zap.NewNop(),
		now:      func() time.Time { return h.clock },
	}

	return h
}

func (h *harness) grant(t *testing.T, mode supportaccess.AccessMode, hours int) *supportaccess.Grant {
	t.Helper()
	grant, err := h.svc.CreateGrant(t.Context(), &CreateGrantRequest{
		TenantInfo:    h.customer,
		DurationHours: hours,
		AccessMode:    mode,
	})
	require.NoError(t, err)
	return grant
}

func (h *harness) start(t *testing.T) *StartedSession {
	t.Helper()
	started, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
		Staff:          h.staff,
		OrganizationID: h.customer.OrgID,
		BusinessUnitID: h.customer.BuID,
		Reason:         validReason,
	})
	require.NoError(t, err)
	return started
}

func requireEnded(t *testing.T, err error, reason supportaccess.EndReason) {
	t.Helper()
	ended, ok := AsSessionEnded(err)
	require.True(t, ok, "expected the session to have ended, got %v", err)
	assert.Equal(t, reason, ended.Reason)
}

func TestCreateGrantRejectsDurationsBeyondTheMaximum(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	for _, hours := range []int{0, -1, 337, 1000} {
		_, err := h.svc.CreateGrant(t.Context(), &CreateGrantRequest{
			TenantInfo:    h.customer,
			DurationHours: hours,
			AccessMode:    supportaccess.AccessModeReadOnly,
		})
		require.Error(t, err, "%d hours", hours)
	}

	_, err := h.svc.CreateGrant(t.Context(), &CreateGrantRequest{
		TenantInfo:    h.customer,
		DurationHours: 24,
		AccessMode:    supportaccess.AccessMode("everything"),
	})
	require.Error(t, err)
}

func TestCreateGrantReplacesTheOpenGrantAndIsAudited(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	first := h.grant(t, supportaccess.AccessModeReadOnly, 24)
	second := h.grant(t, supportaccess.AccessModeReadWrite, 72)

	state, err := h.svc.GrantState(t.Context(), h.customer)
	require.NoError(t, err)
	require.NotNil(t, state.Grant)
	assert.Equal(t, second.ID, state.Grant.ID)
	assert.Equal(t, h.clock.Unix()+72*3600, state.Grant.ExpiresAt)
	assert.NotNil(t, h.store.grants[0].RevokedAt, "the first grant %s is closed", first.ID)
	assert.Equal(t, []int{24, 72, 168, 336}, state.DurationHours)
	require.Len(t, h.auditor.changes, 2)
}

func TestGrantExpiresOnItsOwn(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)

	h.clock = h.clock.Add(25 * time.Hour)
	state, err := h.svc.GrantState(t.Context(), h.customer)
	require.NoError(t, err)
	assert.Nil(t, state.Grant)

	_, err = h.svc.StartSession(t.Context(), &StartSessionRequest{
		Staff:          h.staff,
		OrganizationID: h.customer.OrgID,
		BusinessUnitID: h.customer.BuID,
		Reason:         validReason,
	})
	require.ErrorIs(t, err, errNoGrant)
}

func TestStartSessionNeedsAGrantStaffAndTwoFactor(t *testing.T) {
	t.Parallel()

	t.Run("no grant", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          h.staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         validReason,
		})
		require.ErrorIs(t, err, errNoGrant)
	})

	t.Run("not staff", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		delete(h.store.staff, h.staff.UserID)
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          h.staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         validReason,
		})
		require.ErrorIs(t, err, errNotStaff)
	})

	t.Run("session not verified with a second factor", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		staff := h.staff
		staff.AuthenticatorAAL = 1
		staff.MFAAuthenticatedAt = 0
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         validReason,
		})
		require.ErrorIs(t, err, errMFARequired)
	})

	t.Run("no second factor enrolled", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		h.mfa.enrolled = false
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          h.staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         validReason,
		})
		require.ErrorIs(t, err, errMFARequired)
	})

	t.Run("reason too short", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          h.staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         "help",
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsMultiError(err))
	})

	t.Run("rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		h.limiter.remaining = 0
		_, err := h.svc.StartSession(t.Context(), &StartSessionRequest{
			Staff:          h.staff,
			OrganizationID: h.customer.OrgID,
			BusinessUnitID: h.customer.BuID,
			Reason:         validReason,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsRateLimitError(err))
	})
}

func TestStartedSessionIsReadOnlyAndAttributedToASupportPrincipal(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	grant := h.grant(t, supportaccess.AccessModeReadWrite, 72)

	started := h.start(t)
	assert.Equal(t, supportaccess.AccessModeReadOnly, started.Session.Mode)
	assert.True(t, started.Session.CanElevate)
	assert.Equal(t, h.clock.Unix()+int64((4*time.Hour).Seconds()), started.Session.ExpiresAt)

	active, session, err := h.svc.Resolve(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.False(t, active.WriteActive())
	assert.Equal(t, grant.ID, active.GrantID)
	assert.Equal(t, h.customer.OrgID, active.OrganizationID)
	assert.NotEqual(t, h.staff.UserID, active.PrincipalUserID)

	principal := h.store.principals[h.customer.OrgID.String()+"/"+h.staff.UserID.String()]
	require.NotNil(t, principal)
	assert.Equal(t, "Trenova Support (Jordan Lee)", principal.Name)
	assert.Equal(t, domaintypes.StatusInactive, principal.Status)
	assert.Equal(t, h.customer.OrgID, principal.CurrentOrganizationID)
	assert.LessOrEqual(t, len(principal.Username), 20)
	assert.True(t, strings.HasSuffix(principal.EmailAddress, "@support.trenova.invalid"))
	assert.Equal(t, principal.ID, session.PrincipalUserID)

	comments := h.auditor.comments()
	assert.Contains(t, comments[len(comments)-1], "Trenova support session started by Jordan Lee")
}

func TestSessionEndsWithItsGrant(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 1)

	started := h.start(t)
	assert.Equal(t, h.clock.Unix()+3600, started.Session.ExpiresAt)

	h.clock = h.clock.Add(61 * time.Minute)
	_, _, err := h.svc.Resolve(t.Context(), h.staff, started.Token)
	requireEnded(t, err, supportaccess.EndReasonGrantExpired)
}

func TestRevokingTheGrantEndsTheSessionImmediately(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadWrite, 24)
	started := h.start(t)

	require.NoError(t, h.svc.RevokeGrant(t.Context(), h.customer))

	_, _, err := h.svc.Resolve(t.Context(), h.staff, started.Token)
	requireEnded(t, err, supportaccess.EndReasonGrantRevoked)

	require.ErrorIs(t, h.svc.RevokeGrant(t.Context(), h.customer), errNoOpenGrant)
}

func TestRemovingStaffEndsTheirSessionsOnTheNextRequest(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	started := h.start(t)

	h.store.staff[h.staff.UserID].Active = false

	_, _, err := h.svc.Resolve(t.Context(), h.staff, started.Token)
	requireEnded(t, err, supportaccess.EndReasonStaffRemoved)
}

func TestStaffManagerRemovalEndsOpenSessions(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	started := h.start(t)

	manager := &StaffManager{repo: h.store, users: h.users, auditor: h.auditor, now: h.svc.now}
	result, err := manager.RemoveStaff(t.Context(), &RemoveStaffRequest{
		EmailAddress: h.staffUsr.EmailAddress,
		RemovedBy:    "ops",
	})
	require.NoError(t, err)
	assert.True(t, result.Removed)
	assert.Equal(t, 1, result.EndedSessions)

	_, _, err = h.svc.Resolve(t.Context(), h.staff, started.Token)
	requireEnded(t, err, supportaccess.EndReasonStaffRemoved)
}

func TestSessionIsBoundToTheStaffLoginItWasOpenedFrom(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	started := h.start(t)

	other := h.staff
	other.SessionID = pulid.MustNew("ses_")
	_, _, err := h.svc.Resolve(t.Context(), other, started.Token)
	requireEnded(t, err, supportaccess.EndReasonSignedOut)
}

func TestTamperedTokenIsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	started := h.start(t)

	tampered := started.Token[:len(started.Token)-4] + "AAAA"
	_, _, err := h.svc.Resolve(t.Context(), h.staff, tampered)
	_, ended := AsSessionEnded(err)
	assert.True(t, ended)

	otherStaff := h.staff
	otherStaff.UserID = pulid.MustNew("usr_")
	_, _, err = h.svc.Resolve(t.Context(), otherStaff, started.Token)
	_, ended = AsSessionEnded(err)
	assert.True(t, ended)
}

func TestElevationNeedsAWriteGrantReauthenticationAndATicket(t *testing.T) {
	t.Parallel()

	t.Run("read-only grant", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadOnly, 24)
		started := h.start(t)
		_, err := h.svc.Elevate(t.Context(), h.staff, started.Token, &ElevateRequest{
			Password:        "correct-horse",
			Code:            "123456",
			Reason:          validReason,
			TicketReference: "SUP-1",
		})
		require.ErrorIs(t, err, errReadOnlyGrant)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadWrite, 24)
		started := h.start(t)
		_, err := h.svc.Elevate(t.Context(), h.staff, started.Token, &ElevateRequest{
			Password:        "wrong",
			Code:            "123456",
			Reason:          validReason,
			TicketReference: "SUP-1",
		})
		require.Error(t, err)
		active, _, resolveErr := h.svc.Resolve(t.Context(), h.staff, started.Token)
		require.NoError(t, resolveErr)
		assert.False(t, active.WriteActive())
	})

	t.Run("missing ticket", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.grant(t, supportaccess.AccessModeReadWrite, 24)
		started := h.start(t)
		_, err := h.svc.Elevate(t.Context(), h.staff, started.Token, &ElevateRequest{
			Password: "correct-horse",
			Code:     "123456",
			Reason:   validReason,
		})
		require.Error(t, err)
		assert.Zero(t, h.mfa.calls)
	})
}

func TestElevationExpiresAndCanBeDroppedEarly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadWrite, 24)
	started := h.start(t)

	view, err := h.svc.Elevate(t.Context(), h.staff, started.Token, &ElevateRequest{
		Password:        "correct-horse",
		Code:            "123456",
		Reason:          validReason,
		TicketReference: "SUP-42",
	})
	require.NoError(t, err)
	assert.Equal(t, supportaccess.AccessModeReadWrite, view.Mode)
	assert.Equal(t, h.clock.Unix()+1800, view.ElevatedUntil)

	active, _, err := h.svc.Resolve(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.True(t, active.WriteActive())

	h.clock = h.clock.Add(31 * time.Minute)
	active, _, err = h.svc.Resolve(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.False(t, active.WriteActive())

	_, err = h.svc.Elevate(t.Context(), h.staff, started.Token, &ElevateRequest{
		Password:        "correct-horse",
		Code:            "123456",
		Reason:          validReason,
		TicketReference: "SUP-42",
	})
	require.NoError(t, err)

	view, err = h.svc.DropElevation(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.Equal(t, supportaccess.AccessModeReadOnly, view.Mode)

	active, _, err = h.svc.Resolve(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.False(t, active.WriteActive())
}

func TestEndingASessionIsFinal(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	started := h.start(t)

	require.NoError(t, h.svc.EndSession(t.Context(), h.staff, started.Token))

	current, err := h.svc.CurrentSession(t.Context(), h.staff, started.Token)
	require.NoError(t, err)
	assert.False(t, current.Active)
	assert.Equal(t, supportaccess.EndReasonExited.String(), current.Ended)

	state, err := h.svc.GrantState(t.Context(), h.customer)
	require.NoError(t, err)
	require.Len(t, state.Sessions, 1)
	assert.Equal(t, "ended", state.Sessions[0].Status)
	assert.Equal(t, "Jordan Lee", state.Sessions[0].StaffName)
}

func TestOpeningASecondSessionEndsTheFirst(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.grant(t, supportaccess.AccessModeReadOnly, 24)
	first := h.start(t)
	second := h.start(t)

	_, _, err := h.svc.Resolve(t.Context(), h.staff, first.Token)
	requireEnded(t, err, supportaccess.EndReasonReplaced)

	_, _, err = h.svc.Resolve(t.Context(), h.staff, second.Token)
	require.NoError(t, err)
}

func TestSupportAccessIsUnavailableOutsideCloud(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.svc.platform = &config.PlatformConfig{Mode: config.PlatformModeSelfHosted}

	_, err := h.svc.GrantState(t.Context(), h.customer)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.False(t, h.svc.Enabled())
}
