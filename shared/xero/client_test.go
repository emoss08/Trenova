package xero_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenant      = "4c7e1f20-3b5a-4d8e-9f10-2a3b4c5d6e7f"
	testAccessToken = "eyJhbGciOiJSUzI1NiJ9.super-secret-access.signature"
	testContact     = "bd2270c3-8706-4c11-9cfb-000b551c3f51"
	testInvoice     = "fee88eea-f2aa-4a71-a372-33d6d83d3c45"
	testBill        = "0032c6e3-7b1e-4b0f-9c7e-2f6d2b5b8a11"
	testCreditNote  = "7f8b6e4a-2d3c-4b1a-9e8f-6a5b4c3d2e1f"
	testAllocation  = "b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0a"
	testPayment     = "b26fd49a-cbae-470a-a8f8-bcbc119e0379"
	testAccount     = "562555f2-8cde-4ce9-8203-0363922537a4"
	testBatch       = "d318c343-208e-49fe-b04a-45642349bcf1"
	testKey         = "trenova-sync-7c1f0b2e"
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

func fastRetry() xero.Option {
	return xero.WithRetry(restx.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
}

func newAPIClient(t *testing.T, handler http.HandlerFunc, opts ...xero.Option) *xero.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	all := append([]xero.Option{xero.WithBaseURL(server.URL), fastRetry()}, opts...)
	client, err := xero.New(testTenant, testAccessToken, all...)
	require.NoError(t, err)
	return client
}

func readJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &body))
	return body
}

func firstOf(t *testing.T, body map[string]any, key string) map[string]any {
	t.Helper()
	list, ok := body[key].([]any)
	require.True(t, ok, "%s is a list", key)
	require.Len(t, list, 1)
	item, ok := list[0].(map[string]any)
	require.True(t, ok)
	return item
}

func assertAPIHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, "Bearer "+testAccessToken, r.Header.Get("Authorization"))
	assert.Equal(t, testTenant, r.Header.Get("xero-tenant-id"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
}

func assertRead(t *testing.T, r *http.Request, path string) {
	t.Helper()
	assert.Equal(t, http.MethodGet, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assertAPIHeaders(t, r)
	assert.Empty(t, r.Header.Get("Idempotency-Key"), "reads carry no idempotency key")
}

func assertWrite(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	assert.Equal(t, method, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assertAPIHeaders(t, r)
	assert.Equal(t, testKey, r.Header.Get("Idempotency-Key"))
	assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func TestNewRequiresTenantAndToken(t *testing.T) {
	t.Parallel()

	_, err := xero.New(" ", testAccessToken)
	require.ErrorIs(t, err, xero.ErrTenantIDRequired)
	_, err = xero.New(testTenant, "")
	require.ErrorIs(t, err, xero.ErrAccessTokenRequired)

	client, err := xero.New(" "+testTenant+" ", testAccessToken)
	require.NoError(t, err)
	assert.Equal(t, testTenant, client.TenantID())
}

func TestOrganisationReadsSettingsAndLockDates(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Organisation")
		_, _ = w.Write(fixture(t, "organisation.json"))
	})

	org, err := client.Organisation(t.Context())
	require.NoError(t, err)
	assert.Equal(t, testTenant, org.OrganisationID)
	assert.Equal(t, "Acme Freight Ltd", org.Name)
	assert.Equal(t, "Acme Freight Limited", org.LegalName)
	assert.Equal(t, "!mJ4Tp", org.ShortCode)
	assert.Equal(t, "US", org.CountryCode)
	assert.Equal(t, "USD", org.BaseCurrency, "the currency is normalised to upper case")
	assert.Equal(t, "COMPANY", org.OrganisationType)
	assert.Equal(t, "ACCRUALS", org.SalesTaxBasis)
	assert.Equal(t, "BUSINESS", org.Edition)
	assert.Equal(t, 31, org.FinancialYearEndDay)
	assert.Equal(t, time.December, org.FinancialYearEndMonth)
	require.NotNil(t, org.PeriodLockDate)
	assert.Equal(t, time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), *org.PeriodLockDate)
	assert.Nil(t, org.EndOfYearLockDate, "an absent lock date stays nil")
	assert.False(t, org.IsDemoCompany)
}

func TestOrganisationActions(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Organisation/Actions")
		_, _ = w.Write(fixture(t, "organisation_actions.json"))
	})

	actions, err := client.OrganisationActions(t.Context())
	require.NoError(t, err)
	require.Len(t, actions, 3)
	assert.Equal(t, "UseMulticurrency", actions[0].Name)
	assert.True(t, actions[0].Allowed())
	assert.Equal(t, "CreateApprovedBill", actions[2].Name)
	assert.False(t, actions[2].Allowed())
}

