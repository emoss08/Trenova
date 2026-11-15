package systememailservice_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/systememailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	apiKey        = "pm-secret-0123456789abcdef0123"
	webhookSecret = "whsec_secret_abcdef"
	receivedAt    = int64(1_794_650_400)
)

type fakeRepo struct {
	mu     sync.Mutex
	sender *systememail.Sender
	saves  int
	gets   int
	err    error
}

func (f *fakeRepo) Get(context.Context) (*systememail.Sender, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.err != nil {
		return nil, f.err
	}
	if f.sender == nil {
		return nil, systememail.ErrNoSender
	}
	c := *f.sender
	return &c, nil
}

func (f *fakeRepo) Save(_ context.Context, s *systememail.Sender) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	c := *s
	f.sender = &c
	f.saves++
	return nil
}

func (f *fakeRepo) Delete(context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	had := f.sender != nil
	f.sender = nil
	return had, f.err
}

func cipher(t *testing.T, key string) kernelports.SecretCipher {
	t.Helper()
	keyring, err := encryptionservice.NewLocalKeyring(key, nil)
	require.NoError(t, err)
	return encryptionservice.NewSecretCipher(encryptionservice.NewWithKeyManager(keyring))
}

type rig struct {
	svc  *systememailservice.Service
	repo *fakeRepo
	logs *observer.ObservedLogs
}

func newRig(t *testing.T, managed bool, repo *fakeRepo, key string) *rig {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	return &rig{
		svc: systememailservice.New(systememailservice.Params{
			Config: &managedconfig.Config{Enabled: managed},
			Repo:   repo,
			Cipher: cipher(t, key),
			Clock:  func() int64 { return receivedAt },
			Logger: zap.New(core),
		}),
		repo: repo,
		logs: logs,
	}
}

func credential() *systememail.Credential {
	return &systememail.Credential{
		Provider:      email.ProviderPostmark,
		APIKey:        apiKey,
		FromAddress:   "no-reply@mail.trenova.app",
		FromName:      "Trenova",
		WebhookSecret: webhookSecret,
	}
}

func assertNoSecretLogged(t *testing.T, logs *observer.ObservedLogs) {
	t.Helper()
	for _, entry := range logs.All() {
		line := entry.Message + " " + fmt.Sprint(entry.ContextMap())
		assert.NotContains(t, line, apiKey)
		assert.NotContains(t, line, webhookSecret)
	}
}

const testKey = "0123456789abcdef0123456789abcdef-active"

func TestTheSystemSenderIsStoredEncrypted(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	require.NoError(t, r.svc.Set(t.Context(), credential()))

	stored := r.repo.sender
	require.NotNil(t, stored)
	assert.NotContains(t, stored.APIKeySealed, apiKey)
	assert.NotContains(t, stored.WebhookSecretSealed, webhookSecret)
	assert.True(t, encryptionservice.IsEnvelope(stored.APIKeySealed))
	assert.True(t, encryptionservice.IsEnvelope(stored.WebhookSecretSealed))
	assert.Equal(t, systememail.Fingerprint(credential()), stored.Fingerprint)
	assert.Equal(t, receivedAt, stored.ReceivedAt)

	current, err := r.svc.Current(t.Context())
	require.NoError(t, err)
	require.NotNil(t, current)
	assert.Equal(t, apiKey, current.APIKey)
	assert.Equal(t, webhookSecret, current.WebhookSecret)
	assertNoSecretLogged(t, r.logs)
}

func TestTheStatusNamesTheSenderButNotItsSecret(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	status, err := r.svc.Status(t.Context())
	require.NoError(t, err)
	assert.False(t, status.Configured)

	require.NoError(t, r.svc.Set(t.Context(), credential()))
	status, err = r.svc.Status(t.Context())
	require.NoError(t, err)
	assert.Equal(t, systememail.Status{
		Configured:  true,
		Provider:    email.ProviderPostmark,
		FromAddress: "no-reply@mail.trenova.app",
		Fingerprint: systememail.Fingerprint(credential()),
	}, status)
}

func TestTheSameCredentialIsStoredOnce(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	require.NoError(t, r.svc.Set(t.Context(), credential()))
	require.NoError(t, r.svc.Set(t.Context(), credential()))
	assert.Equal(t, 1, r.repo.saves)
}

func TestANewCredentialReplacesTheOld(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	require.NoError(t, r.svc.Set(t.Context(), credential()))
	_, err := r.svc.Current(t.Context())
	require.NoError(t, err)

	rotated := credential()
	rotated.APIKey = "pm-rotated-0123456789abcdef01"
	require.NoError(t, r.svc.Set(t.Context(), rotated))
	assert.Equal(t, 2, r.repo.saves)

	current, err := r.svc.Current(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rotated.APIKey, current.APIKey)
}

func TestRemovingTheSenderLeavesNone(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	require.NoError(t, r.svc.Set(t.Context(), credential()))
	require.NoError(t, r.svc.Remove(t.Context()))
	require.NoError(t, r.svc.Remove(t.Context()))

	current, err := r.svc.Current(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current)
}

func TestASelfHostedInstanceHasNoSystemSenderAndNeverReadsOne(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	r := newRig(t, false, repo, testKey)
	current, err := r.svc.Current(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current)
	status, err := r.svc.Status(t.Context())
	require.NoError(t, err)
	assert.False(t, status.Configured)
	assert.Zero(t, repo.gets)
}

func TestASenderSealedUnderALostKeyCountsAsUnsetAndIsStoredAgain(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	before := newRig(t, true, repo, "an-old-key-that-was-lost-0123456789abcdef")
	require.NoError(t, before.svc.Set(t.Context(), credential()))

	after := newRig(t, true, repo, testKey)
	status, err := after.svc.Status(t.Context())
	require.NoError(t, err)
	assert.False(t, status.Configured)
	require.NotEmpty(t, after.logs.All())
	assertNoSecretLogged(t, after.logs)

	require.NoError(t, after.svc.Set(t.Context(), credential()))
	assert.Equal(t, 2, repo.saves)
	status, err = after.svc.Status(t.Context())
	require.NoError(t, err)
	assert.True(t, status.Configured)
}

func TestAMalformedCredentialIsRefusedAndNothingIsStored(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	bad := credential()
	bad.FromAddress = "not an address"
	err := r.svc.Set(t.Context(), bad)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey)
	assert.Zero(t, r.repo.saves)
}

func TestAStoreFailureNamesNoSecret(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{err: errors.New("connection refused")}
	r := newRig(t, true, repo, testKey)
	err := r.svc.Set(t.Context(), credential())
	require.Error(t, err)
	assert.False(t, strings.Contains(err.Error(), apiKey))
	assert.False(t, strings.Contains(err.Error(), webhookSecret))
	assertNoSecretLogged(t, r.logs)
}

func TestTheWebhookSecretIsOnlyForTheSendersProvider(t *testing.T) {
	t.Parallel()

	r := newRig(t, true, &fakeRepo{}, testKey)
	require.NoError(t, r.svc.Set(t.Context(), credential()))

	secret, ok, err := r.svc.WebhookSecret(t.Context(), email.ProviderPostmark)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, webhookSecret, secret)

	_, ok, err = r.svc.WebhookSecret(t.Context(), email.ProviderResend)
	require.NoError(t, err)
	assert.False(t, ok)
}
