package quickbooks_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClientID     = "ABclientid123"
	testClientSecret = "super-secret-client-value"
	testRedirect     = "https://app.example.com/admin/integrations/quickbooks/callback"
)

func newOAuthClient(t *testing.T, handler http.HandlerFunc) *quickbooks.OAuthClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := quickbooks.NewOAuthClient(quickbooks.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
	}, quickbooks.WithBaseURL(server.URL), fastRetry())
	require.NoError(t, err)
	return client
}

func TestNewOAuthClientRequiresConfig(t *testing.T) {
	t.Parallel()

	_, err := quickbooks.NewOAuthClient(quickbooks.OAuthConfig{ClientSecret: "s", RedirectURL: "r"})
	require.ErrorIs(t, err, quickbooks.ErrClientIDRequired)
	_, err = quickbooks.NewOAuthClient(quickbooks.OAuthConfig{ClientID: "c", RedirectURL: "r"})
	require.ErrorIs(t, err, quickbooks.ErrClientSecretRequired)
	_, err = quickbooks.NewOAuthClient(quickbooks.OAuthConfig{ClientID: "c", ClientSecret: "s"})
	require.ErrorIs(t, err, quickbooks.ErrRedirectURLRequired)
}

func TestAuthorizeURLCarriesStateScopeAndRedirect(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(http.ResponseWriter, *http.Request) {})
	raw, err := client.AuthorizeURL("state-123")
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "appcenter.intuit.com", parsed.Host)
	query := parsed.Query()
	assert.Equal(t, testClientID, query.Get("client_id"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, "com.intuit.quickbooks.accounting", query.Get("scope"))
	assert.Equal(t, testRedirect, query.Get("redirect_uri"))
	assert.Equal(t, "state-123", query.Get("state"))
	assert.NotContains(t, raw, testClientSecret)

	_, err = client.AuthorizeURL(" ")
	require.ErrorIs(t, err, quickbooks.ErrStateRequired)
}

func TestExchangeCodePostsFormWithBasicAuth(t *testing.T) {
	t.Parallel()

	var gotAuth, gotContentType string
	var gotForm url.Values
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oauth2/v1/tokens/bearer", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.ExchangeCode(t.Context(), "auth-code-1")
	require.NoError(t, err)
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte(testClientID+":"+testClientSecret))
	assert.Equal(t, expected, gotAuth)
	assert.Equal(t, "application/x-www-form-urlencoded", gotContentType)
	assert.Equal(t, "authorization_code", gotForm.Get("grant_type"))
	assert.Equal(t, "auth-code-1", gotForm.Get("code"))
	assert.Equal(t, testRedirect, gotForm.Get("redirect_uri"))
	assert.Equal(t, "eyJ.access.token", token.AccessToken)
	assert.Equal(t, "AB11.refresh.token", token.RefreshToken)
	assert.Equal(t, time.Hour, token.AccessTokenTTL)
	assert.Equal(t, 8726400*time.Second, token.RefreshTokenTTL)
}

func TestRefreshRejectedAsInvalidGrant(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		assert.Equal(t, "refresh_token", form.Get("grant_type"))
		assert.Equal(t, "old-refresh", form.Get("refresh_token"))
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "invalid_grant.json"))
	})

	_, err := client.Refresh(t.Context(), "old-refresh")
	require.Error(t, err)
	assert.True(t, quickbooks.IsInvalidGrant(err))
	assert.False(t, quickbooks.IsTransient(err))
	assert.NotContains(t, err.Error(), testClientSecret)
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
	assert.True(t, quickbooks.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load())
}

func TestTokenResponseWithoutTokensIsRejected(t *testing.T) {
	t.Parallel()

	client := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer","expires_in":3600}`))
	})

	_, err := client.ExchangeCode(t.Context(), "code")
	require.ErrorIs(t, err, quickbooks.ErrUnexpectedPayload)
}

func TestRevokeSendsTokenAsJSON(t *testing.T) {
	t.Parallel()

	var got map[string]string
	client := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/oauth2/tokens/revoke", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		_ = sonic.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	})

	require.NoError(t, client.Revoke(t.Context(), "refresh-to-revoke"))
	assert.Equal(t, "refresh-to-revoke", got["token"])
	require.ErrorIs(t, client.Revoke(t.Context(), ""), quickbooks.ErrTokenRequired)
}