func TestAccountsSendIfModifiedSinceAndClassifyControlAccounts(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 1, 8, 30, 0, 0, time.FixedZone("EST", -5*3600))
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Accounts")
		assert.Equal(t, "Tue, 01 Sep 2026 13:30:00 GMT", r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "accounts.json"))
	})

	accounts, err := client.Accounts(t.Context(), &since)
	require.NoError(t, err)
	require.Len(t, accounts, 4)

	bank := accounts[0]
	assert.Equal(t, testAccount, bank.AccountID)
	assert.Equal(t, "090", bank.Code)
	assert.Equal(t, "BANK", bank.Type)
	assert.Equal(t, "ASSET", bank.Class)
	assert.Equal(t, "BANK", bank.BankAccountType)
	assert.Equal(t, "USD", bank.CurrencyCode)
	assert.True(t, bank.Active())
	assert.Equal(t, time.Date(2019, 11, 14, 18, 10, 38, 314_000_000, time.UTC), bank.UpdatedAt)

	assert.True(t, accounts[1].IsReceivable())
	assert.False(t, accounts[1].IsPayable())
	assert.True(t, accounts[2].IsPayable())
	assert.False(t, accounts[3].Active())
	assert.True(t, accounts[3].EnablePaymentsToAccount)
}

func TestAccountsWithoutModifiedSinceSendNoHeader(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "accounts.json"))
	})

	accounts, err := client.Accounts(t.Context(), nil)
	require.NoError(t, err)
	assert.Len(t, accounts, 4)
}

func TestNotModifiedIsAnEmptyResult(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	})

	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	items, err := client.Items(t.Context(), &since)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestItemsReadSalesDetails(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Items")
		_, _ = w.Write(fixture(t, "items.json"))
	})

	items, err := client.Items(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "LINEHAUL", items[0].Code)
	assert.Equal(t, "Linehaul", items[0].Name)
	assert.True(t, items[0].IsSold)
	assert.False(t, items[0].IsPurchased)
	assert.True(t, dec("1500").Equal(items[0].SalesUnitPrice))
	assert.Equal(t, "200", items[0].SalesAccountCode)
	assert.True(t, items[1].SalesUnitPrice.IsZero(), "a missing unit price reads as zero")
}

func TestCreateItemPutsASoldItem(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/Items")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_item.json"))
	})

	item, err := client.CreateItem(t.Context(), testKey, xero.ItemInput{
		Code:             " DETENTION ",
		Name:             "Detention",
		Description:      "Detention time",
		SalesAccountCode: "200",
	})
	require.NoError(t, err)
	sent := firstOf(t, body, "Items")
	assert.Equal(t, "DETENTION", sent["Code"])
	assert.Equal(t, "Detention", sent["Name"])
	assert.Equal(t, "Detention time", sent["Description"])
	assert.Equal(t, true, sent["IsSold"])
	assert.Equal(t, map[string]any{"AccountCode": "200"}, sent["SalesDetails"])
	assert.Equal(t, "1f6a2c3b-4d5e-4a6b-8c7d-9e0f1a2b3c4d", item.ItemID)
	assert.Equal(t, "DETENTION", item.Code)
}

func TestCreateItemValidatesBeforeCallingXero(t *testing.T) {
	t.Parallel()

	calls := 0
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls++ })

	_, err := client.CreateItem(t.Context(), testKey, xero.ItemInput{Name: "Detention"})
	require.ErrorIs(t, err, xero.ErrItemCodeInvalid)
	_, err = client.CreateItem(t.Context(), testKey, xero.ItemInput{
		Code: "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
		Name: "Too long a code",
	})
	require.ErrorIs(t, err, xero.ErrItemCodeInvalid)
	_, err = client.CreateItem(t.Context(), testKey, xero.ItemInput{
		Code: "DET",
		Name: "A name that runs well past the fifty characters Xero allows",
	})
	require.ErrorIs(t, err, xero.ErrItemNameInvalid)
	_, err = client.CreateItem(t.Context(), "", xero.ItemInput{Code: "DET", Name: "Detention"})
	require.ErrorIs(t, err, xero.ErrIdempotencyKey)
	long := make([]byte, xero.MaxIdempotencyKeyLength+1)
	for idx := range long {
		long[idx] = 'k'
	}
	_, err = client.CreateItem(t.Context(), string(long), xero.ItemInput{Code: "DET", Name: "Detention"})
	require.ErrorIs(t, err, xero.ErrIdempotencyKey)
	_, err = client.CreateItem(t.Context(), "key\r\nX-Evil: 1", xero.ItemInput{Code: "DET", Name: "Detention"})
	require.ErrorIs(t, err, xero.ErrIdempotencyKey)
	assert.Zero(t, calls, "nothing reaches Xero")
}

