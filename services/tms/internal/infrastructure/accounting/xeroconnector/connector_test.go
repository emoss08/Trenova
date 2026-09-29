package xeroconnector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUnconfiguredInstanceHasNoApp(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.XeroConfig{})
	_, ok := provider.InstanceApp()
	assert.False(t, ok)
	_, err := provider.Bind(nil)
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, provider.ClassifyError(err))
	assert.Equal(t, integration.TypeXero, provider.IntegrationType())
	assert.Equal(t, xero.SignatureHeader, provider.WebhookSignatureHeader())
}

func TestConfiguredInstanceAppIsProductionWithTheWebhookKey(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.XeroConfig{
		ClientID:     " id ",
		ClientSecret: "secret",
		WebhookKey:   "hook-key",
	})
	app, ok := provider.InstanceApp()
	require.True(t, ok)
	assert.Equal(t, accountingsync.AppSourceInstance, app.Source)
	assert.Equal(t, accountingsync.AppEnvironmentProduction, app.Environment)
	assert.Equal(t, "id", app.ClientID)
	assert.Equal(t, "hook-key", app.WebhookVerifierToken)
	assert.Equal(t, "https://app.example.com/admin/integrations/xero/callback", provider.RedirectURL())

	app.ClientID = "changed"
	again, _ := provider.InstanceApp()
	assert.Equal(t, "id", again.ClientID)

	conn, err := provider.Bind(again)
	require.NoError(t, err)
	assert.Equal(t, integration.TypeXero, conn.IntegrationType())
	raw, err := conn.AuthorizeURL("state-1")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, "https://login.xero.com/identity/connect/authorize?"))
	assert.Contains(t, raw, "client_id=id")
	assert.Contains(t, raw, "state=state-1")
	assert.Contains(t, raw, "accounting.invoices")
	assert.Contains(t, raw,
		"redirect_uri=https%3A%2F%2Fapp.example.com%2Fadmin%2Fintegrations%2Fxero%2Fcallback")
}

func TestExplicitRedirectURLWins(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.XeroConfig{RedirectURL: "https://tms.example.com/cb"})
	assert.Equal(t, "https://tms.example.com/cb", provider.RedirectURL())
}

func TestBindingReusesTheConnectorForTheSameKeys(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.XeroConfig{})
	first, err := provider.Bind(tenantApp())
	require.NoError(t, err)
	second, err := provider.Bind(tenantApp())
	require.NoError(t, err)
	assert.Same(t, first, second)

	rotated := tenantApp()
	rotated.WebhookVerifierToken = "rotated"
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
	_, err = conn.AuthorizeURL("state")
	require.Error(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
	_, err = conn.ExchangeCode(t.Context(), "code")
	require.Error(t, err)
	require.ErrorIs(t, conn.VerifyApp(t.Context()), errNoRedirect)
}

func TestExchangeCodeUsesBasicAuthAndReportsTheTokenLifetimes(t *testing.T) {
	t.Parallel()

	var authorization string
	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path == "/connect/token" {
			return http.StatusOK, `{"access_token":"a.b.c","refresh_token":"r1","expires_in":1800,"token_type":"Bearer"}`
		}
		return 0, ""
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		fake.handler(t)(w, r)
	}))
	t.Cleanup(server.Close)

	provider := newProvider(t, config.XeroConfig{})
	provider.oauthOpts = []xero.Option{xero.WithIdentityURL(server.URL)}
	conn, err := provider.Bind(tenantApp())
	require.NoError(t, err)

	grant, err := conn.ExchangeCode(t.Context(), "code-1")
	require.NoError(t, err)
	assert.Equal(t, "a.b.c", grant.AccessToken)
	assert.Equal(t, "r1", grant.RefreshToken)
	assert.Equal(t, 30*time.Minute, grant.AccessTokenTTL)
	assert.Equal(t, 60*24*time.Hour, grant.RefreshTokenTTL)
	expected := base64.StdEncoding.EncodeToString([]byte("tenant-id:tenant-secret"))
	assert.Equal(t, "Basic "+expected, authorization)
}

