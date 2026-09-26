package qboconnector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type changeCall struct {
	Path  string
	Query string
	Since string
	Items string
}

type fakeChangeFeed struct {
	mu      sync.Mutex
	calls   []changeCall
	respond func(call changeCall) string
}

func (f *fakeChangeFeed) connector(t *testing.T) *Connector {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := changeCall{
			Path:  strings.TrimPrefix(r.URL.Path, "/v3/company/123"),
			Query: r.URL.Query().Get("query"),
			Since: r.URL.Query().Get("changedSince"),
			Items: r.URL.Query().Get("entities"),
		}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()
		_, _ = w.Write([]byte(f.respond(call)))
	}))
	t.Cleanup(server.Close)
	conn := newConnector(t, config.QuickBooksConfig{ClientID: "id", ClientSecret: "secret"})
	conn.apiOpts = []quickbooks.Option{
		quickbooks.WithBaseURL(server.URL),
		quickbooks.WithRetry(restx.RetryConfig{}),
	}
	return conn
}

func (f *fakeChangeFeed) recorded() []changeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]changeCall{}, f.calls...)
}

const cdcPaymentAndCustomer = `{"CDCResponse":[{"QueryResponse":[
{"Payment":[{"Id":"301","TxnDate":"2026-09-24","TotalAmt":1500.25,"UnappliedAmt":49.75,
 "CustomerRef":{"value":"58","name":"Acme Foods"},"CurrencyRef":{"value":"USD"},
 "PaymentMethodRef":{"value":"2","name":"Check"},"DepositToAccountRef":{"value":"35"},
 "PaymentRefNum":"10442",
 "Line":[{"Amount":1250.50,"LinkedTxn":[{"TxnId":"145","TxnType":"Invoice"}]},
         {"Amount":75,"LinkedTxn":[{"TxnId":"160","TxnType":"CreditMemo"}]},
         {"Amount":10,"LinkedTxn":[{"TxnId":"9","TxnType":"JournalEntry"}]}],
 "MetaData":{"LastUpdatedTime":"2026-09-24T16:12:30Z","LastModifiedByRef":{"value":"9","name":"J Doe"}}},
 {"Id":"302","TotalAmt":0,"PrivateNote":"Voided","Line":[],"MetaData":{"LastUpdatedTime":"2026-09-24T17:00:00Z"}},
 {"Id":"303","status":"Deleted","MetaData":{"LastUpdatedTime":"2026-09-24T18:00:00Z"}}]},
{"BillPayment":[{"Id":"410","DocNumber":"5521","TxnDate":"2026-09-25","TotalAmt":980,
 "VendorRef":{"value":"77","name":"Swift Haulers"},"PayType":"Check",
 "CheckPayment":{"BankAccountRef":{"value":"35"}},
 "Line":[{"Amount":980,"LinkedTxn":[{"TxnId":"390","TxnType":"Bill"}]}],
 "MetaData":{"LastUpdatedTime":"2026-09-25T08:00:00Z"}}]},
{"Vendor":[{"Id":"81","status":"Deleted","MetaData":{"LastUpdatedTime":"2026-09-24T13:00:00Z"}}]}
]}]}`

func readRequest(cursor string, now time.Time) *services.ReadAccountingChangesRequest {
	return &services.ReadAccountingChangesRequest{
		Auth:         auth(),
		Cursor:       cursor,
		Now:          now,
		Payments:     true,
		BillPayments: true,
		ReferenceKinds: []accountingsync.ReferenceKind{
			accountingsync.ReferenceKindVendor,
		},
	}
}

func TestChangeCursorRoundTrips(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	conn := &Connector{}
	steady, err := parseChangeCursor(conn.ChangeCursorAt(now), now)
	require.NoError(t, err)
	assert.False(t, steady.paging)
	assert.Equal(t, now.Unix(), steady.since.Unix())

	paging := changeCursor{
		since:   now.Add(-40 * 24 * time.Hour),
		paging:  true,
		started: now,
		targets: []quickbooks.ChangeEntity{quickbooks.ChangePayment, "Vendor"},
		index:   1,
		start:   1001,
	}
	parsed, err := parseChangeCursor(paging.String(), now)
	require.NoError(t, err)
	assert.Equal(t, paging.String(), parsed.String())

	for _, bad := range []string{"x:1", "t:abc", "q:1:2:Estimate:0:1", "q:1:2:Payment:3:1", "q:1:2"} {
		_, err = parseChangeCursor(bad, now)
		require.ErrorIs(t, err, errChangeCursor, bad)
	}
}

