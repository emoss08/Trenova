package mfaservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

func (r *memoryRepo) ActivateTOTP(_ context.Context, req *repositories.ActivateTOTPRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authenticator.Enabled = true
	r.authenticator.VerifiedAt = req.VerifiedAt
	r.authenticator.LastUsedStep = req.Step
	r.codes = req.RecoveryCodes
	return nil
}

func (r *memoryRepo) RecordTOTPUse(_ context.Context, req repositories.RecordTOTPUseRequest) (bool, error) {
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
	req *repositories.ReplaceRecoveryCodesRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codes = req.Codes
	return nil
}

func (r *memoryRepo) ConsumeRecoveryCode(
	_ context.Context,
	req repositories.ConsumeRecoveryCodeRequest,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, code := range r.codes {
		if code.CodeHash == req.CodeHash && code.UsedAt == 0 {
			code.UsedAt = req.UsedAt
			return true, nil
		}
	}
	return false, nil
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
	changes []*services.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *services.SecurityChange) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.changes = append(a.changes, change)
}

type harness struct {
	svc      *Service
	repo     *memoryRepo
	users    *mocks.MockUserRepository
	sessions *mocks.MockSessionRepository
	auditor  *recordingAuditor
	user     *tenant.User
	tenant   pagination.TenantInfo
	clock    time.Time
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

	users := mocks.NewMockUserRepository(t)
	users.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil).Maybe()
	sessions := mocks.NewMockSessionRepository(t)
	repo := &memoryRepo{}
	auditor := &recordingAuditor{}

	h := &harness{
		repo:     repo,
		users:    users,
		sessions: sessions,
		auditor:  auditor,
		user:     usr,
		tenant: pagination.TenantInfo{
			OrgID:  usr.CurrentOrganizationID,
			BuID:   usr.BusinessUnitID,
			UserID: usr.ID,
		},
		clock: time.Unix(1_800_000_000, 0),
	}

	h.svc = newService(Params{
		Repository: repo,
		Users:      users,
		Sessions:   sessions,
		Encryption: encryptionservice.New(encryptionservice.Params{
			Config: &config.Config{
				Security: config.SecurityConfig{
					Encryption: config.EncryptionConfig{
						Key: "unit-test-encryption-key-with-at-least-32-bytes",
					},
				},
			},
		}),
		Auditor: auditor,
		Config:  &config.Config{App: config.AppConfig{Name: "Trenova"}},
		Logger:  zap.NewNop(),
	})
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

	enrollment, err := h.svc.BeginTOTPEnrollment(t.Context(), &services.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
	})
	require.NoError(t, err)

	resp, err := h.svc.ConfirmTOTPEnrollment(t.Context(), &services.ConfirmTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Code:       h.codeAt(t, enrollment.Secret, 0),
	})
	require.NoError(t, err)

	return enrollment.Secret, resp.RecoveryCodes
}

func TestEnrollmentNeedsThePassword(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.BeginTOTPEnrollment(t.Context(), &services.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "wrong",
	})
	require.ErrorIs(t, err, errInvalidPassword)
	assert.Nil(t, h.repo.authenticator)
}

