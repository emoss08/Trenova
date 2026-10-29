package mfaservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/emoss08/trenova/shared/totputils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingAuthEvents struct {
	mu      sync.Mutex
	records []*iamports.AuthEventRecord
}

func (r *recordingAuthEvents) Record(_ context.Context, rec *iamports.AuthEventRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
}

func (r *recordingAuthEvents) snapshot() []*iamports.AuthEventRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*iamports.AuthEventRecord(nil), r.records...)
}

func TestRecoveryCodesAreStoredAsKeyedDigests(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, codes := h.enroll(t)

	key, err := (&securityconfig.Config{}).RecoveryCodePepperKey()
	require.NoError(t, err)

	require.Len(t, h.repo.codes, len(codes))
	for i, code := range codes {
		normalized := totputils.NormalizeRecoveryCode(code)
		stored := h.repo.codes[i]
		assert.Equal(t, iam.RecoveryCodeHashHMAC, stored.HashVersion)
		assert.NotEqual(t, tokenutils.Hash(normalized), stored.CodeHash)

		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(normalized))
		assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), stored.CodeHash)
	}

	method, err := h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: codes[0],
	})
	require.NoError(t, err)
	assert.Equal(t, sessionports.MFAMethodRecoveryCode, method)
}

func TestRecoveryCodeDigestsDependOnThePepper(t *testing.T) {
	t.Parallel()

	first, err := newService(&Params{
		SecurityConfig: &securityconfig.Config{
			MFA: securityconfig.MFAConfig{RecoveryCodePepper: "0123456789abcdef0123456789abcdef01"},
		},
		Logger: newHarness(t).svc.l,
	})
	require.NoError(t, err)
	second, err := newService(&Params{
		SecurityConfig: &securityconfig.Config{
			MFA: securityconfig.MFAConfig{RecoveryCodePepper: "fedcba9876543210fedcba9876543210fe"},
		},
		Logger: newHarness(t).svc.l,
	})
	require.NoError(t, err)

	assert.NotEqual(
		t,
		first.hashRecoveryCode("abcde-fghjk"),
		second.hashRecoveryCode("abcde-fghjk"),
	)
	assert.Equal(t, first.hashRecoveryCode("abcde-fghjk"), first.hashRecoveryCode("abcde-fghjk"))
}

func TestLegacyRecoveryCodesKeepWorkingUntilReplaced(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)

	legacy := []string{"abcde-fghjk", "mnpqr-stuvw"}
	h.repo.mu.Lock()
	h.repo.codes = []*iam.MFARecoveryCode{
		{
			UserID:      h.user.ID,
			CodeHash:    tokenutils.Hash("abcdefghjk"),
			HashVersion: iam.RecoveryCodeHashSHA256,
		},
		{
			UserID:      h.user.ID,
			CodeHash:    tokenutils.Hash("mnpqrstuvw"),
			HashVersion: iam.RecoveryCodeHashSHA256,
		},
	}
	h.repo.mu.Unlock()

	status, err := h.svc.Status(t.Context(), h.tenant, 2)
	require.NoError(t, err)
	assert.True(t, status.RecoveryCodesNeedReplacing)

	_, err = h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: legacy[0],
	})
	require.NoError(t, err)

	h.clock = h.clock.Add(time.Minute)
	_, err = h.svc.RegenerateRecoveryCodes(
		t.Context(),
		&sessionports.RegenerateRecoveryCodesRequest{
			TenantInfo: h.tenant,
			Code:       h.codeAt(t, secret, 0),
		},
	)
	require.NoError(t, err)

	status, err = h.svc.Status(t.Context(), h.tenant, 2)
	require.NoError(t, err)
	assert.False(t, status.RecoveryCodesNeedReplacing)

	_, err = h.svc.VerifySecondFactor(t.Context(), &sessionports.VerifySecondFactorRequest{
		UserID:       h.user.ID,
		RecoveryCode: legacy[1],
	})
	require.ErrorIs(t, err, errInvalidRecoveryCode)
}

