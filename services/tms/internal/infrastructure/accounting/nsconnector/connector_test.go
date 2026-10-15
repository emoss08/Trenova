package nsconnector

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

var testVerifier = strings.Repeat("v", 43)

func TestUnconfiguredInstanceHasNoApp(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.NetSuiteConfig{})
	_, ok := provider.InstanceApp()
	assert.False(t, ok)
	_, err := provider.Bind(nil)
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, provider.ClassifyError(err))
	assert.Equal(t, integration.TypeNetSuite, provider.IntegrationType())
	assert.Empty(t, provider.WebhookSignatureHeader())
	realms, err := provider.WebhookRealmIDs([]byte(`{}`))
	require.ErrorIs(t, err, errNoWebhooks)
	assert.Nil(t, realms)
}

func TestConfiguredInstanceAppAndAuthorizeURL(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.NetSuiteConfig{
		ClientID:     " " + testClientID + " ",
		ClientSecret: "secret",
	})
	app, ok := provider.InstanceApp()
	require.True(t, ok)
	assert.Equal(t, accountingsync.AppSourceInstance, app.Source)
	assert.Equal(t, accountingsync.AppEnvironmentProduction, app.Environment)
	assert.Equal(t, testClientID, app.ClientID)
	assert.Equal(t,
		"https://app.example.com/admin/integrations/netsuite/callback",
		provider.RedirectURL(),
	)

	app.ClientID = "changed"
	again, _ := provider.InstanceApp()
	assert.Equal(t, testClientID, again.ClientID)

	conn, err := provider.Bind(again)
	require.NoError(t, err)
	assert.Equal(t, integration.TypeNetSuite, conn.IntegrationType())
	raw, err := conn.AuthorizeURL(accountingsyncports.AccountingAuthorizeRequest{
		State:         "state-0123456789abcdefghij",
		CodeChallenge: testChallenge,
	})
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "system.netsuite.com", parsed.Host)
	assert.Equal(t, "/app/login/oauth2/authorize.nl", parsed.Path)
	query := parsed.Query()
	assert.Equal(t, testClientID, query.Get("client_id"))
	assert.Equal(t, "rest_webservices", query.Get("scope"))
	assert.Equal(t, testChallenge, query.Get("code_challenge"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Equal(t, provider.RedirectURL(), query.Get("redirect_uri"))
}

func TestBindingReusesTheConnectorForTheSameKeys(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.NetSuiteConfig{})
	first, err := provider.Bind(tenantApp())
	require.NoError(t, err)
	second, err := provider.Bind(tenantApp())
	require.NoError(t, err)
	assert.Same(t, first, second)

	rotated := tenantApp()
	rotated.ClientSecret = "rotated"
	third, err := provider.Bind(rotated)
	require.NoError(t, err)
	assert.NotSame(t, first, third)
}

func TestWithoutARedirectAddressOAuthIsAConfigurationFailure(t *testing.T) {
	t.Parallel()

	provider, err := New(Params{Config: &config.Config{}, Logger: zap.NewNop()})
	require.NoError(t, err)
	conn, err := provider.Bind(tenantApp())
	require.NoError(t, err)
	_, err = conn.AuthorizeURL(accountingsyncports.AccountingAuthorizeRequest{State: "state"})
	require.ErrorIs(t, err, errNoRedirect)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
	_, err = conn.ExchangeCode(t.Context(), &accountingsyncports.AccountingCodeExchange{Code: "c"})
	require.ErrorIs(t, err, errNoRedirect)
	_, err = conn.Refresh(t.Context(), &accountingsyncports.AccountingRefreshRequest{
		RefreshToken: "refresh",
	})
	require.ErrorIs(t, err, errNoRedirect)
	require.ErrorIs(t, conn.Revoke(t.Context(), &accountingsyncports.AccountingRevokeRequest{
		Token: "refresh",
	}), errNoRedirect)
	require.ErrorIs(t, conn.VerifyApp(t.Context()), errNoRedirect)
	nsConn, ok := conn.(*Connector)
	require.True(t, ok)
	_, err = nsConn.TokenFromCertificate(
		t.Context(),
		&accountingsyncports.AccountingCertificateGrant{
			RealmID: testAccount,
		},
	)
	require.ErrorIs(t, err, errNoRedirect)
}

