package mfaservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountguard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/sessiontest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/totputils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryRepo struct {
	mu            sync.Mutex
	authenticator *iam.MFAAuthenticator
	codes         []*iam.MFARecoveryCode
}

func (r *memoryRepo) GetTOTP(context.Context, pulid.ID) (*iam.MFAAuthenticator, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.authenticator == nil {
		return nil, nil //nolint:nilnil // mirrors the repository contract
	}
	copied := *r.authenticator
	return &copied, nil
}

func (r *memoryRepo) ReplacePendingTOTP(_ context.Context, a *iam.MFAAuthenticator) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.authenticator != nil && r.authenticator.Enabled {
		return errortypes.NewBusinessError("already enabled")
	}
	a.ID = pulid.MustNew("mfa_")
	copied := *a
	r.authenticator = &copied
	return nil
}

func (r *memoryRepo) ActivateTOTP(_ context.Context, req *iamports.ActivateTOTPRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authenticator.Enabled = true
	r.authenticator.VerifiedAt = req.VerifiedAt
	r.authenticator.LastUsedStep = req.Step
	r.codes = req.RecoveryCodes
	return nil
}

func (r *memoryRepo) RecordTOTPUse(
	_ context.Context,
	req iamports.RecordTOTPUseRequest,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.authenticator.LastUsedStep >= req.Step {
		return false, nil
	}
	r.authenticator.LastUsedStep = req.Step
	r.authenticator.LastUsedAt = req.UsedAt
	return true, nil
}

func (r *memoryRepo) DeleteAll(context.Context, pulid.ID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	if r.authenticator != nil {
		removed = 1
	}
	r.authenticator = nil
	r.codes = nil
	return removed, nil
}

func (r *memoryRepo) ReplaceRecoveryCodes(
	_ context.Context,
	req *iamports.ReplaceRecoveryCodesRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codes = req.Codes
	return nil
}

func (r *memoryRepo) ConsumeRecoveryCode(
	_ context.Context,
	req iamports.ConsumeRecoveryCodeRequest,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, code := range r.codes {
		if code.UsedAt != 0 {
			continue
		}
		matches := code.HashVersion == iam.RecoveryCodeHashHMAC && code.CodeHash == req.CodeHash ||
			code.HashVersion == iam.RecoveryCodeHashSHA256 && code.CodeHash == req.LegacyCodeHash
		if matches {
			code.UsedAt = req.UsedAt
			return true, nil
		}
	}
	return false, nil
}

func (r *memoryRepo) CountLegacyRecoveryCodes(context.Context, pulid.ID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, code := range r.codes {
		if code.UsedAt == 0 && code.HashVersion == iam.RecoveryCodeHashSHA256 {
			count++
		}
	}
	return count, nil
}

func (r *memoryRepo) CountUnusedRecoveryCodes(context.Context, pulid.ID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, code := range r.codes {
		if code.UsedAt == 0 {
			count++
		}
	}
	return count, nil
}

type recordingAuditor struct {
	mu      sync.Mutex
	changes []*permissionports.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.changes = append(a.changes, change)
}

type harness struct {
	svc      *Service
	repo     *memoryRepo
	users    *tenantportsmock.MockUserRepository
	sessions *sessionportsmock.MockSessionRepository
	auditor  *recordingAuditor
	user     *tenant.User
	tenant   pagination.TenantInfo
	clock    time.Time
	held     *heldPermissions
	admin    pulid.ID
}

type heldPermissions struct {
	permissionports.PermissionEngine

	byUser map[pulid.ID]map[string]permissionports.EffectiveResourcePermission
}

