package businesscentral_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClientID     = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
	testClientSecret = "super-secret-client-value"
	testRedirect     = "https://app.example.com/admin/integrations/businesscentral/callback"
)

func newOAuthClient(t *testing.T, handler http.HandlerFunc) *businesscentral.OAuthClient {
	t.Helper()
	server := newServer(t, handler)
	client, err := businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
	}, businesscentral.WithLoginURL(server.URL), fastRetry())
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

func jwt(payload string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return header + "." + body + ".c2lnbmF0dXJl"
}

func TestNewOAuthClientRequiresConfig(t *testing.T) {
	t.Parallel()

	_, err := businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
		ClientSecret: "s", RedirectURL: "r",
	})
	require.ErrorIs(t, err, businesscentral.ErrClientIDRequired)
	_, err = businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
		ClientID: "c", RedirectURL: "r",
	})
	require.ErrorIs(t, err, businesscentral.ErrClientSecretRequired)
	_, err = businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
		ClientID: "c", ClientSecret: "s",
	})
	require.ErrorIs(t, err, businesscentral.ErrRedirectURLRequired)
}

func TestAuthorizeURL(t *testing.T) {
	t.Parallel()

	client, err := businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
	})
	require.NoError(t, err)

	raw, err := client.AuthorizeURL("state-123")
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "login.microsoftonline.com", parsed.Host)
	assert.Equal(t, "/organizations/oauth2/v2.0/authorize", parsed.Path)
	query := parsed.Query()
	assert.Equal(t, testClientID, query.Get("client_id"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, testRedirect, query.Get("redirect_uri"))
	assert.Equal(t, "query", query.Get("response_mode"))
	assert.Equal(t, "state-123", query.Get("state"))
	assert.Equal(t, "select_account", query.Get("prompt"))
	assert.Equal(t,
		"https://api.businesscentral.dynamics.com/Financials.ReadWrite.All offline_access",
		query.Get("scope"))
	assert.Contains(t, raw, "Financials.ReadWrite.All%20offline_access")
	assert.NotContains(t, raw, testClientSecret)

	_, err = client.AuthorizeURL(" ")
	require.ErrorIs(t, err, businesscentral.ErrStateRequired)
}

func TestExchangeCodePostsTheForm(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/organizations/oauth2/v2.0/token", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Empty(t, r.Header.Get("Authorization"))
		form = readForm(t, r)
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.ExchangeCode(t.Context(), "auth-code-1")
	require.NoError(t, err)
	assert.Equal(t, "authorization_code", form.Get("grant_type"))
	assert.Equal(t, "auth-code-1", form.Get("code"))
	assert.Equal(t, testClientID, form.Get("client_id"))
	assert.Equal(t, testClientSecret, form.Get("client_secret"))
	assert.Equal(t, testRedirect, form.Get("redirect_uri"))
	assert.Equal(t, strings.Join(businesscentral.DefaultScopes(), " "), form.Get("scope"))
	assert.Equal(t, "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.access.sig", token.AccessToken)
	assert.Equal(t, "0.AQoA-rotated-refresh", token.RefreshToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, 4307*time.Second, token.ExpiresIn)
	assert.Equal(t, 4307*time.Second, token.ExtExpiresIn)
	assert.Contains(t, token.Scope, "Financials.ReadWrite.All")

	_, err = client.ExchangeCode(t.Context(), " ")
	require.ErrorIs(t, err, businesscentral.ErrCodeRequired)
}

func TestRefreshPostsTheRotatingToken(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		form = readForm(t, r)
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.Refresh(t.Context(), "old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "old-refresh", form.Get("refresh_token"))
	assert.Equal(t, testClientSecret, form.Get("client_secret"))
	assert.Equal(t, testRedirect, form.Get("redirect_uri"))
	assert.Equal(t, "0.AQoA-rotated-refresh", token.RefreshToken)
	assert.Equal(t, 90*24*time.Hour, businesscentral.RefreshTokenLifetime)

	_, err = client.Refresh(t.Context(), "")
	require.ErrorIs(t, err, businesscentral.ErrTokenRequired)
}

func TestRefreshRejectedAsInvalidGrant(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, serveFixture(t, http.StatusBadRequest, "invalid_grant.json"))
	_, err := client.Refresh(t.Context(), "old-refresh")
	require.Error(t, err)
	assert.True(t, businesscentral.IsInvalidGrant(err))
	assert.False(t, businesscentral.IsInvalidClient(err))
	assert.False(t, businesscentral.IsTransient(err))
	var oauthErr *businesscentral.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, http.StatusBadRequest, oauthErr.Status)
	assert.Equal(t, []int{70008}, oauthErr.ErrorCodes)
	assert.NotContains(t, err.Error(), testClientSecret)
}

