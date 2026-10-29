package netsuite_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClientID     = "8f1e0c2b7a6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f"
	testClientSecret = "super-secret-client-value"
	testRedirect     = "https://app.example.com/admin/integrations/netsuite/callback"
	testState        = "state-0123456789abcdefghij"
	testVerifier     = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	testTokenPath    = "/services/rest/auth/oauth2/v1/token"
)

func testOAuthConfig() netsuite.OAuthConfig {
	return netsuite.OAuthConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirect,
	}
}

func newOAuthClient(
	t *testing.T,
	handler http.HandlerFunc,
	opts ...netsuite.Option,
) (*netsuite.OAuthClient, string) {
	t.Helper()
	server := newServer(t, handler)
	all := append([]netsuite.Option{
		netsuite.WithBaseURL(server.URL),
		netsuite.WithHTTPClient(server.Client()),
		fastRetry(),
	}, opts...)
	client, err := netsuite.NewOAuthClient(testOAuthConfig(), all...)
	require.NoError(t, err)
	return client, server.URL
}

func readForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	form, err := url.ParseQuery(string(body))
	require.NoError(t, err)
	return form
}

func assertBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	id, secret, ok := r.BasicAuth()
	require.True(t, ok, "the client authenticates with HTTP Basic")
	assert.Equal(t, testClientID, id)
	assert.Equal(t, testClientSecret, secret)
	assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(testClientID+":"+testClientSecret)), r.Header.Get("Authorization"))
}

func assertTokenRequest(t *testing.T, r *http.Request, path string) url.Values {
	t.Helper()
	assert.Equal(t, http.MethodPost, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
	form := readForm(t, r)
	assert.Empty(t, form.Get("client_secret"), "the secret travels only in the header")
	return form
}

func TestNewOAuthClientRequiresConfig(t *testing.T) {
	t.Parallel()

	_, err := netsuite.NewOAuthClient(netsuite.OAuthConfig{ClientSecret: "s", RedirectURL: "r"})
	require.ErrorIs(t, err, netsuiteerrors.ErrClientIDRequired)
	_, err = netsuite.NewOAuthClient(netsuite.OAuthConfig{ClientID: "c", RedirectURL: "r"})
	require.ErrorIs(t, err, netsuiteerrors.ErrClientSecretRequired)
	_, err = netsuite.NewOAuthClient(netsuite.OAuthConfig{ClientID: "c", ClientSecret: "s"})
	require.ErrorIs(t, err, netsuiteerrors.ErrRedirectURLRequired)
}

func TestAuthorizeURL(t *testing.T) {
	t.Parallel()

	client, err := netsuite.NewOAuthClient(testOAuthConfig())
	require.NoError(t, err)

	verifier, err := tokenutils.PKCEVerifier()
	require.NoError(t, err)
	challenge := tokenutils.PKCEChallenge(verifier)

	raw, err := client.AuthorizeURL(netsuite.AuthorizeRequest{
		State:         testState,
		CodeChallenge: challenge,
	})
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "system.netsuite.com", parsed.Host)
	assert.Equal(t, "/app/login/oauth2/authorize.nl", parsed.Path)
	query := parsed.Query()
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, testClientID, query.Get("client_id"))
	assert.Equal(t, testRedirect, query.Get("redirect_uri"))
	assert.Equal(t, "rest_webservices", query.Get("scope"))
	assert.Equal(t, testState, query.Get("state"))
	assert.Equal(t, challenge, query.Get("code_challenge"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Len(t, query, 7)
	assert.NotContains(t, raw, testClientSecret)
	assert.NotContains(t, raw, verifier)
}

func TestAuthorizeURLValidates(t *testing.T) {
	t.Parallel()

	client, err := netsuite.NewOAuthClient(testOAuthConfig(),
		netsuite.WithLoginURL("https://login.test/"))
	require.NoError(t, err)
	challenge := tokenutils.PKCEChallenge(testVerifier)

	raw, err := client.AuthorizeURL(netsuite.AuthorizeRequest{
		State: testState, CodeChallenge: challenge,
	})
	require.NoError(t, err)
	assert.Contains(t, raw, "https://login.test/app/login/oauth2/authorize.nl?")

	for _, state := range []string{"", "short", "state with spaces 0123456789"} {
		_, err = client.AuthorizeURL(netsuite.AuthorizeRequest{
			State: state, CodeChallenge: challenge,
		})
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidState, state)
	}
	for _, bad := range []string{"", testVerifier[:42], challenge[:42] + "+"} {
		_, err = client.AuthorizeURL(netsuite.AuthorizeRequest{
			State: testState, CodeChallenge: bad,
		})
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidChallenge, bad)
	}
}

func TestTokenURLIsPerAccount(t *testing.T) {
	t.Parallel()

	client, err := netsuite.NewOAuthClient(testOAuthConfig())
	require.NoError(t, err)
	tokenURL, err := client.TokenURL("1234567_SB1")
	require.NoError(t, err)
	assert.Equal(t,
		"https://1234567-sb1.suitetalk.api.netsuite.com/services/rest/auth/oauth2/v1/token",
		tokenURL)
	_, err = client.TokenURL("evil.example/x")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)
}