func TestExchangeCodeSendsTheVerifierToTheCallbackAccount(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPost, tokenPath) {
			return fixtureReply(t, http.StatusOK, "token.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)

	grant, err := conn.ExchangeCode(t.Context(), &accountingsyncports.AccountingCodeExchange{
		Code:            "code-1",
		CallbackRealmID: "1234567_sb1",
		CodeVerifier:    testVerifier,
	})
	require.NoError(t, err)
	assert.Equal(t, "eyJraWQiOiJjLjEyMzQ1NjcifQ.refresh-seven-days.sig", grant.RefreshToken)
	assert.Equal(t, time.Hour, grant.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, grant.RefreshTokenTTL)

	call, ok := fake.find(http.MethodPost, tokenPath)
	require.True(t, ok)
	assert.Equal(t, "authorization_code", call.Form.Get("grant_type"))
	assert.Equal(t, "code-1", call.Form.Get("code"))
	assert.Equal(t, testVerifier, call.Form.Get("code_verifier"))
	assert.Equal(t,
		"Basic "+base64.StdEncoding.EncodeToString([]byte(testClientID+":tenant-secret")),
		call.Auth,
	)

	_, err = conn.ExchangeCode(t.Context(), &accountingsyncports.AccountingCodeExchange{
		Code:            "code-1",
		CallbackRealmID: "not an account",
		CodeVerifier:    testVerifier,
	})
	require.ErrorIs(t, err, netsuite.ErrInvalidCompanyRef)
	_, err = conn.ExchangeCode(t.Context(), &accountingsyncports.AccountingCodeExchange{
		Code:         "code-1",
		CodeVerifier: testVerifier,
	})
	require.ErrorIs(t, err, errNoAccount)
	assert.Equal(t, 1, fake.count(http.MethodPost, tokenPath))
}

func TestRefreshParsesCompositeRealmsAndKeepsAMissingRefreshTokenEmpty(t *testing.T) {
	t.Parallel()

	fixture := "token_refresh.json"
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPost, tokenPath) {
			return fixtureReply(t, http.StatusOK, fixture)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)

	grant, err := conn.Refresh(t.Context(), &accountingsyncports.AccountingRefreshRequest{
		RefreshToken: "stored-refresh",
		RealmID:      testRef().String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "eyJraWQiOiJjLjEyMzQ1NjcifQ.refreshed-access.sig", grant.AccessToken)
	assert.Empty(t, grant.RefreshToken)
	assert.Zero(t, grant.RefreshTokenTTL)
	assert.Equal(t, time.Hour, grant.AccessTokenTTL)
	call, ok := fake.find(http.MethodPost, tokenPath)
	require.True(t, ok)
	assert.Equal(t, "refresh_token", call.Form.Get("grant_type"))
	assert.Equal(t, "stored-refresh", call.Form.Get("refresh_token"))

	_, err = conn.Refresh(t.Context(), &accountingsyncports.AccountingRefreshRequest{
		RefreshToken: "stored-refresh",
		RealmID:      testAccount + "-s0",
	})
	require.ErrorIs(t, err, netsuite.ErrInvalidCompanyRef)
	assert.Equal(t, 1, fake.count(http.MethodPost, tokenPath))
}

func TestRefreshReportsARotatedRefreshTokenWithTheProfileLifetime(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPost, tokenPath) {
			return fixtureReply(t, http.StatusOK, "token_refresh_rotated.json")
		}
		return reply{}
	}})
	grant, err := conn.Refresh(t.Context(), &accountingsyncports.AccountingRefreshRequest{
		RefreshToken: "stored-refresh",
		RealmID:      testAccount,
	})
	require.NoError(t, err)
	assert.Equal(t, "eyJraWQiOiJjLjEyMzQ1NjcifQ.rotated-refresh.sig", grant.RefreshToken)
	assert.Equal(t, 7*24*time.Hour, grant.RefreshTokenTTL)
}

func TestRefreshWithAnExpiredTokenIsRevoked(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{respond: func(call nsCall) reply {
		return fixtureReply(t, http.StatusBadRequest, "invalid_grant.json")
	}})
	_, err := conn.Refresh(t.Context(), &accountingsyncports.AccountingRefreshRequest{
		RefreshToken: "expired",
		RealmID:      testAccount,
	})
	require.Error(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryRevoked, conn.ClassifyError(err))
}

func TestRevokeCallsTheAccountRevokeEndpoint(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPost, revokePath) {
			return reply{status: http.StatusOK}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	require.NoError(t, conn.Revoke(t.Context(), &accountingsyncports.AccountingRevokeRequest{
		Token:   "refresh-token",
		RealmID: testRef().String(),
	}))
	call, ok := fake.find(http.MethodPost, revokePath)
	require.True(t, ok)
	assert.Equal(t, "refresh-token", call.Form.Get("token"))
	assert.True(t, strings.HasPrefix(call.Auth, "Basic "))

	require.ErrorIs(t, conn.Revoke(t.Context(), &accountingsyncports.AccountingRevokeRequest{
		Token:   "refresh-token",
		RealmID: "bad realm",
	}), netsuite.ErrInvalidCompanyRef)
	require.NoError(
		t,
		conn.ReleaseCompanies(t.Context(), nil, []accountingsyncports.AccountingCompany{
			{ID: "x"},
		}),
	)
	assert.Equal(t, 1, fake.count(http.MethodPost, revokePath))
}

