package systememail_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/systememail"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const apiKey = "pm-operator-0123456789abcdef01"

type fakeTMS struct {
	mu      sync.Mutex
	held    *tms.SystemEmail
	puts    []tms.SystemEmail
	deletes int
	putErr  error
	readErr error
}

func (f *fakeTMS) SystemEmail(context.Context) (tms.HeldSystemEmail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readErr != nil {
		return tms.HeldSystemEmail{}, f.readErr
	}
	if f.held == nil {
		return tms.HeldSystemEmail{}, nil
	}
	return tms.HeldSystemEmail{
		Configured:  true,
		Provider:    f.held.Provider,
		FromAddress: f.held.FromAddress,
		Fingerprint: systememail.Fingerprint(f.held),
	}, nil
}

func (f *fakeTMS) PutSystemEmail(_ context.Context, sender tms.SystemEmail) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts = append(f.puts, sender)
	if f.putErr != nil {
		return f.putErr
	}
	f.held = &sender
	return nil
}

func (f *fakeTMS) DeleteSystemEmail(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	f.held = nil
	return nil
}

func (f *fakeTMS) forget() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.held = nil
}

type fakeControlPlane struct {
	mu      sync.Mutex
	secrets instanceapi.InstanceSecrets
	err     error
	fetches int
}

func (f *fakeControlPlane) Secrets(context.Context) (instanceapi.InstanceSecrets, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fetches++
	return f.secrets, f.err
}

func credential(key string) *instanceapi.SystemEmailCredential {
	return &instanceapi.SystemEmailCredential{
		Provider:    "Postmark",
		ApiKey:      key,
		FromAddress: "no-reply@mail.trenova.app",
		FromName:    "Trenova",
	}
}

func sender(key string) tms.SystemEmail {
	return tms.SystemEmail{
		Provider:    "Postmark",
		APIKey:      key,
		FromAddress: "no-reply@mail.trenova.app",
		FromName:    "Trenova",
	}
}

func desired(key string) *instanceapi.SpecSystemEmail {
	s := sender(key)
	return &instanceapi.SpecSystemEmail{Fingerprint: systememail.Fingerprint(&s)}
}

type rig struct {
	relay *systememail.Relay
	tms   *fakeTMS
	cp    *fakeControlPlane
	logs  *bytes.Buffer
	now   time.Time
}

func newRig() *rig {
	r := &rig{
		tms:  &fakeTMS{},
		cp:   &fakeControlPlane{},
		logs: &bytes.Buffer{},
		now:  time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC),
	}
	r.cp.secrets.SystemEmail = credential(apiKey)
	r.relay = systememail.New(&systememail.Params{
		TMS:          r.tms,
		ControlPlane: r.cp,
		Logger:       slog.New(slog.NewTextHandler(r.logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Now:          func() time.Time { return r.now },
	})
	return r
}

func (r *rig) assertNoSecret(t *testing.T, errs ...error) {
	t.Helper()
	assert.NotContains(t, r.logs.String(), apiKey)
	for _, err := range errs {
		if err != nil {
			assert.NotContains(t, err.Error(), apiKey)
		}
	}
}

func contractFingerprint(fields ...string) string {
	sum := sha256.Sum256([]byte("trenova.system-email.v1\n" + strings.Join(fields, "\n")))
	return hex.EncodeToString(sum[:])
}

func TestTheFingerprintMatchesTheContractFormula(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		contractFingerprint(
			"Postmark",
			"no-reply@mail.trenova.app",
			"Trenova",
			"pm-0123456789abcdef0123456789",
		),
		systememail.Fingerprint(&tms.SystemEmail{
			Provider:    "Postmark",
			APIKey:      "pm-0123456789abcdef0123456789",
			FromAddress: "no-reply@mail.trenova.app",
			FromName:    "Trenova",
		}),
	)
	assert.Equal(
		t,
		contractFingerprint(
			"Resend",
			"no-reply@mail.trenova.app",
			"Trenova",
			"re_0123456789abcdef0123456789",
		),
		systememail.Fingerprint(&tms.SystemEmail{
			Provider:    "Resend",
			APIKey:      "re_0123456789abcdef0123456789",
			FromAddress: "no-reply@mail.trenova.app",
			FromName:    "Trenova",
		}),
	)
}

func TestSendsTheSenderWhenTheFingerprintDiffers(t *testing.T) {
	t.Parallel()
	r := newRig()

	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	assert.Equal(t, []tms.SystemEmail{sender(apiKey)}, r.tms.puts)
	assert.Equal(t, 1, r.cp.fetches)
	r.assertNoSecret(t)
}