func TestEnrollmentStoresTheSecretEncryptedAndReturnsAQRCode(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	enrollment, err := h.svc.BeginTOTPEnrollment(t.Context(), &services.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
	})
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

	enrollment, err := h.svc.BeginTOTPEnrollment(t.Context(), &services.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
	})
	require.NoError(t, err)

	_, err = h.svc.ConfirmTOTPEnrollment(t.Context(), &services.ConfirmTOTPEnrollmentRequest{
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
	sess := &session.Session{ID: sessionID, UserID: h.user.ID, AuthenticatorAAL: 1}
	h.sessions.On("Get", mock.Anything, sessionID).Return(sess, nil)
	h.sessions.On("Update", mock.Anything, mock.MatchedBy(func(s *session.Session) bool {
		return s.AuthenticatorAAL == 2 && s.MFAAuthenticatedAt == h.clock.Unix()
	})).Return(nil)

	enrollment, err := h.svc.BeginTOTPEnrollment(t.Context(), &services.BeginTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
	})
	require.NoError(t, err)

	resp, err := h.svc.ConfirmTOTPEnrollment(t.Context(), &services.ConfirmTOTPEnrollmentRequest{
		TenantInfo: h.tenant,
		SessionID:  sessionID,
		Code:       h.codeAt(t, enrollment.Secret, 0),
	})
	require.NoError(t, err)
	assert.Len(t, resp.RecoveryCodes, totputils.RecoveryCodeCount)

	status, err := h.svc.Status(t.Context(), h.tenant, 2)
	require.NoError(t, err)
	assert.True(t, status.TOTPEnabled)
	assert.True(t, status.SessionVerified)
	assert.Equal(t, totputils.RecoveryCodeCount, status.RecoveryCodesRemaining)
	require.Len(t, h.auditor.changes, 1)
	h.sessions.AssertExpectations(t)
}

func TestVerifySecondFactorRefusesAReplayedCode(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)

	h.clock = h.clock.Add(2 * totputils.Period)
	code := h.codeAt(t, secret, 0)

	method, err := h.svc.VerifySecondFactor(t.Context(), &services.VerifySecondFactorRequest{
		UserID: h.user.ID,
		Code:   code,
	})
	require.NoError(t, err)
	assert.Equal(t, services.MFAMethodTOTP, method)

	_, err = h.svc.VerifySecondFactor(t.Context(), &services.VerifySecondFactorRequest{
		UserID: h.user.ID,
		Code:   code,
	})
	require.ErrorIs(t, err, errInvalidCode)
}

func TestRecoveryCodesWorkOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, codes := h.enroll(t)

	method, err := h.svc.VerifySecondFactor(t.Context(), &services.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: " " + codes[0] + " ",
	})
	require.NoError(t, err)
	assert.Equal(t, services.MFAMethodRecoveryCode, method)

	_, err = h.svc.VerifySecondFactor(t.Context(), &services.VerifySecondFactorRequest{
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

	_, err := h.svc.Reauthenticate(t.Context(), &services.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "correct-horse",
		Code:     "123456",
	})
	require.ErrorIs(t, err, errNoActiveFactor)

	secret, _ := h.enroll(t)
	h.clock = h.clock.Add(2 * totputils.Period)

	_, err = h.svc.Reauthenticate(t.Context(), &services.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "wrong",
		Code:     h.codeAt(t, secret, 0),
	})
	require.ErrorIs(t, err, errInvalidPassword)

	method, err := h.svc.Reauthenticate(t.Context(), &services.ReauthenticateRequest{
		UserID:   h.user.ID,
		Password: "correct-horse",
		Code:     h.codeAt(t, secret, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, services.MFAMethodTOTP, method)
}

func TestDisableNeedsPasswordAndFactor(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)
	h.clock = h.clock.Add(2 * totputils.Period)

	err := h.svc.DisableTOTP(t.Context(), &services.DisableTOTPRequest{
		TenantInfo: h.tenant,
		Password:   "correct-horse",
		Code:       "000000",
	})
	require.Error(t, err)

	err = h.svc.DisableTOTP(t.Context(), &services.DisableTOTPRequest{
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

	resp, err := h.svc.RegenerateRecoveryCodes(t.Context(), &services.RegenerateRecoveryCodesRequest{
		TenantInfo: h.tenant,
		Code:       h.codeAt(t, secret, 0),
	})
	require.NoError(t, err)
	assert.Len(t, resp.RecoveryCodes, totputils.RecoveryCodeCount)

	_, err = h.svc.VerifySecondFactor(t.Context(), &services.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: original[0],
	})
	require.ErrorIs(t, err, errInvalidRecoveryCode)
}
