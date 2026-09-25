package qboconnector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newProvider(t *testing.T, qbo config.QuickBooksConfig) *Provider {
	t.Helper()
	cfg := &config.Config{
		App:        config.AppConfig{WebBaseURL: "https://app.example.com"},
		Accounting: config.AccountingConfig{QuickBooks: qbo},
	}
	provider, err := New(Params{Config: cfg, Logger: zap.NewNop()})
	require.NoError(t, err)
	return provider
}

func tenantApp(environment accountingsync.AppEnvironment) *services.AccountingApp {
	return &services.AccountingApp{
		Source:               accountingsync.AppSourceTenant,
		Environment:          environment,
		ClientID:             "tenant-id",
		ClientSecret:         "tenant-secret",
		WebhookVerifierToken: "tenant-verifier",
	}
}

func newConnector(t *testing.T, qbo config.QuickBooksConfig) *Connector {
	t.Helper()
	provider := newProvider(t, qbo)
	app, ok := provider.InstanceApp()
	if !ok {
		environment, parsed := accountingsync.ParseAppEnvironment(qbo.GetEnvironment())
		require.True(t, parsed)
		app = tenantApp(environment)
	}
	conn, err := provider.bind(app)
	require.NoError(t, err)
	return conn
}

func TestUnconfiguredInstanceHasNoApp(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	_, ok := provider.InstanceApp()
	assert.False(t, ok)
	_, err := provider.Bind(nil)
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, provider.ClassifyError(err))
}

func TestConfiguredInstanceAppBuildsAuthorizeURLWithDerivedRedirect(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		Environment:  "sandbox",
	})
	app, ok := provider.InstanceApp()
	require.True(t, ok)
	assert.Equal(t, accountingsync.AppSourceInstance, app.Source)
	assert.Equal(t, accountingsync.AppEnvironmentSandbox, app.Environment)
	assert.Equal(t,
		"https://app.example.com/admin/integrations/quickbooks/callback",
		provider.RedirectURL(),
	)

	conn, err := provider.Bind(app)
	require.NoError(t, err)
	raw, err := conn.AuthorizeURL("state-1")
	require.NoError(t, err)
	assert.Contains(t, raw, "client_id=id")
	assert.Contains(t, raw, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fadmin%2Fintegrations%2Fquickbooks%2Fcallback")
	assert.Contains(t, raw, "state=state-1")
}

func TestInstanceAppIsReturnedAsACopy(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{ClientID: "id", ClientSecret: "secret"})
	app, ok := provider.InstanceApp()
	require.True(t, ok)
	app.ClientID = "changed"
	again, _ := provider.InstanceApp()
	assert.Equal(t, "id", again.ClientID)
}

func TestTenantAppBindsWithoutAnInstanceApp(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	conn, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentProduction))
	require.NoError(t, err)
	raw, err := conn.AuthorizeURL("state-2")
	require.NoError(t, err)
	assert.Contains(t, raw, "client_id=tenant-id")
}

func TestBindingReusesTheConnectorForTheSameKeys(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	first, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	second, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	assert.Same(t, first, second)

	rotated := tenantApp(accountingsync.AppEnvironmentSandbox)
	rotated.ClientSecret = "rotated"
	third, err := provider.Bind(rotated)
	require.NoError(t, err)
	assert.NotSame(t, first, third)
}

func TestBindingKeepsEachAppsEnvironment(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	sandbox, err := provider.bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	production, err := provider.bind(tenantApp(accountingsync.AppEnvironmentProduction))
	require.NoError(t, err)
	assert.Equal(t, quickbooks.EnvironmentSandbox, sandbox.env)
	assert.Equal(t, quickbooks.EnvironmentProduction, production.env)
}

func TestBindingRejectsAnUnknownEnvironment(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	_, err := provider.Bind(tenantApp("Staging"))
	require.Error(t, err)
}

func TestWithoutARedirectAddressOAuthIsAConfigurationFailure(t *testing.T) {
	t.Parallel()

	provider, err := New(Params{Config: &config.Config{}, Logger: zap.NewNop()})
	require.NoError(t, err)
	assert.Empty(t, provider.RedirectURL())
	conn, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	_, err = conn.AuthorizeURL("state")
	require.Error(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
	_, err = conn.ExchangeCode(t.Context(), "code")
	require.Error(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
}

func TestUnknownEnvironmentIsRejected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Accounting: config.AccountingConfig{
		QuickBooks: config.QuickBooksConfig{Environment: "staging"},
	}}
	_, err := New(Params{Config: cfg, Logger: zap.NewNop()})
	require.Error(t, err)
}