func TestRegenerateRecoveryCodesRefusesWhileThrottled(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)
	before := h.repo.codes

	throttle := sessionportsmock.NewMockLoginThrottleStore(t)
	events := &recordingAuthEvents{}
	h.svc.throttle = throttle
	h.svc.authEvents = events
	key := sessionports.LoginThrottleKey{Account: h.user.EmailAddress}
	throttle.EXPECT().Begin(mock.Anything, key).Return(&sessionports.LoginThrottleDecision{
		Blocked:    true,
		Scope:      sessionports.LoginThrottleScopeAccount,
		RetryAfter: time.Minute,
	}, nil).Once()

	h.clock = h.clock.Add(time.Minute)
	_, err := h.svc.RegenerateRecoveryCodes(
		t.Context(),
		&sessionports.RegenerateRecoveryCodesRequest{
			TenantInfo: h.tenant,
			Code:       h.codeAt(t, secret, 0),
		},
	)

	require.True(t, errortypes.IsRateLimitError(err))
	assert.Equal(t, before, h.repo.codes)
	recorded := events.snapshot()
	require.Len(t, recorded, 1)
	assert.Equal(t, iamports.AuthEventProviderMFAStepUp, recorded[0].Provider)
	assert.Equal(t, iam.AuthEventOutcomeDenied, recorded[0].Outcome)
	assert.Equal(t, stepUpErrorThrottled, recorded[0].ErrorCode)
}

func TestRegenerateRecoveryCodesCountsEveryWrongCode(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, _ = h.enroll(t)

	throttle := sessionportsmock.NewMockLoginThrottleStore(t)
	events := &recordingAuthEvents{}
	h.svc.throttle = throttle
	h.svc.authEvents = events
	key := sessionports.LoginThrottleKey{Account: h.user.EmailAddress}
	throttle.EXPECT().Begin(mock.Anything, key).Return(&sessionports.LoginThrottleDecision{
		AccountFailures: 5,
		Tripped:         true,
		TrippedScope:    sessionports.LoginThrottleScopeAccount,
	}, nil).Once()

	_, err := h.svc.RegenerateRecoveryCodes(
		t.Context(),
		&sessionports.RegenerateRecoveryCodesRequest{
			TenantInfo: h.tenant,
			Code:       "000000",
		},
	)

	require.ErrorIs(t, err, errInvalidCode)
	throttle.AssertNotCalled(t, "Succeed", mock.Anything, mock.Anything)
	recorded := events.snapshot()
	require.Len(t, recorded, 2)
	assert.Equal(t, iam.AuthEventOutcomeFailed, recorded[0].Outcome)
	assert.Equal(t, stepUpErrorRejected, recorded[0].ErrorCode)
	assert.Equal(t, h.user.ID, recorded[0].UserID)
	assert.Equal(t, iam.AuthEventOutcomeDenied, recorded[1].Outcome)
	assert.Equal(t, stepUpErrorThrottled, recorded[1].ErrorCode)
}

func TestRegenerateRecoveryCodesClearsTheCounterOnSuccess(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	secret, _ := h.enroll(t)

	throttle := sessionportsmock.NewMockLoginThrottleStore(t)
	events := &recordingAuthEvents{}
	h.svc.throttle = throttle
	h.svc.authEvents = events
	key := sessionports.LoginThrottleKey{Account: h.user.EmailAddress}
	throttle.EXPECT().Begin(mock.Anything, key).
		Return(&sessionports.LoginThrottleDecision{AccountFailures: 1}, nil).Once()
	throttle.EXPECT().Succeed(mock.Anything, key).Return(nil).Once()

	h.clock = h.clock.Add(time.Minute)
	resp, err := h.svc.RegenerateRecoveryCodes(
		t.Context(),
		&sessionports.RegenerateRecoveryCodesRequest{
			TenantInfo: h.tenant,
			Code:       h.codeAt(t, secret, 0),
		},
	)

	require.NoError(t, err)
	assert.Len(t, resp.RecoveryCodes, totputils.RecoveryCodeCount)
	recorded := events.snapshot()
	require.Len(t, recorded, 1)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, recorded[0].Outcome)
	assert.Equal(t, 2, recorded[0].AuthenticatorAAL)
}
