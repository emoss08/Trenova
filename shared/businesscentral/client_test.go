package businesscentral_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenant      = "6e1f2a3b-4c5d-4e6f-8a9b-0c1d2e3f4a5b"
	testCompany     = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
	testEnv         = "Production"
	testAccessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.super-secret-access.signature"
	testEnvPath     = "/v2.0/" + testTenant + "/" + testEnv + "/api/v2.0"
	testCompanyPath = testEnvPath + "/companies(" + testCompany + ")"
	testCustomer    = "55555555-6666-4777-8888-999999999991"
	testVendor      = "77777777-8888-4999-8aaa-bbbbbbbbbbb1"
	testInvoice     = "99999999-aaaa-4bbb-8ccc-ddddddddddd1"
	testAccount     = "33333333-4444-4555-8666-777777777771"
	testJournal     = "eeeeeeee-ffff-4000-8111-222222222221"
)

type recordingLimiter struct {
	mu      sync.Mutex
	buckets []restx.Bucket
}

func (l *recordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = append(l.buckets, bucket)
	return nil
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func fastRetry() businesscentral.Option {
	return businesscentral.WithRetry(restx.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
}

func testRef() businesscentral.CompanyRef {
	return businesscentral.CompanyRef{
		TenantID:    testTenant,
		Environment: testEnv,
		CompanyID:   testCompany,
	}
}

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func newAPIClient(
	t *testing.T,
	handler http.HandlerFunc,
	opts ...businesscentral.Option,
) *businesscentral.Client {
	t.Helper()
	server := newServer(t, handler)
	all := append([]businesscentral.Option{
		businesscentral.WithBaseURL(server.URL),
		fastRetry(),
	}, opts...)
	client, err := businesscentral.New(testRef(), testAccessToken, all...)
	require.NoError(t, err)
	return client
}

func serveFixture(t *testing.T, status int, name string) http.HandlerFunc {
	t.Helper()
	body := fixture(t, name)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

func readJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &body))
	return body
}

func assertAPIHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, "Bearer "+testAccessToken, r.Header.Get("Authorization"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
	assert.Equal(t, "en-US", r.Header.Get("Accept-Language"))
}

func assertRead(t *testing.T, r *http.Request, path string) {
	t.Helper()
	assert.Equal(t, http.MethodGet, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assert.NotContains(t, r.URL.EscapedPath(), "%28", "parentheses travel unescaped")
	assertAPIHeaders(t, r)
	assert.Empty(t, r.Header.Get("If-Match"))
}

func assertWrite(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	assert.Equal(t, method, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assert.NotContains(t, r.URL.EscapedPath(), "%28", "parentheses travel unescaped")
	assertAPIHeaders(t, r)
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func TestNewValidatesTokenAndCompany(t *testing.T) {
	t.Parallel()

	_, err := businesscentral.New(testRef(), " ")
	require.ErrorIs(t, err, businesscentral.ErrAccessTokenRequired)

	bad := testRef()
	bad.CompanyID = "not-a-guid"
	_, err = businesscentral.New(bad, testAccessToken)
	require.ErrorIs(t, err, businesscentral.ErrInvalidCompanyRef)

	bad = testRef()
	bad.Environment = "bad env"
	_, err = businesscentral.New(bad, testAccessToken)
	require.ErrorIs(t, err, businesscentral.ErrInvalidEnvironment)

	ref := testRef()
	ref.TenantID = strings.ToUpper(strings.ReplaceAll(testTenant, "-", ""))
	client, err := businesscentral.New(ref, testAccessToken)
	require.NoError(t, err)
	assert.Equal(t, testTenant, client.Ref().TenantID, "ids are normalised to lower-case GUIDs")
}

func TestCompanyInformation(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/companyInformation")
		_, _ = w.Write(fixture(t, "company_information.json"))
	})

	info, err := client.CompanyInformation(t.Context())
	require.NoError(t, err)
	assert.Equal(t, testCompany, info.ID)
	assert.Equal(t, "CRONUS USA, Inc.", info.DisplayName)
	assert.Equal(t, "US", info.CountryRegionCode)
	assert.Equal(t, "USD", info.CurrencyCode)
}

func TestCompanyInformationEmptyIsUnexpected(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	_, err := client.CompanyInformation(t.Context())
	require.ErrorIs(t, err, businesscentral.ErrUnexpectedPayload)
}

func TestGeneralLedgerSetupTreatsTheZeroDateAsUnset(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/generalLedgerSetup")
		_, _ = w.Write(fixture(t, "general_ledger_setup.json"))
	})

	setup, err := client.GeneralLedgerSetup(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "USD", setup.LocalCurrencyCode)
	assert.Equal(t, "2026-01-01", setup.AllowPostingFrom)
	assert.Empty(t, setup.AllowPostingTo)

	allowed, err := setup.AllowsPosting("2026-10-01")
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = setup.AllowsPosting("2025-12-31")
	require.NoError(t, err)
	assert.False(t, allowed)
	_, err = setup.AllowsPosting("10/01/2026")
	require.ErrorIs(t, err, businesscentral.ErrInvalidDate)
}

