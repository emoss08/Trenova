package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/systememailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	systemAPIKey        = "pm-system-0123456789abcdef0123"
	systemWebhookSecret = "user:webhook-password-0123"
	systemEmailBody     = `{"provider":"Postmark","apiKey":"` + systemAPIKey +
		`","fromAddress":"no-reply@mail.trenova.app","fromName":"Trenova","webhookSecret":"` +
		systemWebhookSecret + `"}`
)

type fakeSystemSenderRepo struct {
	mu     sync.Mutex
	sender *systememail.Sender
	err    error
}

func (f *fakeSystemSenderRepo) Get(context.Context) (*systememail.Sender, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if f.sender == nil {
		return nil, systememail.ErrNoSender
	}
	c := *f.sender
	return &c, nil
}

func (f *fakeSystemSenderRepo) Save(_ context.Context, s *systememail.Sender) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	c := *s
	f.sender = &c
	return nil
}

func (f *fakeSystemSenderRepo) Delete(context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	had := f.sender != nil
	f.sender = nil
	return had, f.err
}

type systemEmailRig struct {
	handler http.Handler
	repo    *fakeSystemSenderRepo
	logs    *observer.ObservedLogs
}

func newSystemEmailRig(t *testing.T, repo *fakeSystemSenderRepo) *systemEmailRig {
	t.Helper()
	keyring, err := encryptionservice.NewLocalKeyring("0123456789abcdef0123456789abcdef-key", nil)
	require.NoError(t, err)
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	svc := systememailservice.New(systememailservice.Params{
		Config: &managedconfig.Config{Enabled: true},
		Repo:   repo,
		Cipher: encryptionservice.NewSecretCipher(encryptionservice.NewWithKeyManager(keyring)),
		Clock:  func() int64 { return featuresReceivedAt },
		Logger: logger,
	})
	return &systemEmailRig{
		handler: api.NewManagedHandler(&api.ManagedHandlerParams{
			Usage:       &fakeManagedUsage{},
			Features:    newFeatures(&fakeFeatureRepo{}),
			Invites:     &fakeInvites{},
			Token:       managedToken,
			Logger:      logger,
			SystemEmail: svc,
		}),
		repo: repo,
		logs: logs,
	}
}

func (r *systemEmailRig) serve(
	t *testing.T,
	method, token, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(
		t.Context(),
		method,
		"/managed/v1/system-email",
		strings.NewReader(body),
	)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.handler.ServeHTTP(rec, req)
	return rec
}

func (r *systemEmailRig) assertNoSecretLeaked(t *testing.T, recs ...*httptest.ResponseRecorder) {
	t.Helper()
	for _, rec := range recs {
		assert.NotContains(t, rec.Body.String(), systemAPIKey)
		assert.NotContains(t, rec.Body.String(), systemWebhookSecret)
	}
	for _, entry := range r.logs.All() {
		line := entry.Message + " " + fmt.Sprint(entry.ContextMap())
		assert.NotContains(t, line, systemAPIKey)
		assert.NotContains(t, line, systemWebhookSecret)
	}
}

func TestManagedSystemEmailIsStoredEncryptedAndNeverReturned(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
	put := r.serve(t, http.MethodPut, managedToken, systemEmailBody)
	require.Equal(t, http.StatusNoContent, put.Code, put.Body.String())

	stored := r.repo.sender
	require.NotNil(t, stored)
	assert.NotContains(t, stored.APIKeySealed, systemAPIKey)
	assert.NotContains(t, stored.WebhookSecretSealed, systemWebhookSecret)
	assert.True(t, encryptionservice.IsEnvelope(stored.APIKeySealed))

	get := r.serve(t, http.MethodGet, managedToken, "")
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var held map[string]any
	require.NoError(t, sonic.Unmarshal(get.Body.Bytes(), &held))
	assert.Equal(t, map[string]any{
		"configured":  true,
		"provider":    "Postmark",
		"fromAddress": "no-reply@mail.trenova.app",
		"fingerprint": systememail.Fingerprint(&systememail.Credential{
			Provider:      email.ProviderPostmark,
			APIKey:        systemAPIKey,
			FromAddress:   "no-reply@mail.trenova.app",
			FromName:      "Trenova",
			WebhookSecret: systemWebhookSecret,
		}),
	}, held)
	r.assertNoSecretLeaked(t, put, get)
}

func TestManagedSystemEmailAnswersNoneBeforeOneIsSet(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
	get := r.serve(t, http.MethodGet, managedToken, "")
	require.Equal(t, http.StatusOK, get.Code)
	assert.JSONEq(t, `{"configured":false}`, get.Body.String())
}

func TestManagedSystemEmailRemoveLeavesNone(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
	require.Equal(t, http.StatusNoContent,
		r.serve(t, http.MethodPut, managedToken, systemEmailBody).Code)
	require.Equal(t, http.StatusNoContent, r.serve(t, http.MethodDelete, managedToken, "").Code)
	require.Equal(t, http.StatusNoContent, r.serve(t, http.MethodDelete, managedToken, "").Code)

	get := r.serve(t, http.MethodGet, managedToken, "")
	assert.JSONEq(t, `{"configured":false}`, get.Body.String())
	assert.Nil(t, r.repo.sender)
}

func TestManagedSystemEmailRefusesACallerWithoutTheManagedToken(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		for _, token := range []string{"", "0123456789abcdef0123456789abcdeX"} {
			rec := r.serve(t, method, token, systemEmailBody)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, method)
		}
	}
	assert.Nil(t, r.repo.sender)
}

func TestManagedSystemEmailRefusesAMalformedBody(t *testing.T) {
	t.Parallel()

	bodies := map[string]string{
		"not json":      `{"provider":`,
		"unknown field": strings.Replace(systemEmailBody, `"provider"`, `"host":"smtp.example.com","provider"`, 1),
		"bad provider":  strings.Replace(systemEmailBody, "Postmark", "SES", 1),
		"bad address":   strings.Replace(systemEmailBody, "no-reply@mail.trenova.app", "nobody", 1),
		"short key":     strings.Replace(systemEmailBody, systemAPIKey, "short", 1),
		"missing name":  strings.Replace(systemEmailBody, `"fromName":"Trenova"`, `"fromName":""`, 1),
	}
	for name, body := range bodies {
		r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
		rec := r.serve(t, http.MethodPut, managedToken, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Nil(t, r.repo.sender, name)
		r.assertNoSecretLeaked(t, rec)
	}
}

func TestManagedSystemEmailRefusesAnOversizedBody(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{})
	body := strings.Replace(
		systemEmailBody,
		`"fromName":"Trenova"`,
		`"fromName":"`+strings.Repeat("n", 9<<10)+`"`,
		1,
	)
	rec := r.serve(t, http.MethodPut, managedToken, body)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Nil(t, r.repo.sender)
}

func TestManagedSystemEmailHidesAStoreFailureAndLogsNoSecret(t *testing.T) {
	t.Parallel()

	r := newSystemEmailRig(t, &fakeSystemSenderRepo{err: errors.New("the database is down")})
	put := r.serve(t, http.MethodPut, managedToken, systemEmailBody)
	assert.Equal(t, http.StatusInternalServerError, put.Code)
	assert.NotContains(t, put.Body.String(), "database")
	require.NotEmpty(t, r.logs.All())
	r.assertNoSecretLeaked(t, put)
}