func TestSendsNothingWhenTheFingerprintIsEqual(t *testing.T) {
	t.Parallel()
	r := newRig()
	held := sender(apiKey)
	r.tms.held = &held

	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	assert.Empty(t, r.tms.puts)
	assert.Zero(t, r.cp.fetches)
}

func TestSendsAgainFromMemoryWhenTheTMSLosesTheSender(t *testing.T) {
	t.Parallel()
	r := newRig()
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))
	r.tms.forget()

	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	assert.Len(t, r.tms.puts, 2)
	assert.Equal(t, 1, r.cp.fetches)
}

func TestRetriesAfterAFailureOnTheNextTick(t *testing.T) {
	t.Parallel()
	r := newRig()
	r.tms.putErr = errors.New("the TMS answered 503")

	first := r.relay.Sync(t.Context(), desired(apiKey))
	require.Error(t, first)

	r.tms.putErr = nil
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	assert.Len(t, r.tms.puts, 2)
	assert.Equal(t, sender(apiKey), r.tms.puts[1])
	assert.Equal(t, 1, r.cp.fetches)
	r.assertNoSecret(t, first)
}

func TestARotatedCredentialReplacesTheOld(t *testing.T) {
	t.Parallel()
	r := newRig()
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	rotated := "pm-rotated-0123456789abcdef0123"
	r.cp.secrets.SystemEmail = credential(rotated)
	require.NoError(t, r.relay.Sync(t.Context(), desired(rotated)))

	require.Len(t, r.tms.puts, 2)
	assert.Equal(t, sender(rotated), r.tms.puts[1])
	assert.Equal(t, 2, r.cp.fetches)
}

func TestRemovesTheSenderWhenTheControlPlaneHasNone(t *testing.T) {
	t.Parallel()
	r := newRig()
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))

	require.NoError(t, r.relay.Sync(t.Context(), nil))
	require.NoError(t, r.relay.Sync(t.Context(), nil))

	assert.Equal(t, 1, r.tms.deletes)
	assert.Nil(t, r.tms.held)
}

func TestWaitsBeforeFetchingAgainWhenTheSecretsDoNotMatchTheSpec(t *testing.T) {
	t.Parallel()
	r := newRig()
	r.cp.secrets.SystemEmail = credential("pm-stale-0123456789abcdef01234")

	err := r.relay.Sync(t.Context(), desired(apiKey))
	require.ErrorIs(t, err, systememail.ErrMismatch)
	err = r.relay.Sync(t.Context(), desired(apiKey))
	require.ErrorIs(t, err, systememail.ErrWaiting)
	assert.Equal(t, 1, r.cp.fetches)
	assert.Empty(t, r.tms.puts)

	r.cp.secrets.SystemEmail = credential(apiKey)
	r.now = r.now.Add(time.Minute)
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))
	assert.Equal(t, 2, r.cp.fetches)
	assert.Len(t, r.tms.puts, 1)
	r.assertNoSecret(t, err)
}

func TestARateLimitedFetchWaitsOutRetryAfter(t *testing.T) {
	t.Parallel()
	r := newRig()
	r.cp.err = &controlplane.RetryLaterError{Reason: "rate limited", RetryAfter: 5 * time.Minute}

	err := r.relay.Sync(t.Context(), desired(apiKey))
	var later *controlplane.RetryLaterError
	require.ErrorAs(t, err, &later)

	r.cp.err = nil
	r.now = r.now.Add(2 * time.Minute)
	require.ErrorIs(t, r.relay.Sync(t.Context(), desired(apiKey)), systememail.ErrWaiting)
	r.now = r.now.Add(3 * time.Minute)
	require.NoError(t, r.relay.Sync(t.Context(), desired(apiKey)))
	assert.Equal(t, 2, r.cp.fetches)
}

func TestASpecWithASenderTheSecretsLackIsAnError(t *testing.T) {
	t.Parallel()
	r := newRig()
	r.cp.secrets.SystemEmail = nil

	require.ErrorIs(t, r.relay.Sync(t.Context(), desired(apiKey)), systememail.ErrNoCredential)
	assert.Empty(t, r.tms.puts)
}

func TestAnUnauthenticatedFetchIsReturnedForCredentialRecovery(t *testing.T) {
	t.Parallel()
	r := newRig()
	r.cp.err = controlplane.ErrUnauthenticated

	require.ErrorIs(t, r.relay.Sync(t.Context(), desired(apiKey)), controlplane.ErrUnauthenticated)
}

func TestTheSenderNeverPrintsItsSecrets(t *testing.T) {
	t.Parallel()

	s := sender(apiKey)
	for _, printed := range []string{
		fmt.Sprintf("%v", s),
		fmt.Sprintf("%+v", s),
		fmt.Sprintf("%#v", s),
	} {
		assert.NotContains(t, printed, apiKey)
	}
}
