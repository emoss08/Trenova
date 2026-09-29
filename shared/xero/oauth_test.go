package xero_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClientID     = "0A1B2C3D4E5F6A7B8C9D0E1F2A3B4C5D"
	testClientSecret = "super-secret-client-value"
	testRedirect     = "https://app.example.com/admin/integrations/xero/callback"
)

func newOAuthClient(t *testing.T, handler http.HandlerFunc, scopes ...string) *xero.OAuthClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := xero.NewOAuthClient(xero.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
		Scopes:       scopes,
	}, xero.WithIdentityURL(server.URL), fastRetry())
	require.NoError(t, err)
	return client
}

func readForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	form, err := url.ParseQuery(string(body))
	require.NoError(t, err)
	return form
}

func basicAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(testClientID+":"+testClientSecret))
}

func jwt(t *testing.T, payload string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return header + "." + body + ".c2lnbmF0dXJl"
}

func TestNewOAuthClientRequiresConfig(t *testing.T) {
	t.Parallel()

	_, err := xero.NewOAuthClient(xero.OAuthConfig{ClientSecret: "s", RedirectURL: "r"})
	require.ErrorIs(t, err, xero.ErrClientIDRequired)
	_, err = xero.NewOAuthClient(xero.OAuthConfig{ClientID: "c", RedirectURL: "r"})
	require.ErrorIs(t, err, xero.ErrClientSecretRequired)
	_, err = xero.NewOAuthClient(xero.OAuthConfig{ClientID: "c", ClientSecret: "s"})
	require.ErrorIs(t, err, xero.ErrRedirectURLRequired)
}

func TestAuthorizeURLCarriesDefaultScopesStateAndRedirect(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(http.ResponseWriter, *http.Request) {})
	raw, err := client.AuthorizeURL("state-123")
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "login.xero.com", parsed.Host)
	assert.Equal(t, "/identity/connect/authorize", parsed.Path)
	query := parsed.Query()
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, testClientID, query.Get("client_id"))
	assert.Equal(t, testRedirect, query.Get("redirect_uri"))
	assert.Equal(t, "state-123", query.Get("state"))
	assert.Equal(t, strings.Join(xero.DefaultScopes(), " "), query.Get("scope"))
	assert.Contains(t, raw, "openid%20profile%20email%20offline_access", "spaces are percent encoded")
	assert.NotContains(t, raw, testClientSecret)

	_, err = client.AuthorizeURL(" ")
	require.ErrorIs(t, err, xero.ErrStateRequired)
}

func TestAuthorizeURLUsesConfiguredScopesAndLoginHost(t *testing.T) {
	t.Parallel()

	client, err := xero.NewOAuthClient(xero.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
		Scopes:       []string{"openid", " ", "accounting.settings.read"},
	}, xero.WithLoginURL("https://login.example.test/"))
	require.NoError(t, err)

	raw, err := client.AuthorizeURL("s")
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "login.example.test", parsed.Host)
	assert.Equal(t, "/identity/connect/authorize", parsed.Path)
	assert.Equal(t, "openid accounting.settings.read", parsed.Query().Get("scope"))
}

func TestDefaultScopes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{
		"openid", "profile", "email", "offline_access",
		"accounting.contacts", "accounting.settings", "accounting.invoices",
		"accounting.payments", "accounting.reports.aged.read",
	}, xero.DefaultScopes())
}