func TestInvalidGrantByErrorCode(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"70000", "70008", "700082", "50173"} {
		client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"interaction_required","error_codes":[` + code + `]}`))
		})
		_, err := client.Refresh(t.Context(), "refresh")
		assert.True(t, businesscentral.IsInvalidGrant(err), code)
		assert.False(t, businesscentral.IsInvalidClient(err), code)
	}
}

func TestRefreshRejectedAsInvalidClient(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, serveFixture(t, http.StatusUnauthorized, "invalid_client.json"))
	_, err := client.Refresh(t.Context(), "refresh")
	require.Error(t, err)
	assert.True(t, businesscentral.IsInvalidClient(err))
	assert.False(t, businesscentral.IsInvalidGrant(err))
	assert.NotContains(t, err.Error(), testClientSecret)

	for _, code := range []string{"700016", "7000222"} {
		client = newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_codes":[` + code + `]}`))
		})
		_, err = client.Refresh(t.Context(), "refresh")
		assert.True(t, businesscentral.IsInvalidClient(err), code)
		assert.False(t, businesscentral.IsInvalidGrant(err), code)
	}
}

func TestOAuthErrorsRedactTheSecretAndGrant(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"bad ` +
			testClientSecret + ` for refresh-value-123"}`))
	})
	_, err := client.Refresh(t.Context(), "refresh-value-123")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testClientSecret)
	assert.NotContains(t, err.Error(), "refresh-value-123")
}

func TestProbeCredentialsSendsADummyRefreshToken(t *testing.T) {
	t.Parallel()

	var form url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		form = readForm(t, r)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "invalid_grant.json"))
	})
	err := client.ProbeCredentials(t.Context())
	assert.True(t, businesscentral.IsInvalidGrant(err), "the app itself was accepted")
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.NotEmpty(t, form.Get("refresh_token"))

	client = newOAuthClient(t, serveFixture(t, http.StatusUnauthorized, "invalid_client.json"))
	assert.True(t, businesscentral.IsInvalidClient(client.ProbeCredentials(t.Context())))
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
	assert.True(t, businesscentral.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load(), "a rotated refresh token must not be sent twice")
}

func TestTokenResponseWithoutTokensIsRejected(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"a"}`))
	})
	_, err := client.ExchangeCode(t.Context(), "code")
	require.ErrorIs(t, err, businesscentral.ErrUnexpectedPayload)
}

func TestTenantIDFromAccessToken(t *testing.T) {
	t.Parallel()

	token := jwt(`{"aud":"https://api.businesscentral.dynamics.com","tid":"` +
		strings.ToUpper(testTenant) + `","scp":"Financials.ReadWrite.All"}`)
	tenant, err := businesscentral.TenantIDFromAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, testTenant, tenant)

	segments := strings.Split(token, ".")
	segments[1] += "=="
	tenant, err = businesscentral.TenantIDFromAccessToken(strings.Join(segments, "."))
	require.NoError(t, err)
	assert.Equal(t, testTenant, tenant)
}

func TestTenantIDFromAccessTokenRejectsBadTokens(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "only.two", "a.!!!.c",
		"a." + base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".c"} {
		_, err := businesscentral.TenantIDFromAccessToken(raw)
		require.ErrorIs(t, err, businesscentral.ErrMalformedToken, raw)
	}
	_, err := businesscentral.TenantIDFromAccessToken(jwt(`{"sub":"x"}`))
	require.ErrorIs(t, err, businesscentral.ErrTenantClaim)
	_, err = businesscentral.TenantIDFromAccessToken(jwt(`{"tid":"../../etc"}`))
	require.ErrorIs(t, err, businesscentral.ErrTenantClaim)
}