func TestReadChangesMapsProviderDocumentsToNeutralOnes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	since := now.Add(-time.Hour)
	fake := &fakeChangeFeed{respond: func(changeCall) string { return cdcPaymentAndCustomer }}
	conn := fake.connector(t)

	page, err := conn.ReadChanges(t.Context(), readRequest(conn.ChangeCursorAt(since), now))
	require.NoError(t, err)

	calls := fake.recorded()
	require.Len(t, calls, 1)
	assert.Equal(t, "/cdc", calls[0].Path)
	assert.Equal(t, "Payment,BillPayment,Vendor", calls[0].Items)
	assert.Equal(t, "2026-09-25T15:00:00Z", calls[0].Since)

	assert.False(t, page.More)
	assert.False(t, page.CursorExpired)
	next, err := parseChangeCursor(page.NextCursor, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-changeOverlap).Unix(), next.since.Unix(),
		"the next read overlaps the last by a margin so a late commit is not missed")

	require.Len(t, page.Payments, 4)
	paid := page.Payments[0]
	assert.Equal(t, accountingsync.InboundCustomerPayment, paid.Kind)
	assert.Equal(t, services.AccountingChangeUpsert, paid.Operation)
	assert.Equal(t, "301", paid.ExternalID)
	assert.Equal(t, "10442", paid.Number)
	assert.Equal(t, "58", paid.PartyExternalID)
	assert.Equal(t, "Acme Foods", paid.PartyName)
	assert.Equal(t, "J Doe", paid.ModifiedBy)
	assert.Equal(t, "2026-09-24", paid.TxnDate)
	assert.True(t, decimal.RequireFromString("49.75").Equal(paid.Unapplied))
	assert.Equal(t, "35", paid.AccountExternalID)
	require.Len(t, paid.Lines, 3)
	assert.Equal(t, accountingsync.InboundDocInvoice, paid.Lines[0].DocumentKind)
	assert.Equal(t, accountingsync.InboundDocCreditMemo, paid.Lines[1].DocumentKind)
	assert.Equal(t, accountingsync.InboundDocOther, paid.Lines[2].DocumentKind)

	assert.Equal(t, services.AccountingChangeVoid, page.Payments[1].Operation)
	assert.Equal(t, services.AccountingChangeDelete, page.Payments[2].Operation)

	bill := page.Payments[3]
	assert.Equal(t, accountingsync.InboundBillPayment, bill.Kind)
	assert.Equal(t, "5521", bill.Number)
	assert.Equal(t, "77", bill.PartyExternalID)
	assert.Equal(t, accountingsync.InboundDocBill, bill.Lines[0].DocumentKind)

	require.Len(t, page.References, 1)
	assert.Equal(t, accountingsync.ReferenceKindVendor, page.References[0].Object.Kind)
	assert.Equal(t, "81", page.References[0].Object.ExternalID)
	assert.True(t, page.References[0].Deleted)
}

func TestReadChangesPagesByQueryOnceTheWindowHasPassed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	since := now.Add(-45 * 24 * time.Hour)
	fake := &fakeChangeFeed{respond: func(call changeCall) string {
		if strings.Contains(call.Query, "from Payment") {
			return `{"QueryResponse":{"Payment":[{"Id":"1","TotalAmt":5,"Line":[]}]}}`
		}
		return `{"QueryResponse":{}}`
	}}
	conn := fake.connector(t)

	page, err := conn.ReadChanges(t.Context(), readRequest(conn.ChangeCursorAt(since), now))
	require.NoError(t, err)
	assert.True(t, page.CursorExpired, "the provider cannot report deletes this far back")
	assert.True(t, page.More)
	require.Len(t, page.Payments, 1)

	cursor := page.NextCursor
	reads := 1
	for page.More {
		page, err = conn.ReadChanges(t.Context(), readRequest(cursor, now.Add(time.Minute)))
		require.NoError(t, err)
		assert.False(t, page.CursorExpired)
		cursor = page.NextCursor
		reads++
	}
	assert.Equal(t, 3, reads, "one query per entity: payments, bill payments, vendors")

	calls := fake.recorded()
	require.Len(t, calls, 3)
	for _, call := range calls {
		assert.Equal(t, "/query", call.Path)
		assert.Contains(t, call.Query, "MetaData.LastUpdatedTime >= '2026-08-11T16:00:00Z'")
	}
	assert.Contains(t, calls[0].Query, "from Payment")
	assert.Contains(t, calls[1].Query, "from BillPayment")
	assert.Contains(t, calls[2].Query, "from Vendor")

	final, err := parseChangeCursor(cursor, now)
	require.NoError(t, err)
	assert.False(t, final.paging)
	assert.Equal(t, now.Add(-changeOverlap).Unix(), final.since.Unix(),
		"the steady cursor resumes from when paging began, not when it ended")
}

func TestReadChangesPagesAnEntityThatFilledTheChangeFeed(t *testing.T) {
	t.Parallel()

	var cdc strings.Builder
	cdc.WriteString(`{"CDCResponse":[{"QueryResponse":[{"Payment":[`)
	for idx := range quickbooks.MaxChangesPerCall {
		if idx > 0 {
			cdc.WriteString(",")
		}
		cdc.WriteString(`{"Id":"p","TotalAmt":1,"Line":[]}`)
	}
	cdc.WriteString(`]},{"BillPayment":[{"Id":"410","TotalAmt":1,"Line":[]}]}]}]}`)

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	fake := &fakeChangeFeed{respond: func(call changeCall) string {
		if call.Path == "/cdc" {
			return cdc.String()
		}
		return `{"QueryResponse":{"Payment":[{"Id":"q1","TotalAmt":1,"Line":[]}]}}`
	}}
	conn := fake.connector(t)

	page, err := conn.ReadChanges(
		t.Context(),
		readRequest(conn.ChangeCursorAt(now.Add(-time.Hour)), now),
	)
	require.NoError(t, err)
	assert.True(t, page.More)
	require.Len(t, page.Payments, 1, "the capped payments are re-read by query, the rest kept")
	assert.Equal(t, "410", page.Payments[0].ExternalID)

	page, err = conn.ReadChanges(t.Context(), readRequest(page.NextCursor, now))
	require.NoError(t, err)
	assert.False(t, page.More)
	require.Len(t, page.Payments, 1)
	assert.Equal(t, "q1", page.Payments[0].ExternalID)
	assert.Contains(t, fake.recorded()[1].Query, "from Payment")
}
