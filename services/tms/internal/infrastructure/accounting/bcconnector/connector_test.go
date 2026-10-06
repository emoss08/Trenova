package bcconnector

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUnconfiguredInstanceHasNoApp(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.BusinessCentralConfig{})
	_, ok := provider.InstanceApp()
	assert.False(t, ok)
	_, err := provider.Bind(nil)
	require.ErrorIs(t, err, ErrNotConfigured)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, provider.ClassifyError(err))
	assert.Equal(t, integration.TypeBusinessCentral, provider.IntegrationType())
	assert.Empty(t, provider.WebhookSignatureHeader())
	realms, err := provider.WebhookRealmIDs([]byte(`{"value":[]}`))
	require.NoError(t, err)
	assert.Nil(t, realms)
}

func TestConfiguredInstanceAppAndAddresses(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.BusinessCentralConfig{
		ClientID:       " id ",
		ClientSecret:   "secret",
		WebhookBaseURL: "https://hooks.example.com/",
	})
	app, ok := provider.InstanceApp()
	require.True(t, ok)
	assert.Equal(t, accountingsync.AppSourceInstance, app.Source)
	assert.Equal(t, accountingsync.AppEnvironmentProduction, app.Environment)
	assert.Equal(t, "id", app.ClientID)
	assert.Empty(t, app.WebhookVerifierToken)
	assert.Equal(t,
		"https://app.example.com/admin/integrations/business-central/callback",
		provider.RedirectURL(),
	)
	assert.Equal(t, "https://hooks.example.com", provider.NotificationBaseURL())

	app.ClientID = "changed"
	again, _ := provider.InstanceApp()
	assert.Equal(t, "id", again.ClientID)

	conn, err := provider.Bind(again)
	require.NoError(t, err)
	assert.Equal(t, integration.TypeBusinessCentral, conn.IntegrationType())
	raw, err := conn.AuthorizeURL("state-1")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw,
		"https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?"))
	assert.Contains(t, raw, "client_id=id")
	assert.Contains(t, raw, "prompt=select_account")
	assert.Contains(t, raw, "offline_access")
}

func TestNotificationBaseDefaultsToTheAPIBase(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.BusinessCentralConfig{})
	assert.Equal(t, "https://app.example.com/api/v1", provider.NotificationBaseURL())
}

func TestBindingReusesTheConnectorForTheSameKeys(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.BusinessCentralConfig{})
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
	_, err = conn.AuthorizeURL("state")
	require.ErrorIs(t, err, errNoRedirect)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
	_, err = conn.ExchangeCode(t.Context(), "code")
	require.ErrorIs(t, err, errNoRedirect)
	_, err = conn.Refresh(t.Context(), "refresh")
	require.ErrorIs(t, err, errNoRedirect)
	require.ErrorIs(t, conn.VerifyApp(t.Context()), errNoRedirect)
}

func TestExchangeCodeAndRefreshReportTheTokenLifetimes(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.Path == "/organizations/oauth2/v2.0/token" {
			return http.StatusOK, readFixture(t, "token.json")
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	grant, err := conn.ExchangeCode(t.Context(), "code-1")
	require.NoError(t, err)
	assert.Equal(t, "0.AQoA-rotated-refresh", grant.RefreshToken)
	assert.Equal(t, 4307*time.Second, grant.AccessTokenTTL)
	assert.Equal(t, 90*24*time.Hour, grant.RefreshTokenTTL)

	refreshed, err := conn.Refresh(t.Context(), "r0")
	require.NoError(t, err)
	assert.Equal(t, grant.AccessToken, refreshed.AccessToken)
	assert.Equal(t, 2, fake.count(http.MethodPost, "/organizations/oauth2/v2.0/token"))
}

func TestRevokeSucceedsWithoutACall(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)
	require.NoError(t, conn.Revoke(t.Context(), "refresh-token"))
	require.NoError(t, conn.ReleaseCompanies(t.Context(), nil, []services.AccountingCompany{{ID: "x"}}))
	assert.Empty(t, fake.all())
}

func TestVerifyAppTreatsInvalidGrantAsAcceptedAndInvalidClientAsRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		status  int
		fixture string
		want    error
	}{
		{"accepted", http.StatusBadRequest, "invalid_grant.json", nil},
		{"rejected", http.StatusUnauthorized, "invalid_client.json", services.ErrAccountingAppRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conn := testConnector(t, &fakeBC{respond: func(bcCall) (int, string) {
				return tc.status, readFixture(t, tc.fixture)
			}})
			err := conn.VerifyApp(t.Context())
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}

	conn := testConnector(t, &fakeBC{respond: func(bcCall) (int, string) {
		return http.StatusServiceUnavailable, readFixture(t, "error_503.txt")
	}})
	err := conn.VerifyApp(t.Context())
	require.Error(t, err)
	assert.NotErrorIs(t, err, services.ErrAccountingAppRejected)
}

func TestVerifyWebhookNeverAcceptsABody(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	require.ErrorIs(t, conn.VerifyWebhook("", []byte(readFixture(t, "notifications.json"))),
		errWebhookSignature)
	require.Error(t, conn.VerifyWebhook("anything", nil))
}