func accessTokenWithEvent(eventID string) string {
	claims := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"authentication_event_id":"` + eventID + `"}`),
	)
	return "eyJhbGciOiJSUzI1NiJ9." + claims + ".signature"
}

func TestCompaniesListsTheOrganisationsOfThisConsent(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path != "/connections" {
			return 0, ""
		}
		return http.StatusOK, `[
		 {"id":"11111111-1111-4111-8111-111111111111","authEventId":"evt-1","tenantId":"` + testTenant + `","tenantType":"ORGANISATION","tenantName":"Acme Freight"},
		 {"id":"22222222-2222-4222-8222-222222222222","authEventId":"evt-1","tenantId":"33333333-3333-4333-8333-333333333333","tenantType":"PRACTICEMANAGER","tenantName":"Practice"},
		 {"id":"44444444-4444-4444-8444-444444444444","authEventId":"evt-1","tenantId":"55555555-5555-4555-8555-555555555555","tenantType":"ORGANISATION","tenantName":"Acme Logistics"}
		]`
	}}
	conn := testConnector(t, fake)

	companies, err := conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenWithEvent("evt-1"),
	}, "")
	require.NoError(t, err)
	assert.Equal(t, []services.AccountingCompany{
		{ID: testTenant, Name: "Acme Freight", ConnectionID: "11111111-1111-4111-8111-111111111111"},
		{
			ID:           "55555555-5555-4555-8555-555555555555",
			Name:         "Acme Logistics",
			ConnectionID: "44444444-4444-4444-8444-444444444444",
		},
	}, companies)
	call, ok := fake.find(http.MethodGet, "/connections")
	require.True(t, ok)
	assert.Equal(t, "evt-1", call.Query.Get("authEventId"))
}

func TestCompaniesFailsWithoutAnOrganisation(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path == "/connections" {
			return http.StatusOK, `[]`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	_, err := conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenWithEvent("evt-2"),
	}, "")
	require.ErrorIs(t, err, errNoCompany)

	_, err = conn.Companies(t.Context(), &services.AccountingTokenGrant{AccessToken: "opaque"}, "")
	require.ErrorIs(t, err, xero.ErrMalformedToken)
}

func TestReleaseCompaniesDeletesTheirConnections(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodDelete {
			if strings.HasSuffix(call.Path, "44444444-4444-4444-8444-444444444444") {
				return http.StatusNotFound, `{}`
			}
			return http.StatusNoContent, ``
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	err := conn.ReleaseCompanies(t.Context(), &services.AccountingTokenGrant{AccessToken: "token"},
		[]services.AccountingCompany{
			{ID: "a", ConnectionID: "11111111-1111-4111-8111-111111111111"},
			{ID: "b", ConnectionID: "44444444-4444-4444-8444-444444444444"},
			{ID: "c"},
		})
	require.NoError(t, err)
	deletes := fake.writes()
	require.Len(t, deletes, 2)
	assert.Equal(t, "/connections/11111111-1111-4111-8111-111111111111", deletes[0].Path)
}

func TestCompanyFactsReadsOrganisationAndActions(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Organisation":
			return http.StatusOK, `{"Organisations":[{"Name":"Acme","LegalName":"Acme Ltd","ShortCode":"!Ab12C","CountryCode":"NZ","BaseCurrency":"nzd","FinancialYearEndMonth":3,"FinancialYearEndDay":31,"PeriodLockDate":"2026-06-30T00:00:00","EndOfYearLockDate":"2026-03-31T00:00:00"}]}`
		case "/Organisation/Actions":
			return http.StatusOK, `{"Actions":[{"Name":"UseMulticurrency","Status":"ALLOWED"},{"Name":"CreateApprovedInvoice","Status":"ALLOWED"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	facts, err := conn.CompanyFacts(t.Context(), testTenant, "token")
	require.NoError(t, err)
	assert.Equal(t, "Acme", facts.CompanyName)
	assert.Equal(t, "Acme Ltd", facts.LegalName)
	assert.Equal(t, "NZ", facts.Country)
	assert.Equal(t, "NZD", facts.HomeCurrency)
	assert.Equal(t, "!Ab12C", facts.ShortCode)
	assert.True(t, facts.MultiCurrencyEnabled)
	assert.Equal(t, time.April, facts.FiscalYearStartMonth)
	require.NotNil(t, facts.BooksClosedThrough)
	assert.Equal(t, time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC).Unix(), *facts.BooksClosedThrough)
	call, ok := fake.find(http.MethodGet, "/Organisation")
	require.True(t, ok)
	assert.Equal(t, testTenant, call.Tenant)
}

func TestFiscalYearStartAndLockDates(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.January, fiscalYearStart(time.December))
	assert.Equal(t, time.July, fiscalYearStart(time.June))
	assert.Equal(t, time.Month(0), fiscalYearStart(0))

	early := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	assert.Nil(t, laterOf(nil, nil))
	assert.Equal(t, &early, laterOf(&early, nil))
	assert.Equal(t, &late, laterOf(nil, &late))
	assert.Equal(t, &late, laterOf(&early, &late))
	assert.Equal(t, &late, laterOf(&late, &early))
}

func verifyAppAgainst(t *testing.T, status int, body string) error {
	t.Helper()
	fake := &fakeXero{respond: func(xeroCall) (int, string) { return status, body }}
	return testConnector(t, fake).VerifyApp(t.Context())
}