func TestExchangeCodePostsFormWithBasicAuth(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/connect/token", r.URL.Path)
		assert.Equal(t, basicAuth(), r.Header.Get("Authorization"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		form = readForm(t, r)
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.ExchangeCode(t.Context(), "auth-code-1")
	require.NoError(t, err)
	assert.Equal(t, "authorization_code", form.Get("grant_type"))
	assert.Equal(t, "auth-code-1", form.Get("code"))
	assert.Equal(t, testRedirect, form.Get("redirect_uri"))
	assert.Equal(t, "eyJhbGciOiJSUzI1NiJ9.access.token", token.AccessToken)
	assert.Equal(t, "f9a1b2c3d4e5.refresh", token.RefreshToken)
	assert.Equal(t, "eyJhbGciOiJSUzI1NiJ9.id.token", token.IDToken)
	assert.Contains(t, token.Scope, "accounting.invoices")
	assert.Equal(t, 30*time.Minute, token.ExpiresIn)

	_, err = client.ExchangeCode(t.Context(), " ")
	require.ErrorIs(t, err, xero.ErrCodeRequired)
}

func TestRefreshPostsTheRotatingToken(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/connect/token", r.URL.Path)
		form = readForm(t, r)
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.Refresh(t.Context(), "old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "old-refresh", form.Get("refresh_token"))
	assert.Empty(t, form.Get("redirect_uri"))
	assert.Equal(t, "f9a1b2c3d4e5.refresh", token.RefreshToken)

	_, err = client.Refresh(t.Context(), "")
	require.ErrorIs(t, err, xero.ErrTokenRequired)
}

func TestRefreshRejectedAsInvalidGrant(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "invalid_grant.json"))
	})

	_, err := client.Refresh(t.Context(), "old-refresh")
	require.Error(t, err)
	assert.True(t, xero.IsInvalidGrant(err))
	assert.False(t, xero.IsInvalidClient(err))
	assert.False(t, xero.IsTransient(err))
	var oauthErr *xero.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, http.StatusBadRequest, oauthErr.Status)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
	assert.NotContains(t, err.Error(), testClientSecret)
}

func TestRefreshRejectedAsInvalidClient(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "invalid_client.json"))
	})

	_, err := client.Refresh(t.Context(), "refresh")
	require.Error(t, err)
	assert.True(t, xero.IsInvalidClient(err))
	assert.False(t, xero.IsInvalidGrant(err))
	assert.NotContains(t, err.Error(), testClientSecret)
}

func TestUnauthorizedClientIsAnInvalidClient(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
	})

	_, err := client.ExchangeCode(t.Context(), "code")
	assert.True(t, xero.IsInvalidClient(err))
}

func TestTokenCallsAreNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := client.Refresh(t.Context(), "refresh")
	require.Error(t, err)
	assert.True(t, xero.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load(), "a rotated refresh token must not be sent twice")
}

func TestTokenResponseWithoutTokensIsRejected(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":1800,"access_token":"a"}`))
	})

	_, err := client.ExchangeCode(t.Context(), "code")
	require.ErrorIs(t, err, xero.ErrUnexpectedPayload)
}

func TestRevokeSendsTheRefreshTokenAsAForm(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/connect/revocation", r.URL.Path)
		assert.Equal(t, basicAuth(), r.Header.Get("Authorization"))
		form = readForm(t, r)
		w.WriteHeader(http.StatusOK)
	})

	require.NoError(t, client.Revoke(t.Context(), "refresh-to-revoke"))
	assert.Equal(t, "refresh-to-revoke", form.Get("token"))
	require.ErrorIs(t, client.Revoke(t.Context(), ""), xero.ErrTokenRequired)
}

func TestAuthEventIDReadsTheClaimWithoutVerifying(t *testing.T) {
	t.Parallel()

	token := jwt(t, `{"nbf":1790000000,"exp":1790001800,"iss":"https://identity.xero.com",`+
		`"xero_userid":"0a1b2c3d","authentication_event_id":"d99ecdfe-391d-43d2-b834-17636ba90e8d",`+
		`"scope":["accounting.invoices"]}`)
	eventID, err := xero.AuthEventID(token)
	require.NoError(t, err)
	assert.Equal(t, "d99ecdfe-391d-43d2-b834-17636ba90e8d", eventID)

	padded := strings.Split(token, ".")
	padded[1] += "=="
	eventID, err = xero.AuthEventID(strings.Join(padded, "."))
	require.NoError(t, err)
	assert.Equal(t, "d99ecdfe-391d-43d2-b834-17636ba90e8d", eventID)
}

func TestAuthEventIDRejectsWhatIsNotAnXeroAccessToken(t *testing.T) {
	t.Parallel()

	_, err := xero.AuthEventID("")
	require.ErrorIs(t, err, xero.ErrMalformedToken)
	_, err = xero.AuthEventID("only.two")
	require.ErrorIs(t, err, xero.ErrMalformedToken)
	_, err = xero.AuthEventID("a.!!!.c")
	require.ErrorIs(t, err, xero.ErrMalformedToken)
	_, err = xero.AuthEventID("a." + base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".c")
	require.ErrorIs(t, err, xero.ErrMalformedToken)
	_, err = xero.AuthEventID(jwt(t, `{"sub":"x"}`))
	require.ErrorIs(t, err, xero.ErrAuthEventMissing)
}