func TestVerifyAppChecksTheClientIDFormOnly(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	require.NoError(t, conn.VerifyApp(t.Context()))
	assert.Empty(t, fake.all())

	provider := newProvider(t, config.NetSuiteConfig{})
	for _, id := range []string{"short", strings.Repeat("a", 300), "has spaces in the client id!!"} {
		app := tenantApp()
		app.ClientID = id
		bound, err := provider.Bind(app)
		require.NoError(t, err)
		require.ErrorIs(
			t,
			bound.VerifyApp(t.Context()),
			accountingsyncports.ErrAccountingAppRejected,
		)
	}
}

func TestVerifyWebhookNeverAcceptsABody(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	require.ErrorIs(t, conn.VerifyWebhook("", []byte(`{}`)), errNoWebhooks)
	require.ErrorIs(t, conn.VerifyWebhook("signature", nil), errNoWebhooks)
}

func TestClassifyErrorMapsConnectionFailures(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	apiErr := func(status int, code, detail string) error {
		return &netsuite.APIError{
			Status:  status,
			Details: []netsuite.ErrorDetail{{Code: code, Detail: detail}},
		}
	}
	cases := []struct {
		name string
		err  error
		want accountingsync.ErrorCategory
	}{
		{"nil", nil, ""},
		{"not configured", ErrNotConfigured, accountingsync.ErrorCategoryConfiguration},
		{"unreachable", errUnreachable, accountingsync.ErrorCategoryConfiguration},
		{
			"revoked",
			&netsuite.OAuthError{Code: "invalid_grant"},
			accountingsync.ErrorCategoryRevoked,
		},
		{"certificate", errCertificateRevoked, accountingsync.ErrorCategoryRevoked},
		{
			"client",
			&netsuite.OAuthError{Code: "invalid_client"},
			accountingsync.ErrorCategoryUnauthorized,
		},
		{
			"login",
			apiErr(401, "INVALID_LOGIN", "Invalid login"),
			accountingsync.ErrorCategoryUnauthorized,
		},
		{"forbidden", apiErr(403, "", "Forbidden"), accountingsync.ErrorCategoryConfiguration},
		{
			"permission",
			apiErr(400, "USER_ERROR", "You need the 'Lists' permission"),
			accountingsync.ErrorCategoryConfiguration,
		},
		{
			"busy",
			apiErr(429, "CONCURRENCY_LIMIT_EXCEEDED", ""),
			accountingsync.ErrorCategoryRateLimited,
		},
		{"server", apiErr(500, "UNEXPECTED_ERROR", ""), accountingsync.ErrorCategoryTransient},
		{"deadline", context.DeadlineExceeded, accountingsync.ErrorCategoryTransient},
		{"other", errors.New("boom"), accountingsync.ErrorCategoryUnknown},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, conn.ClassifyError(tc.err), tc.name)
	}
}

func TestAccountGateBoundsConcurrentCallsPerAccount(t *testing.T) {
	t.Parallel()

	var current, peak atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := current.Add(1)
		for {
			seen := peak.Load()
			if now <= seen || peak.CompareAndSwap(seen, now) {
				break
			}
		}
		<-release
		current.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: newAccountGate().transport(server.Client().Transport)}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				server.URL+"/services/rest/record/v1/invoice/1",
				http.NoBody,
			)
			if !assert.NoError(t, err) {
				return
			}
			resp, err := client.Do(req)
			if assert.NoError(t, err) {
				_ = resp.Body.Close()
			}
		})
	}
	assert.Eventually(t, func() bool { return current.Load() == maxConcurrentPerAccount },
		time.Second, time.Millisecond)
	close(release)
	wg.Wait()
	assert.Equal(t, int32(maxConcurrentPerAccount), peak.Load())
}

func TestAccountGateHonoursCancellationAndKeysByHost(t *testing.T) {
	t.Parallel()

	gate := newAccountGate()
	parsed, err := url.Parse("https://1234567-SB1.suitetalk.api.netsuite.com/services/rest")
	require.NoError(t, err)
	assert.Equal(
		t,
		"1234567-sb1.suitetalk.api.netsuite.com",
		accountKey(&http.Request{URL: parsed}),
	)
	assert.Empty(t, accountKey(&http.Request{}))

	entry := gate.join("host")
	for range maxConcurrentPerAccount {
		entry.slots <- struct{}{}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://host/x", http.NoBody)
	require.NoError(t, err)
	_, err = gate.transport(http.DefaultTransport).RoundTrip(req)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	gate.leave("host", entry)
	assert.Empty(t, gate.byAccount)
}