func verifyAppAgainst(t *testing.T, status int, body string) error {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	provider := newProvider(t, config.QuickBooksConfig{})
	provider.oauthOpts = []quickbooks.Option{quickbooks.WithBaseURL(server.URL)}
	conn, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	return conn.VerifyApp(t.Context())
}

func TestVerifyAppAcceptsKeysThatOnlyTheProbeTokenFails(t *testing.T) {
	t.Parallel()

	require.NoError(t, verifyAppAgainst(t, http.StatusBadRequest, `{"error":"invalid_grant"}`))
}

func TestVerifyAppRejectsKeysTheProviderRefuses(t *testing.T) {
	t.Parallel()

	err := verifyAppAgainst(t, http.StatusUnauthorized, `{"error":"invalid_client"}`)
	require.ErrorIs(t, err, services.ErrAccountingAppRejected)
}

func TestVerifyAppPassesOnOtherFailures(t *testing.T) {
	t.Parallel()

	err := verifyAppAgainst(t, http.StatusBadGateway, `{}`)
	require.Error(t, err)
	assert.NotErrorIs(t, err, services.ErrAccountingAppRejected)
}

func TestVerifyWebhookUsesTheBoundAppsVerifier(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.QuickBooksConfig{})
	conn, err := provider.Bind(tenantApp(accountingsync.AppEnvironmentSandbox))
	require.NoError(t, err)
	body := []byte(`[]`)
	mac := hmac.New(sha256.New, []byte("tenant-verifier"))
	_, _ = mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	require.NoError(t, conn.VerifyWebhook(signature, body))
	require.Error(t, conn.VerifyWebhook("bogus", body))
}

func TestCompanyFactsCombinesInfoAndPreferences(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/company/123/companyinfo/123":
			_, _ = w.Write([]byte(`{"CompanyInfo":{"CompanyName":"Acme","LegalName":"Acme LLC","Country":"US"}}`))
		case "/v3/company/123/preferences":
			_, _ = w.Write([]byte(`{"Preferences":{"CurrencyPrefs":{"MultiCurrencyEnabled":true,"HomeCurrency":{"value":"cad"}},"AccountingInfoPrefs":{"BookCloseDate":"2026-06-30"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	conn := newConnector(t, config.QuickBooksConfig{})
	conn.apiOpts = []quickbooks.Option{quickbooks.WithBaseURL(server.URL)}

	facts, err := conn.CompanyFacts(t.Context(), "123", "token")
	require.NoError(t, err)
	assert.Equal(t, "Acme", facts.CompanyName)
	assert.Equal(t, "Acme LLC", facts.LegalName)
	assert.Equal(t, "CAD", facts.HomeCurrency)
	assert.True(t, facts.MultiCurrencyEnabled)
	require.NotNil(t, facts.BooksClosedThrough)
	expected := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC).Unix()
	assert.Equal(t, expected, *facts.BooksClosedThrough)
}

func TestClassifyError(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	cases := []struct {
		name string
		err  error
		want accountingsync.ErrorCategory
	}{
		{"nil", nil, ""},
		{"revoked", &quickbooks.OAuthError{StatusCode: 400, Code: "invalid_grant"}, accountingsync.ErrorCategoryRevoked},
		{"unauthorized", &quickbooks.FaultError{StatusCode: 401}, accountingsync.ErrorCategoryUnauthorized},
		{"forbidden", &quickbooks.FaultError{StatusCode: 403}, accountingsync.ErrorCategoryUnauthorized},
		{"rate limited", &quickbooks.FaultError{StatusCode: 429}, accountingsync.ErrorCategoryRateLimited},
		{"server", &quickbooks.FaultError{StatusCode: 503}, accountingsync.ErrorCategoryTransient},
		{"transport", &restx.TransportError{Err: errors.New("reset")}, accountingsync.ErrorCategoryTransient},
		{"deadline", context.DeadlineExceeded, accountingsync.ErrorCategoryTransient},
		{"validation", &quickbooks.FaultError{StatusCode: 400}, accountingsync.ErrorCategoryUnknown},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, conn.ClassifyError(tc.err), tc.name)
	}
}