func TestContactsPageThroughEverythingIncludingArchived(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Contacts")
		query := r.URL.Query()
		assert.Equal(t, "1", query.Get("page"))
		assert.Equal(t, "1000", query.Get("pageSize"))
		assert.Equal(t, "true", query.Get("includeArchived"))
		assert.Equal(t, "Sun, 20 Sep 2026 00:00:00 GMT", r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "contacts.json"))
	})

	page, err := client.Contacts(t.Context(), 1, &since)
	require.NoError(t, err)
	assert.True(t, page.More, "page 1 of 2")
	require.Len(t, page.Contacts, 2)

	globex := page.Contacts[0]
	assert.Equal(t, testContact, globex.ContactID)
	assert.Equal(t, "CUST-1001", globex.ContactNumber)
	assert.Equal(t, "Globex Shipping", globex.Name)
	assert.Equal(t, "Hank", globex.FirstName)
	assert.Equal(t, "Scorpio", globex.LastName)
	assert.Equal(t, "ap@globex.example", globex.EmailAddress)
	assert.Equal(t, "12-3456789", globex.TaxNumber)
	assert.True(t, globex.IsCustomer)
	assert.False(t, globex.IsSupplier)
	require.Len(t, globex.Addresses, 2)
	assert.Equal(t, "POBOX", globex.Addresses[0].AddressType)
	assert.Equal(t, "Cypress Creek", globex.Addresses[0].City)
	assert.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), globex.UpdatedAt)

	assert.Equal(t, "ARCHIVED", page.Contacts[1].ContactStatus)
	assert.True(t, page.Contacts[1].IsSupplier)
}

func TestContactsPaginationKeyIsReadInEitherCase(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Contacts":[{"ContactID":"` + testContact +
			`","Name":"Globex"}],"Pagination":{"page":2,"pageSize":1000,"pageCount":2,"itemCount":1001}}`))
	})

	page, err := client.Contacts(t.Context(), 2, nil)
	require.NoError(t, err)
	assert.False(t, page.More, "the last page")
	require.Len(t, page.Contacts, 1)
}

func TestContactsWithoutPaginationUseAFullPageAsMore(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Contacts":[{"ContactID":"` + testContact + `","Name":"Globex"}]}`))
	})

	page, err := client.Contacts(t.Context(), 1, nil)
	require.NoError(t, err)
	assert.False(t, page.More)

	_, err = client.Contacts(t.Context(), 0, nil)
	require.ErrorIs(t, err, xero.ErrInvalidPage)
}

func TestCreateContactPutsTheContact(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/Contacts")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_contact.json"))
	})

	contact, err := client.CreateContact(t.Context(), testKey, xero.ContactInput{
		Name:          " Globex Shipping ",
		ContactNumber: "CUST-1001",
		EmailAddress:  "ap@globex.example",
		TaxNumber:     "12-3456789",
		Addresses: []xero.Address{{
			AddressType:  "pobox",
			AddressLine1: "PO Box 42",
			City:         "Cypress Creek",
			Region:       "OR",
			PostalCode:   "97000",
			Country:      "USA",
		}},
	})
	require.NoError(t, err)
	sent := firstOf(t, body, "Contacts")
	assert.NotContains(t, sent, "ContactID")
	assert.Equal(t, "Globex Shipping", sent["Name"])
	assert.Equal(t, "CUST-1001", sent["ContactNumber"])
	assert.Equal(t, "ap@globex.example", sent["EmailAddress"])
	assert.Equal(t, "12-3456789", sent["TaxNumber"])
	addresses, ok := sent["Addresses"].([]any)
	require.True(t, ok)
	require.Len(t, addresses, 1)
	address, ok := addresses[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "POBOX", address["AddressType"])
	assert.Equal(t, "Cypress Creek", address["City"])
	assert.NotContains(t, address, "AddressLine2", "blank address parts are left out")
	assert.Equal(t, testContact, contact.ContactID)
}