func (h *heldPermissions) GetEffectivePermissions(
	_ context.Context,
	userID, _ pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	return &permissionports.EffectivePermissions{Resources: h.byUser[userID]}, nil
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	usr := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		EmailAddress:          "ada@example.com",
	}
	hashed, err := usr.GeneratePassword("correct-horse")
	require.NoError(t, err)
	usr.Password = hashed

	users := tenantportsmock.NewMockUserRepository(t)
	users.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil).Maybe()
	users.On("ListOrganizationMemberships", mock.Anything, mock.Anything, mock.Anything).
		Return([]*tenant.OrganizationMembership{}, nil).Maybe()
	held := &heldPermissions{}
	sessions := sessionportsmock.NewMockSessionRepository(t)
	repo := &memoryRepo{}
	auditor := &recordingAuditor{}

	h := &harness{
		repo:     repo,
		users:    users,
		sessions: sessions,
		auditor:  auditor,
		user:     usr,
		held:     held,
		admin:    pulid.MustNew("usr_"),
		tenant: pagination.TenantInfo{
			OrgID:  usr.CurrentOrganizationID,
			BuID:   usr.BusinessUnitID,
			UserID: usr.ID,
		},
		clock: time.Unix(1_800_000_000, 0),
	}

	svc, err := newService(&Params{
		Repository: repo,
		Users:      users,
		Sessions:   sessions,
		Rotator:    sessiontest.Rotator(sessions),
		Encryption: encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
		),
		Auditor:    auditor,
		AppConfig:  &appconfig.Config{Name: "Trenova"},
		Logger:     zap.NewNop(),
		Throttle:   sessiontest.OpenThrottle{},
		AuthEvents: &recordingAuthEvents{},
		Accounts: accountguard.New(accountguard.Params{
			Users:       users,
			Permissions: held,
			Auditor:     auditor,
		}),
	})
	require.NoError(t, err)
	h.svc = svc
	h.svc.now = func() time.Time { return h.clock }

	return h
}

func (h *harness) codeAt(t *testing.T, secret string, offsetSteps int64) string {
	t.Helper()
	code, err := totputils.CodeAt(secret, totputils.Step(h.clock)+offsetSteps)
	require.NoError(t, err)
	return code
}

func (h *harness) enroll(t *testing.T) (string, []string) {
	t.Helper()

	enrollment, err := h.svc.BeginTOTPEnrollment(
		t.Context(),
		&sessionports.BeginTOTPEnrollmentRequest{
			TenantInfo: h.tenant,
			Password:   "correct-horse",
		},
	)
	require.NoError(t, err)

	resp, err := h.svc.ConfirmTOTPEnrollment(
		t.Context(),
		&sessionports.ConfirmTOTPEnrollmentRequest{
			TenantInfo: h.tenant,
			Code:       h.codeAt(t, enrollment.Secret, 0),
		},
	)
	require.NoError(t, err)

	return enrollment.Secret, resp.RecoveryCodes
}

func TestEnrollmentNeedsThePassword(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.BeginTOTPEnrollment(t.Context(), &sessionports.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "wrong",
	})
	require.ErrorIs(t, err, errInvalidPassword)
	assert.Nil(t, h.repo.authenticator)
}

func TestEnrollmentStoresTheSecretEncryptedAndReturnsAQRCode(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	enrollment, err := h.svc.BeginTOTPEnrollment(
		t.Context(),
		&sessionports.BeginTOTPEnrollmentRequest{
			TenantInfo: h.tenant,
			Password:   "correct-horse",
		},
	)
	require.NoError(t, err)

	assert.NotEmpty(t, enrollment.Secret)
	assert.Contains(t, enrollment.URI, "otpauth://totp/Trenova:")
	assert.Contains(t, enrollment.QRCode, "data:image/png;base64,")
	require.NotNil(t, h.repo.authenticator)
	assert.False(t, h.repo.authenticator.Enabled)
	assert.NotContains(t, h.repo.authenticator.SecretCipher, enrollment.Secret)

	status, err := h.svc.Status(t.Context(), h.tenant, 1)
	require.NoError(t, err)
	assert.True(t, status.EnrollmentPending)
	assert.False(t, status.TOTPEnabled)
}