func TestProviderParsesNotificationsAndValidationTokens(t *testing.T) {
	t.Parallel()

	provider := newProvider(t, config.BusinessCentralConfig{})
	notes, err := provider.ParseNotifications([]byte(readFixture(t, "notifications.json")))
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, services.AccountingWebhookNotification{
		SubscriptionID: "3a6f0b1c2d4e4f5a8b9c0d1e2f3a4b5c",
		ClientState:    "s3cr3t-state",
		Resource: "api/v2.0/companies(a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d)/" +
			"salesInvoices(99999999-aaaa-4bbb-8ccc-ddddddddddd1)",
		ChangeType: "updated",
	}, notes[0])
	assert.Equal(t, "collection", notes[1].ChangeType)

	_, err = provider.ParseNotifications([]byte("not json"))
	require.Error(t, err)

	token, ok := provider.ValidationToken("abc-123")
	assert.True(t, ok)
	assert.Equal(t, "abc-123", token)
	_, ok = provider.ValidationToken("has space")
	assert.False(t, ok)
}

func companiesResponder(t *testing.T, envStatus int) func(call bcCall) (int, string) {
	return func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodGet, "/environments/v1.2"):
			if envStatus != http.StatusOK {
				return envStatus, readFixture(t, "error_403.json")
			}
			return http.StatusOK, readFixture(t, "environments.json")
		case call.is(http.MethodGet, "/companies") && call.Env == testEnv:
			return http.StatusOK, readFixture(t, "companies.json")
		case call.is(http.MethodGet, "/companies") && call.Env == "uat_sandbox-2":
			return http.StatusOK, `{"value":[
			 {"id":"f0e1d2c3-b4a5-4687-9a8b-7c6d5e4f3a2b","name":"Acme Freight","displayName":""},
			 {"id":"not-a-guid","name":"Broken","displayName":"Broken"}
			]}`
		}
		return 0, ""
	}
}

func TestCompaniesListsEveryEnvironmentWithTheSandboxLabel(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: companiesResponder(t, http.StatusOK)}
	conn := testConnector(t, fake)
	companies, err := conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenFor(strings.ToUpper(testTenant)),
	}, "")
	require.NoError(t, err)

	production := businesscentral.CompanyRef{TenantID: testTenant, Environment: testEnv, CompanyID: testCompany}
	acme := businesscentral.CompanyRef{
		TenantID:    testTenant,
		Environment: testEnv,
		CompanyID:   "f0e1d2c3-b4a5-4687-9a8b-7c6d5e4f3a2b",
	}
	sandbox := acme
	sandbox.Environment = "uat_sandbox-2"
	assert.Equal(t, []services.AccountingCompany{
		{ID: production.String(), Name: "CRONUS USA, Inc. · Production"},
		{ID: acme.String(), Name: "Acme Freight LLC · Production"},
		{ID: sandbox.String(), Name: "Acme Freight · uat_sandbox-2 (Sandbox)"},
	}, companies)
	assert.Zero(t, fake.count(http.MethodGet, "/companies")-2)
}

func TestCompaniesProbesProductionWhenEnvironmentsAreRefused(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: companiesResponder(t, http.StatusForbidden)}
	conn := testConnector(t, fake)
	companies, err := conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenFor(testTenant),
	}, "")
	require.NoError(t, err)
	require.Len(t, companies, 2)
	assert.Equal(t, "CRONUS USA, Inc. · Production", companies[0].Name)

	refused := &fakeBC{respond: func(call bcCall) (int, string) {
		return http.StatusUnauthorized, readFixture(t, "error_401.json")
	}}
	conn = testConnector(t, refused)
	_, err = conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenFor(testTenant),
	}, "")
	require.ErrorIs(t, err, errNoAccess)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))
}

func TestCompaniesFailsWithoutACompanyOrATenant(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/environments/v1.2") {
			return http.StatusOK, readFixture(t, "environments.json")
		}
		if call.Path == "/companies" {
			return http.StatusOK, `{"value":[]}`
		}
		return 0, ""
	}})
	_, err := conn.Companies(t.Context(), &services.AccountingTokenGrant{
		AccessToken: accessTokenFor(testTenant),
	}, "")
	require.ErrorIs(t, err, errNoCompany)

	_, err = conn.Companies(t.Context(), &services.AccountingTokenGrant{AccessToken: "opaque"}, "")
	require.ErrorIs(t, err, businesscentral.ErrMalformedToken)
	_, err = conn.Companies(t.Context(), nil, "")
	require.ErrorIs(t, err, businesscentral.ErrAccessTokenRequired)
}

