package qboconnector

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newConnector(t *testing.T, qbo config.QuickBooksConfig) *Connector {
	t.Helper()
	cfg := &config.Config{
		App:        config.AppConfig{WebBaseURL: "https://app.example.com"},
		Accounting: config.AccountingConfig{QuickBooks: qbo},
	}
	conn, err := New(Params{Config: cfg, Logger: zap.NewNop()})
	require.NoError(t, err)
	return conn
}

func TestUnconfiguredConnectorIsUnavailable(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	assert.False(t, conn.Available())
	_, err := conn.AuthorizeURL("state")
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = conn.ExchangeCode(t.Context(), "code")
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
}

func TestConfiguredConnectorBuildsAuthorizeURLWithDerivedRedirect(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{ClientID: "id", ClientSecret: "secret"})
	require.True(t, conn.Available())
	raw, err := conn.AuthorizeURL("state-1")
	require.NoError(t, err)
	assert.Contains(t, raw, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fadmin%2Fintegrations%2Fquickbooks%2Fcallback")
	assert.Contains(t, raw, "state=state-1")
}

func TestUnknownEnvironmentIsRejected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Accounting: config.AccountingConfig{
		QuickBooks: config.QuickBooksConfig{Environment: "staging"},
	}}
	_, err := New(Params{Config: cfg, Logger: zap.NewNop()})
	require.Error(t, err)
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