func TestExchangeCodeSendsBasicAuthAndVerifier(t *testing.T) {
	t.Parallel()

	var form url.Values
	client, _ := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		form = assertTokenRequest(t, r, testTokenPath)
		_, _ = w.Write(fixture(t, "token.json"))
	})

	token, err := client.ExchangeCode(t.Context(), netsuite.CodeExchange{
		AccountID:    testAccount,
		Code:         "auth-code-1",
		CodeVerifier: testVerifier,
	})
	require.NoError(t, err)
	assert.Equal(t, "authorization_code", form.Get("grant_type"))
	assert.Equal(t, "auth-code-1", form.Get("code"))
	assert.Equal(t, testRedirect, form.Get("redirect_uri"))
	assert.Equal(t, testVerifier, form.Get("code_verifier"))
	assert.Empty(t, form.Get("client_id"))
	assert.Equal(t, "eyJraWQiOiJjLjEyMzQ1NjcifQ.refresh-seven-days.sig", token.RefreshToken)
	assert.Contains(t, token.AccessToken, ".access.")
	assert.Equal(t, "bearer", token.TokenType)
	assert.Equal(t, time.Hour, token.ExpiresIn)
}

func TestExchangeCodeValidates(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request is sent")
	})
	_, err := client.ExchangeCode(t.Context(), netsuite.CodeExchange{
		AccountID: testAccount, CodeVerifier: testVerifier,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrCodeRequired)
	_, err = client.ExchangeCode(t.Context(), netsuite.CodeExchange{
		AccountID: testAccount, Code: "c", CodeVerifier: testVerifier[:42],
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidVerifier)
	_, err = client.ExchangeCode(t.Context(), netsuite.CodeExchange{
		AccountID: "../evil", Code: "c", CodeVerifier: testVerifier,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)
}

func TestExchangeWithoutRefreshTokenIsUnexpected(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "token_refresh.json"))
	})
	_, err := client.ExchangeCode(t.Context(), netsuite.CodeExchange{
		AccountID: testAccount, Code: "c", CodeVerifier: testVerifier,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrUnexpectedPayload)
}

func TestRefreshWithoutARefreshTokenInTheResponse(t *testing.T) {
	t.Parallel()

	var form url.Values
	client, _ := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		form = assertTokenRequest(t, r, testTokenPath)
		_, _ = w.Write(fixture(t, "token_refresh.json"))
	})

	token, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "stored-refresh",
	})
	require.NoError(t, err)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "stored-refresh", form.Get("refresh_token"))
	assert.Empty(t, token.RefreshToken, "a missing refresh token is never invented")
	assert.Contains(t, token.AccessToken, "refreshed-access")
	assert.Equal(t, time.Hour, token.ExpiresIn)

	_, err = client.Refresh(t.Context(), netsuite.RefreshRequest{AccountID: testAccount})
	require.ErrorIs(t, err, netsuiteerrors.ErrTokenRequired)
}

func TestRefreshWithARotatedRefreshToken(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "token_refresh_rotated.json"))
	})
	token, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "stored-refresh",
	})
	require.NoError(t, err)
	assert.Equal(t, "eyJraWQiOiJjLjEyMzQ1NjcifQ.rotated-refresh.sig", token.RefreshToken)
	assert.Equal(t, time.Hour, token.ExpiresIn)
}

func TestRevoke(t *testing.T) {
	t.Parallel()

	var form url.Values
	client, _ := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		form = assertTokenRequest(t, r, "/services/rest/auth/oauth2/v1/revoke")
		w.WriteHeader(http.StatusOK)
	})
	require.NoError(t, client.Revoke(t.Context(), netsuite.RevokeRequest{
		AccountID: testAccount, Token: "stored-refresh",
	}))
	assert.Equal(t, "stored-refresh", form.Get("token"))

	require.ErrorIs(t, client.Revoke(t.Context(), netsuite.RevokeRequest{
		AccountID: testAccount,
	}), netsuiteerrors.ErrTokenRequired)
}

func TestRefreshRejectedAsInvalidGrant(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, serveFixture(t, http.StatusBadRequest, "invalid_grant.json"))
	_, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "expired-refresh",
	})
	require.Error(t, err)
	assert.True(t, netsuiteerrors.IsInvalidGrant(err))
	assert.False(t, netsuiteerrors.IsInvalidClient(err))
	assert.False(t, netsuiteerrors.IsTransient(err))
	var oauthErr *netsuiteerrors.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, http.StatusBadRequest, oauthErr.Status)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
}

func TestRefreshRejectedAsInvalidClient(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, serveFixture(t, http.StatusUnauthorized, "invalid_client.json"))
	_, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "refresh",
	})
	require.Error(t, err)
	assert.True(t, netsuiteerrors.IsInvalidClient(err))
	assert.False(t, netsuiteerrors.IsInvalidGrant(err))
	assert.NotContains(t, err.Error(), testClientSecret)
}

func TestOAuthErrorsRedactSecrets(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"bad ` +
			testClientSecret + ` for refresh-value-123"}`))
	})
	_, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "refresh-value-123",
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testClientSecret)
	assert.NotContains(t, err.Error(), "refresh-value-123")
}

func TestTokenCallsAreNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newOAuthClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})
	_, err := client.Refresh(t.Context(), netsuite.RefreshRequest{
		AccountID: testAccount, RefreshToken: "refresh",
	})
	require.Error(t, err)
	assert.True(t, netsuiteerrors.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load())
}