func TestAccountingPeriods(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/accountingPeriods")
		assert.Equal(t, "1000", r.URL.Query().Get("$top"))
		_, _ = w.Write(fixture(t, "accounting_periods.json"))
	})

	periods, err := client.AccountingPeriods(t.Context())
	require.NoError(t, err)
	require.Len(t, periods, 2)
	assert.Equal(t, businesscentral.AccountingPeriod{
		StartingDate: "2026-01-01", Name: "January", NewFiscalYear: true, Closed: true,
		DateLocked: true,
	}, periods[0])
	assert.False(t, periods[1].Closed)
}

func TestCurrencies(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/currencies")
		_, _ = w.Write(fixture(t, "currencies.json"))
	})

	currencies, err := client.Currencies(t.Context())
	require.NoError(t, err)
	require.Len(t, currencies, 2)
	assert.Equal(t, "CAD", currencies[0].Code)
	assert.Equal(t, "Euro", currencies[1].DisplayName)
}

func TestListingFollowsNextLink(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var base string
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/salesInvoices")
		if calls.Add(1) == 1 {
			assert.Equal(t, "1000", r.URL.Query().Get("$top"))
			page := strings.ReplaceAll(
				string(fixture(t, "sales_invoices_page1.json")),
				"{{base}}",
				base,
			)
			_, _ = w.Write([]byte(page))
			return
		}
		assert.Equal(t, "PS-INV103001", r.URL.Query().Get("$skiptoken"))
		assert.Equal(t, "FIN", r.URL.Query().Get("aid"))
		_, _ = w.Write(fixture(t, "sales_invoices_page2.json"))
	})
	base = server.URL
	client, err := businesscentral.New(testRef(), testAccessToken,
		businesscentral.WithBaseURL(server.URL), fastRetry())
	require.NoError(t, err)

	docs, err := client.Documents(t.Context(), businesscentral.DocumentSalesInvoice, nil)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, "TRN-1001", docs[0].ExternalDocumentNumber)
	assert.Equal(t, "TRN-1002", docs[1].ExternalDocumentNumber)
	assert.Equal(t, int32(2), calls.Load())
}

func TestListingRefusesANextLinkOnAnotherHost(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		page := strings.ReplaceAll(string(fixture(t, "sales_invoices_page1.json")),
			"{{base}}", "https://evil.example")
		_, _ = w.Write([]byte(page))
	})

	_, err := client.Documents(t.Context(), businesscentral.DocumentSalesInvoice, nil)
	require.ErrorIs(t, err, businesscentral.ErrForeignNextLink)
}

func TestListingPagesWithSkipWhenAPageIsFull(t *testing.T) {
	t.Parallel()

	var skips []string
	var mu sync.Mutex
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		skips = append(skips, r.URL.Query().Get("$skip"))
		count := len(skips)
		mu.Unlock()
		assert.Equal(t, "1", r.URL.Query().Get("$top"))
		if count < 3 {
			_, _ = w.Write([]byte(`{"value":[{"code":"C` + string(rune('0'+count)) + `"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"value":[]}`))
	}, businesscentral.WithPageSize(1))

	currencies, err := client.Currencies(t.Context())
	require.NoError(t, err)
	require.Len(t, currencies, 2)
	assert.Equal(t, []string{"", "1", "2"}, skips)
}

func TestLimiterBucketIsPerTenantAndEnvironment(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "currencies.json"))
	}, businesscentral.WithLimiter(limiter, "acct"))

	_, err := client.Currencies(t.Context())
	require.NoError(t, err)
	require.Len(t, limiter.buckets, 1)
	assert.Equal(t, businesscentral.TenantBucket("acct", testTenant, "production"),
		limiter.buckets[0])
	assert.Equal(t, "acct:tenant:"+testTenant+":env:production", limiter.buckets[0].Key)
}

func TestTransportErrorsAreRetriedForReads(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			hijacker, ok := w.(http.Hijacker)
			require.True(t, ok)
			conn, _, err := hijacker.Hijack()
			require.NoError(t, err)
			_ = conn.Close()
			return
		}
		_, _ = w.Write(fixture(t, "currencies.json"))
	})

	currencies, err := client.Currencies(t.Context())
	require.NoError(t, err)
	assert.Len(t, currencies, 2)
	assert.Equal(t, int32(2), calls.Load())
}

func TestTransportErrorIsTransientAndWritesAreNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		hijacker, ok := w.(http.Hijacker)
		require.True(t, ok)
		conn, _, err := hijacker.Hijack()
		require.NoError(t, err)
		_ = conn.Close()
	})

	err := client.PostDocument(t.Context(), businesscentral.DocumentSalesInvoice, testInvoice)
	require.Error(t, err)
	assert.True(t, businesscentral.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load(), "a post is never sent twice")
	assert.NotContains(t, err.Error(), "super-secret-access")
}