func TestConfirmRejectsAWrongCodeAndLeavesTheFactorOff(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	enrollment, err := h.svc.BeginTOTPEnrollment(
		t.Context(),
		&sessionports.BeginTOTPEnrollmentRequest{
			TenantInfo: h.tenant,
			Password:   "correct-horse",
		},
	)
	require.NoError(t, err)

	_, err = h.svc.ConfirmTOTPEnrollment(t.Context(), &sessionports.ConfirmTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Code:       h.codeAt(t, enrollment.Secret, 5),
	})
	require.ErrorIs(t, err, errInvalidCode)

	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestConfirmTurnsTheFactorOnAndRaisesTheSession(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	sessionID := pulid.MustNew("ses_")
	sess := &session.Session{
		ID:               sessionID,
		UserID:           h.user.ID,
		SecretHash:       "enrollment-secret-hash",
		AuthenticatorAAL: 1,
	}
	h.sessions.EXPECT().Update(mock.Anything, sessionID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(sess)).
		Once()

	enrollment, err := h.svc.BeginTOTPEnrollment(
		t.Context(),
		&sessionports.BeginTOTPEnrollmentRequest{
			TenantInfo: h.tenant,
			Password:   "correct-horse",
		},
	)
	require.NoError(t, err)

	resp, err := h.svc.ConfirmTOTPEnrollment(
		t.Context(),
		&sessionports.ConfirmTOTPEnrollmentRequest{
			TenantInfo:        h.tenant,
			SessionID:         sessionID,
			SessionSecretHash: "enrollment-secret-hash",
			Code:              h.codeAt(t, enrollment.Secret, 0),
		},
	)
	require.NoError(t, err)
	assert.Len(t, resp.RecoveryCodes, totputils.RecoveryCodeCount)
	rotatedID, secret, err := sessiontoken.Parse(resp.SessionToken)
	require.NoError(t, err)
	assert.Equal(t, sessionID, rotatedID)
	assert.True(t, sessiontest.Tokens().Matches(sessionID, h.user.ID, secret, sess.SecretHash))
	assert.Equal(t, "enrollment-secret-hash", sess.PreviousSecretHash)

	status, err := h.svc.Status(t.Context(), h.tenant, 2)
	require.NoError(t, err)
	assert.True(t, status.TOTPEnabled)
	assert.True(t, status.SessionVerified)
	assert.Equal(t, totputils.RecoveryCodeCount, status.RecoveryCodesRemaining)
	require.Len(t, h.auditor.changes, 1)
	assert.Equal(t, 2, sess.AuthenticatorAAL)
	assert.Equal(t, h.clock.Unix(), sess.MFAAuthenticatedAt)
}

func TestVerifySecondFactorRefusesAReplayedCode(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)

	h.clock = h.clock.Add(2 * totputils.Period)
	code := h.codeAt(t, secret, 0)

	method, err := h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID: h.user.ID,
		Code:   code,
	})
	require.NoError(t, err)
	assert.Equal(t, sessionports.MFAMethodTOTP, method)

	_, err = h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID: h.user.ID,
		Code:   code,
	})
	require.ErrorIs(t, err, errInvalidCode)
}

func TestRecoveryCodesWorkOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, codes := h.enroll(t)

	method, err := h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: " " + codes[0] + " ",
	})
	require.NoError(t, err)
	assert.Equal(t, sessionports.MFAMethodRecoveryCode, method)

	_, err = h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: codes[0],
	})
	require.ErrorIs(t, err, errInvalidRecoveryCode)

	status, err := h.svc.Status(t.Context(), h.tenant, 2)
	require.NoError(t, err)
	assert.Equal(t, totputils.RecoveryCodeCount-1, status.RecoveryCodesRemaining)
}

func TestReauthenticateNeedsBothFactors(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.Reauthenticate(t.Context(), &sessionports.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "correct-horse",
		Code:     "123456",
	})
	require.ErrorIs(t, err, errNoActiveFactor)

	secret, _ := h.enroll(t)
	h.clock = h.clock.Add(2 * totputils.Period)

	_, err = h.svc.Reauthenticate(t.Context(), &sessionports.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "wrong",
		Code:     h.codeAt(t, secret, 0),
	})
	require.ErrorIs(t, err, errInvalidPassword)

	method, err := h.svc.Reauthenticate(t.Context(), &sessionports.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "correct-horse",
		Code:     h.codeAt(t, secret, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, sessionports.MFAMethodTOTP, method)
}