func TestVerifyApp(t *testing.T) {
	t.Parallel()

	require.NoError(t, verifyAppAgainst(t, http.StatusBadRequest, `{"error":"invalid_grant"}`))
	err := verifyAppAgainst(t, http.StatusBadRequest, `{"error":"invalid_client"}`)
	require.ErrorIs(t, err, services.ErrAccountingAppRejected)
	err = verifyAppAgainst(t, http.StatusBadGateway, `{}`)
	require.Error(t, err)
	assert.NotErrorIs(t, err, services.ErrAccountingAppRejected)
}

func sign(key string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWebhookVerificationAndCompanies(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.XeroConfig{})
	conn, err := provider.Bind(tenantApp())
	require.NoError(t, err)

	body := []byte(`{"events":[
	 {"resourceId":"` + testInvoice + `","tenantId":"` + testTenant + `","eventCategory":"INVOICE","eventType":"UPDATE","eventDateUtc":"2026-09-29T10:00:00.000"},
	 {"resourceId":"` + testCustomer + `","tenantId":"` + testTenant + `","eventCategory":"CONTACT","eventType":"CREATE","eventDateUtc":"2026-09-29T10:00:01.000"}
	],"firstEventSequence":1,"lastEventSequence":2,"entropy":"X"}`)
	require.NoError(t, conn.VerifyWebhook(sign("tenant-webhook-key", body), body))
	require.ErrorIs(t, conn.VerifyWebhook(sign("other-key", body), body), xero.ErrInvalidSignature)
	require.Error(t, conn.VerifyWebhook("", body))

	realms, err := provider.WebhookRealmIDs(body)
	require.NoError(t, err)
	assert.Equal(t, []string{testTenant}, realms)

	empty := []byte(`{"events":[],"firstEventSequence":0,"lastEventSequence":0,"entropy":"Y"}`)
	require.NoError(t, conn.VerifyWebhook(sign("tenant-webhook-key", empty), empty))
	realms, err = provider.WebhookRealmIDs(empty)
	require.NoError(t, err)
	assert.Empty(t, realms)

	_, err = provider.WebhookRealmIDs([]byte(`not json`))
	require.Error(t, err)
}

func TestClassifyError(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	cases := []struct {
		name string
		err  error
		want accountingsync.ErrorCategory
	}{
		{"nil", nil, ""},
		{"not configured", ErrNotConfigured, accountingsync.ErrorCategoryConfiguration},
		{"revoked", &xero.OAuthError{Status: 400, Code: "invalid_grant"}, accountingsync.ErrorCategoryRevoked},
		{"bad client", &xero.OAuthError{Status: 400, Code: "invalid_client"}, accountingsync.ErrorCategoryUnauthorized},
		{"unauthorized", &xero.APIError{Status: 401}, accountingsync.ErrorCategoryUnauthorized},
		{"disconnected", &xero.APIError{Status: 403, Type: xero.TypeAuthUnsuccessful}, accountingsync.ErrorCategoryUnauthorized},
		{"rate limited", &xero.APIError{Status: 429}, accountingsync.ErrorCategoryRateLimited},
		{"server", &xero.APIError{Status: 503}, accountingsync.ErrorCategoryTransient},
		{"transport", &restx.TransportError{Err: errors.New("reset")}, accountingsync.ErrorCategoryTransient},
		{"deadline", context.DeadlineExceeded, accountingsync.ErrorCategoryTransient},
		{"validation", &xero.APIError{Status: 400}, accountingsync.ErrorCategoryUnknown},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, conn.ClassifyError(tc.err), tc.name)
	}
}

func TestTenantGateCapsConcurrencyPerOrganisation(t *testing.T) {
	t.Parallel()

	var inFlight, peak atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := inFlight.Add(1)
		for {
			seen := peak.Load()
			if current <= seen || peak.CompareAndSwap(seen, current) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	gate := newTenantGate()
	client := &http.Client{Transport: gate.transport(http.DefaultTransport)}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			if !assert.NoError(t, err) {
				return
			}
			req.Header.Set(tenantHeader, testTenant)
			resp, err := client.Do(req)
			if assert.NoError(t, err) {
				_ = resp.Body.Close()
			}
		})
	}
	require.Eventually(t, func() bool { return inFlight.Load() == maxConcurrentPerOrg },
		5*time.Second, 5*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int32(maxConcurrentPerOrg), peak.Load())
	assert.Zero(t, trackedTenants(gate))
}

func TestTenantGateGivesUpWhenTheCallerDoes(t *testing.T) {
	t.Parallel()

	gate := &tenantGate{limit: 1, byOrg: make(map[string]*tenantSlots)}
	entry := gate.join(testTenant)
	entry.slots <- struct{}{}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:1", nil)
	require.NoError(t, err)
	req.Header.Set(tenantHeader, testTenant)
	_, err = gate.transport(http.DefaultTransport).RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)

	<-entry.slots
	gate.leave(testTenant, entry)
	assert.Zero(t, trackedTenants(gate))
}

func trackedTenants(gate *tenantGate) int {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	return len(gate.byOrg)
}