func TestUpdateContactPostsToTheContact(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/Contacts/"+testContact)
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_contact.json"))
	})

	contact, err := client.UpdateContact(t.Context(), testKey, testContact, xero.ContactInput{
		Name: "Globex Shipping",
	})
	require.NoError(t, err)
	sent := firstOf(t, body, "Contacts")
	assert.Equal(t, testContact, sent["ContactID"])
	assert.NotContains(t, sent, "Addresses")
	assert.Equal(t, "Globex Shipping", contact.Name)

	_, err = client.UpdateContact(t.Context(), testKey, "not-a-guid", xero.ContactInput{Name: "X"})
	require.ErrorIs(t, err, xero.ErrInvalidID)
	_, err = client.CreateContact(t.Context(), testKey, xero.ContactInput{Name: " "})
	require.ErrorIs(t, err, xero.ErrContactNameInvalid)
}

func TestDuplicateContactNameIsADuplicate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_duplicate_contact.json"))
	})

	_, err := client.CreateContact(t.Context(), testKey, xero.ContactInput{Name: "Globex Shipping"})
	require.Error(t, err)
	assert.True(t, xero.IsValidation(err))
	assert.True(t, xero.IsDuplicateNumber(err))
	assert.False(t, xero.IsLockDate(err))
}

func TestLimiterBucketIsPerTenant(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "organisation.json"))
	}, xero.WithLimiter(limiter, "xr"))

	_, err := client.Organisation(t.Context())
	require.NoError(t, err)
	require.Len(t, limiter.buckets, 1)
	assert.Equal(t, "xr:tenant:"+testTenant, limiter.buckets[0].Key)
	assert.Equal(t, 50, limiter.buckets[0].Limit)
	assert.Equal(t, time.Minute, limiter.buckets[0].Period)
}

func TestCustomHTTPClientStillCarriesIdempotencyKey(t *testing.T) {
	t.Parallel()

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		_, _ = w.Write(fixture(t, "create_item.json"))
	}))
	t.Cleanup(server.Close)

	client, err := xero.New(testTenant, testAccessToken,
		xero.WithBaseURL(server.URL),
		xero.WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
		xero.WithUserAgent("trenova-test/1"),
		xero.WithTimeout(5*time.Second),
	)
	require.NoError(t, err)
	_, err = client.CreateItem(t.Context(), testKey, xero.ItemInput{Code: "DET", Name: "Detention"})
	require.NoError(t, err)
	assert.Equal(t, testKey, got)
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	parsed, err := xero.ParseDate("/Date(1573755038314+0000)/")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2019, 11, 14, 18, 10, 38, 314_000_000, time.UTC), parsed)

	parsed, err = xero.ParseDate("/Date(1573755038314-0500)/")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2019, 11, 14, 18, 10, 38, 314_000_000, time.UTC), parsed,
		"the offset names the zone, the milliseconds are already UTC")

	parsed, err = xero.ParseDate("/Date(1790208000000)/")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), parsed)

	parsed, err = xero.ParseDate("2019-11-14T00:00:00")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2019, 11, 14, 0, 0, 0, 0, time.UTC), parsed)

	parsed, err = xero.ParseDate("2026-09-20T23:40:30.1833130")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 20, 23, 40, 30, 183_313_000, time.UTC), parsed)

	parsed, err = xero.ParseDate("2026-09-20T23:40:30Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 20, 23, 40, 30, 0, time.UTC), parsed)

	parsed, err = xero.ParseDate(" 2026-01-02 ")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), parsed)

	for _, bad := range []string{"", "yesterday", "/Date(abc)/", "/Date(123+00)/", "/Date()/"} {
		_, err = xero.ParseDate(bad)
		require.ErrorIs(t, err, xero.ErrInvalidDate, bad)
	}
}

func TestUnreadableDateFailsTheDecode(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Accounts":[{"AccountID":"a","UpdatedDateUTC":"not a date"}]}`))
	})

	_, err := client.Accounts(t.Context(), nil)
	require.Error(t, err)
}