func TestCompanyFactsReadsTheBooks(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodGet, "/companyInformation"):
			return http.StatusOK, readFixture(t, "company_information.json")
		case call.is(http.MethodGet, "/accountingPeriods"):
			return http.StatusOK, readFixture(t, "accounting_periods.json")
		case call.is(http.MethodGet, "/currencies"):
			return http.StatusOK, readFixture(t, "currencies.json")
		case call.is(http.MethodGet, "/companies"):
			return http.StatusOK, readFixture(t, "companies.json")
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	facts, err := conn.CompanyFacts(t.Context(), testRef().String(), "access-token")
	require.NoError(t, err)

	closed := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC).Unix()
	assert.Equal(t, &accountingsync.CompanyFacts{
		CompanyName:          "CRONUS USA, Inc.",
		LegalName:            "CRONUS USA, Inc.",
		Country:              "US",
		HomeCurrency:         "USD",
		MultiCurrencyEnabled: true,
		BooksClosedThrough:   &closed,
		FiscalYearStartMonth: time.January,
	}, facts)
}

func TestBooksClosedThroughTakesTheLaterOfPostingFromAndClosedPeriods(t *testing.T) {
	t.Parallel()

	periods := []businesscentral.AccountingPeriod{
		{StartingDate: "2026-01-01", NewFiscalYear: true, Closed: true},
		{StartingDate: "2026-02-01", Closed: true},
		{StartingDate: "2026-03-01"},
	}
	setup := &businesscentral.GeneralLedgerSetup{AllowPostingFrom: "2026-05-01"}
	got := booksClosedThrough(setup, periods)
	require.NotNil(t, got)
	assert.Equal(t, time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC).Unix(), *got)

	setup.AllowPostingFrom = ""
	got = booksClosedThrough(setup, periods)
	require.NotNil(t, got)
	assert.Equal(t, time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC).Unix(), *got)

	assert.Nil(t, booksClosedThrough(setup, periods[2:]))
	assert.Equal(t, time.January, fiscalYearStart(periods))
	assert.Equal(t, time.Month(0), fiscalYearStart(nil))
	assert.False(t, hasForeignCurrency([]businesscentral.Currency{{Code: "usd"}}, "USD"))
}

func TestClassifyErrorMapsConnectionFailures(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	cases := []struct {
		name string
		err  error
		want accountingsync.ErrorCategory
	}{
		{"nil", nil, ""},
		{"not configured", ErrNotConfigured, accountingsync.ErrorCategoryConfiguration},
		{"no access", errNoAccess, accountingsync.ErrorCategoryConfiguration},
		{"revoked", &businesscentral.OAuthError{Code: "invalid_grant"}, accountingsync.ErrorCategoryRevoked},
		{"client", &businesscentral.OAuthError{Code: "invalid_client"}, accountingsync.ErrorCategoryUnauthorized},
		{"auth", &businesscentral.APIError{Status: 401}, accountingsync.ErrorCategoryUnauthorized},
		{"forbidden", &businesscentral.APIError{Status: 403}, accountingsync.ErrorCategoryConfiguration},
		{"rate", &businesscentral.APIError{Status: 429}, accountingsync.ErrorCategoryRateLimited},
		{"busy", &businesscentral.APIError{Status: 503}, accountingsync.ErrorCategoryTransient},
		{"deadline", context.DeadlineExceeded, accountingsync.ErrorCategoryTransient},
		{"other", errors.New("boom"), accountingsync.ErrorCategoryUnknown},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, conn.ClassifyError(tc.err), tc.name)
	}
}

func TestCompanyGateBoundsConcurrentCallsPerCompany(t *testing.T) {
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

	client := &http.Client{Transport: newCompanyGate().transport(http.DefaultTransport)}
	target := server.URL + "/v2.0/t/Production/api/v2.0/companies(" + testCompany + ")/items"
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
			if !assert.NoError(t, err) {
				return
			}
			resp, err := client.Do(req)
			if assert.NoError(t, err) {
				_ = resp.Body.Close()
			}
		})
	}
	assert.Eventually(t, func() bool { return current.Load() == maxConcurrentPerCompany },
		time.Second, time.Millisecond)
	close(release)
	wg.Wait()
	assert.Equal(t, int32(maxConcurrentPerCompany), peak.Load())
}

func TestCompanyKeyNamesTheCompanyInThePath(t *testing.T) {
	t.Parallel()

	parsed, err := url.Parse("https://api.example.com/v2.0/t/Production/api/v2.0/companies(ABC)/items(1)")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com/v2.0/t/production/api/v2.0/companies(abc)",
		companyKey(&http.Request{URL: parsed}))
	parsed, err = url.Parse("https://api.example.com/environments/v1.2")
	require.NoError(t, err)
	assert.Empty(t, companyKey(&http.Request{URL: parsed}))
}

func TestBatchLocksSerializeAndHonourCancellation(t *testing.T) {
	t.Parallel()

	locks := newBatchLocks()
	release, err := locks.acquire(t.Context(), "batch")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err = locks.acquire(ctx, "batch")
	require.ErrorIs(t, err, context.DeadlineExceeded)

	other, err := locks.acquire(t.Context(), "other")
	require.NoError(t, err)
	other()

	release()
	again, err := locks.acquire(t.Context(), "batch")
	require.NoError(t, err)
	again()
	assert.Empty(t, locks.byBatch)
}