func TestDisableNeedsPasswordAndFactor(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)
	h.clock = h.clock.Add(2 * totputils.Period)

	err := h.svc.DisableTOTP(t.Context(), &sessionports.DisableTOTPRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
		Code:       "000000",
	})
	require.Error(t, err)

	err = h.svc.DisableTOTP(t.Context(), &sessionports.DisableTOTPRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
		Code:       h.codeAt(t, secret, 0),
	})
	require.NoError(t, err)

	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestRegenerateRecoveryCodesReplacesTheSet(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, original := h.enroll(t)
	h.clock = h.clock.Add(2 * totputils.Period)

	resp, err := h.svc.RegenerateRecoveryCodes(
		t.Context(),
		&sessionports.RegenerateRecoveryCodesRequest{
			TenantInfo: h.tenant,
			Code:       h.codeAt(t, secret, 0),
		},
	)
	require.NoError(t, err)
	assert.Len(t, resp.RecoveryCodes, totputils.RecoveryCodeCount)

	_, err = h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: original[0],
	})
	require.ErrorIs(t, err, errInvalidRecoveryCode)
}

func (h *harness) expectResetTarget() *sessionports.ResetUserMFARequest {
	req := &sessionports.ResetUserMFARequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  h.tenant.OrgID,
			BuID:   h.tenant.BuID,
			UserID: h.admin,
		},
		TargetUserID: h.user.ID,
	}
	h.users.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: h.user.ID,
	}).Return(h.user, nil).Once()
	return req
}

func TestResetUserMFARefusesAnActorWhoDoesNotHoldWhatTheTargetHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.enroll(t)
	h.held.byUser = map[pulid.ID]map[string]permissionports.EffectiveResourcePermission{
		h.user.ID: {
			permission.ResourceRole.String(): {
				Operations: []permission.Operation{permission.OpUpdate},
				DataScope:  permission.DataScopeAll,
			},
		},
	}
	req := h.expectResetTarget()

	err := h.svc.ResetUserMFA(t.Context(), req)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), err.Error())
	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.True(t, active, "the factor stays")
}

func TestResetUserMFAEndsEverySessionOfTheUser(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.enroll(t)
	events := &recordingAuthEvents{}
	h.svc.authEvents = events
	req := h.expectResetTarget()
	h.users.EXPECT().AdvanceSessionEpoch(mock.Anything, h.user.ID).Return(nil).Twice()
	h.sessions.EXPECT().DeleteAllForUser(mock.Anything, h.user.ID).Return(nil).Once()

	require.NoError(t, h.svc.ResetUserMFA(t.Context(), req))

	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.False(t, active)
	recorded := events.snapshot()
	require.Len(t, recorded, 1)
	assert.Equal(t, iamports.AuthEventProviderSessionRevoked, recorded[0].Provider)
	assert.Equal(t, h.user.ID, recorded[0].UserID)
	assert.Equal(t, iamports.SessionRevocationAll, recorded[0].MFAState)
}

func TestResetUserMFAKeepsTheFactorWhenSessionsCannotBeEnded(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.enroll(t)
	req := h.expectResetTarget()
	h.users.EXPECT().AdvanceSessionEpoch(mock.Anything, h.user.ID).Return(nil).Once()
	h.sessions.EXPECT().DeleteAllForUser(mock.Anything, h.user.ID).
		Return(errors.New("redis unavailable")).
		Once()

	require.Error(t, h.svc.ResetUserMFA(t.Context(), req))

	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.True(t, active)
}

func TestResetUserMFALeavesSessionsAloneWhenThereIsNoFactor(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	events := &recordingAuthEvents{}
	h.svc.authEvents = events
	req := h.expectResetTarget()

	err := h.svc.ResetUserMFA(t.Context(), req)

	require.ErrorIs(t, err, errNoActiveFactor)
	h.sessions.AssertNotCalled(t, "DeleteAllForUser", mock.Anything, mock.Anything)
	assert.Empty(t, events.snapshot())
}

func TestResetUserMFAKeepsEverythingWhenTheEpochCannotAdvance(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.enroll(t)
	req := h.expectResetTarget()
	h.users.EXPECT().AdvanceSessionEpoch(mock.Anything, h.user.ID).
		Return(errors.New("database unavailable")).
		Once()

	require.Error(t, h.svc.ResetUserMFA(t.Context(), req))

	h.sessions.AssertNotCalled(t, "DeleteAllForUser", mock.Anything, mock.Anything)
	active, err := h.svc.HasActiveFactor(t.Context(), h.user.ID)
	require.NoError(t, err)
	assert.True(t, active)
}
