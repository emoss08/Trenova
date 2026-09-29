package xeroconnector

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pollNow = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func TestChangeFeedLimitsAndCursorAt(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	assert.Equal(t, services.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: true,
		MaxPerRead:     xero.MaxPageSize,
	}, conn.ChangeFeedLimits())
	at := time.Date(2026, 9, 1, 8, 30, 15, 900, time.FixedZone("x", 3600))
	assert.Equal(t, "2026-09-01T07:30:15Z", conn.ChangeCursorAt(at))
}

func TestChangeCursorRoundTripsAndRejectsForeignCursors(t *testing.T) {
	t.Parallel()

	paging := &changeCursor{
		since:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		paging:  true,
		started: pollNow,
		latest:  pollNow.Add(-time.Minute),
		targets: []changeTarget{targetPayments, targetContacts},
		index:   1,
		page:    3,
	}
	parsed, err := parseChangeCursor(paging.String(), pollNow)
	require.NoError(t, err)
	assert.Equal(t, paging, parsed)

	empty, err := parseChangeCursor("", pollNow)
	require.NoError(t, err)
	assert.Equal(t, pollNow, empty.since)

	for _, bad := range []string{
		"yesterday",
		"t:1790000000",
		"p|2026-09-01T00:00:00Z||||0|1",
		"p|2026-09-01T00:00:00Z|||journals|0|1",
		"p|2026-09-01T00:00:00Z|||payments|1|1",
		"p|2026-09-01T00:00:00Z|||payments|0|0",
	} {
		_, err = parseChangeCursor(bad, pollNow)
		require.ErrorIs(t, err, errChangeCursor, bad)
	}
}

func TestNextSinceIsTheLatestChangeLessOneSecond(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 9, 29, 11, 15, 30, 500, time.UTC)
	cases := []struct {
		name    string
		latest  time.Time
		started time.Time
		want    time.Time
	}{
		{"nothing seen", time.Time{}, time.Time{}, since},
		{"latest less a second", latest, time.Time{}, time.Date(2026, 9, 29, 11, 15, 29, 0, time.UTC)},
		{"never before since", since, time.Time{}, since},
		{"paging bounded by its start", latest, since.Add(30 * time.Minute), since.Add(30*time.Minute - time.Second)},
		{"start after latest", latest, latest.Add(time.Hour), time.Date(2026, 9, 29, 11, 15, 29, 0, time.UTC)},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, nextSince(since, tc.latest, tc.started), tc.name)
	}
}

func changesRequest(cursor string) *services.ReadAccountingChangesRequest {
	return &services.ReadAccountingChangesRequest{
		Auth:         testAuth(),
		Cursor:       cursor,
		Now:          pollNow,
		Payments:     true,
		BillPayments: true,
		Documents:    true,
		ReferenceKinds: []accountingsync.ReferenceKind{
			accountingsync.ReferenceKindAccount,
			accountingsync.ReferenceKindCustomer,
		},
	}
}

func TestReadChangesMapsPaymentsDocumentsAndReferences(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method != http.MethodGet {
			return 0, ""
		}
		switch call.Path {
		case "/Payments":
			return http.StatusOK, `{"Payments":[
			 {"PaymentID":"` + testPayment + `","Status":"AUTHORISED","PaymentType":"ACCRECPAYMENT","Amount":500,"CurrencyRate":1,"Reference":"ACH 7781","Date":"2026-09-15T00:00:00","UpdatedDateUTC":"2026-09-29T11:00:00","Account":{"AccountID":"` + testBank + `"},"Invoice":{"InvoiceID":"` + testInvoice + `","Type":"ACCREC","Contact":{"ContactID":"` + testCustomer + `"}}},
			 {"PaymentID":"` + testPayment2 + `","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","PaymentType":"ACCRECPAYMENT","Amount":200,"CurrencyRate":0.8,"Date":"2026-09-16T00:00:00","UpdatedDateUTC":"2026-09-29T11:10:00","Invoice":{"InvoiceID":"` + testInvoice2 + `","Contact":{"ContactID":"` + testCustomer + `"}}},
			 {"PaymentID":"b26fd49a-cbae-470a-a8f8-bcbc119e0390","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","PaymentType":"ACCRECPAYMENT","Amount":100,"CurrencyRate":0.8,"Date":"2026-09-16T00:00:00","UpdatedDateUTC":"2026-09-29T11:05:00","Invoice":{"InvoiceID":"` + testInvoice + `"}},
			 {"PaymentID":"` + testOldPayment + `","Status":"DELETED","PaymentType":"ACCPAYPAYMENT","Amount":900,"Date":"2026-09-10T00:00:00","UpdatedDateUTC":"2026-09-29T11:20:00","Invoice":{"InvoiceID":"` + testBill + `","Contact":{"ContactID":"` + testVendor + `"}}},
			 {"PaymentID":"b26fd49a-cbae-470a-a8f8-bcbc119e0391","Status":"AUTHORISED","PaymentType":"ARCREDITPAYMENT","Amount":10,"UpdatedDateUTC":"2026-09-29T11:21:00","CreditNote":{"CreditNoteID":"` + testCreditNote + `"}}
			]}`
		case "/Invoices":
			if call.Query.Get("IDs") != "" {
				return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testInvoice2 + `","CurrencyCode":"EUR"},{"InvoiceID":"` + testInvoice + `","CurrencyCode":"EUR"}]}`
			}
			return http.StatusOK, `{"Invoices":[
			 {"InvoiceID":"` + testInvoice + `","Type":"ACCREC","Status":"VOIDED","UpdatedDateUTC":"2026-09-29T11:30:00"},
			 {"InvoiceID":"` + testBill + `","Type":"ACCPAY","Status":"AUTHORISED","UpdatedDateUTC":"2026-09-29T11:31:00"}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Type":"ACCRECCREDIT","Status":"DELETED","UpdatedDateUTC":"2026-09-29T11:40:30"}]}`
		case "/Accounts":
			return http.StatusOK, `{"Accounts":[{"AccountID":"` + testRevenue + `","Code":"200","Name":"Freight Revenue","Type":"REVENUE","Status":"ACTIVE","UpdatedDateUTC":"2026-09-29T11:00:00"}]}`
		case "/Contacts":
			return http.StatusOK, `{"Contacts":[
			 {"ContactID":"` + testCustomer + `","Name":"Globex","ContactStatus":"ACTIVE","IsCustomer":true,"UpdatedDateUTC":"2026-09-29T11:00:00"},
			 {"ContactID":"` + testVendor + `","Name":"Hauler","ContactStatus":"ACTIVE","IsSupplier":true,"UpdatedDateUTC":"2026-09-29T11:00:00"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	page, err := conn.ReadChanges(t.Context(), changesRequest("2026-09-29T10:00:00Z"))
	require.NoError(t, err)
	assert.False(t, page.More)
	assert.Equal(t, "2026-09-29T11:40:29Z", page.NextCursor)

	for _, call := range fake.all() {
		if call.Query.Get("IDs") == "" && call.Path != "/Accounts" || call.ModifiedSince != "" {
			assert.Equal(t, "Tue, 29 Sep 2026 10:00:00 GMT", call.ModifiedSince, call.Path)
		}
	}
	payments, _ := fake.find(http.MethodGet, "/Payments")
	assert.Equal(t, "AUTHORISED,DELETED", payments.Query.Get("Statuses"))

	require.Len(t, page.Payments, 3)
	single := page.Payments[0]
	assert.Equal(t, accountingsync.InboundCustomerPayment, single.Kind)
	assert.Equal(t, services.AccountingChangeUpsert, single.Operation)
	assert.Equal(t, testPayment, single.ExternalID)
	assert.Equal(t, testCustomer, single.PartyExternalID)
	assert.Equal(t, "2026-09-15", single.TxnDate)
	assert.Equal(t, testBank, single.AccountExternalID)
	assert.Empty(t, single.CurrencyCode)
	assert.True(t, single.Amount.Equal(dec("500")))
	require.Len(t, single.Lines, 1)
	assert.Equal(t, accountingsync.InboundDocInvoice, single.Lines[0].DocumentKind)

	batch := page.Payments[1]
	assert.Equal(t, testBatch, batch.ExternalID)
	assert.True(t, batch.Amount.Equal(dec("300")))
	assert.Len(t, batch.Lines, 2)
	assert.Equal(t, "EUR", batch.CurrencyCode)
	assert.Equal(t, time.Date(2026, 9, 29, 11, 10, 0, 0, time.UTC).Unix(), batch.ModifiedAt)

	deleted := page.Payments[2]
	assert.Equal(t, accountingsync.InboundBillPayment, deleted.Kind)
	assert.Equal(t, services.AccountingChangeDelete, deleted.Operation)
	assert.Empty(t, deleted.Lines)

	require.Len(t, page.Documents, 3)
	assert.Equal(t, services.AccountingChangeVoid, page.Documents[0].Operation)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo,
	}, page.Documents[0].ObjectTypes)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCarrierBill, accountingsync.SyncObjectDriverBill,
	}, page.Documents[1].ObjectTypes)
	assert.Equal(t, services.AccountingChangeDelete, page.Documents[2].Operation)

	require.Len(t, page.References, 2)
	assert.Equal(t, accountingsync.AccountClassIncome, page.References[0].Object.AccountClass)
	assert.Equal(t, accountingsync.ReferenceKindCustomer, page.References[1].Object.Kind)
	assert.Equal(t, testCustomer, page.References[1].Object.ExternalID)
}

func TestReadChangesWithNothingNewKeepsTheCursor(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodGet && call.Path != "/Accounts" {
			return http.StatusOK, `{}`
		}
		if call.Path == "/Accounts" {
			return http.StatusOK, `{"Accounts":[]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(t.Context(), changesRequest("2026-09-29T10:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, "2026-09-29T10:00:00Z", page.NextCursor)
	assert.False(t, page.More)

	fresh, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)
	assert.Equal(t, "2026-09-29T12:00:00Z", fresh.NextCursor)

	none, err := conn.ReadChanges(t.Context(), &services.ReadAccountingChangesRequest{
		Auth: testAuth(), Cursor: "2026-09-29T10:00:00Z", Now: pollNow,
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-09-29T10:00:00Z", none.NextCursor)

	_, err = conn.ReadChanges(t.Context(), changesRequest("t:12"))
	require.ErrorIs(t, err, errChangeCursor)
}

func fullInvoicePage(start int, updated string) string {
	var b strings.Builder
	b.WriteString(`{"pagination":{"page":1,"pageSize":1000,"pageCount":2,"itemCount":1001},"Invoices":[`)
	for idx := range xero.MaxPageSize {
		if idx > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b,
			`{"InvoiceID":"00000000-0000-4000-8000-%012d","Type":"ACCREC","Status":"AUTHORISED","UpdatedDateUTC":"%s"}`,
			start+idx, updated)
	}
	b.WriteString("]}")
	return b.String()
}

func TestReadChangesPagesThroughAFullEndpoint(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method != http.MethodGet {
			return 0, ""
		}
		switch call.Path {
		case "/Invoices":
			if call.Query.Get("page") == "1" {
				return http.StatusOK, fullInvoicePage(0, "2026-09-29T11:00:00")
			}
			return http.StatusOK, `{"pagination":{"page":2,"pageSize":1000,"pageCount":2,"itemCount":1001},"Invoices":[{"InvoiceID":"00000000-0000-4000-8000-999999999999","Type":"ACCREC","Status":"AUTHORISED","UpdatedDateUTC":"2026-09-29T11:59:30"}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Type":"ACCRECCREDIT","Status":"AUTHORISED","UpdatedDateUTC":"2026-09-29T10:30:00"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)
	req := &services.ReadAccountingChangesRequest{
		Auth: testAuth(), Cursor: "2026-09-29T10:00:00Z", Now: pollNow, Documents: true,
	}

	first, err := conn.ReadChanges(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, first.More)
	assert.Len(t, first.Documents, xero.MaxPageSize+1)
	assert.True(t, strings.HasPrefix(first.NextCursor, "p|2026-09-29T10:00:00Z|2026-09-29T12:00:00Z|"))

	req.Cursor = first.NextCursor
	req.Now = pollNow.Add(time.Minute)
	second, err := conn.ReadChanges(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, second.More)
	require.Len(t, second.Documents, 1)
	assert.Equal(t, "2026-09-29T11:59:29Z", second.NextCursor)

	pages := 0
	for _, call := range fake.all() {
		if call.Path == "/Invoices" {
			pages++
			assert.Equal(t, "Tue, 29 Sep 2026 10:00:00 GMT", call.ModifiedSince)
		}
	}
	assert.Equal(t, 2, pages)
}

func TestReadChangesPagingAdvancesAcrossTargets(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	cursor := &changeCursor{
		since:   time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
		paging:  true,
		started: pollNow,
		latest:  time.Date(2026, 9, 29, 11, 0, 0, 0, time.UTC),
		targets: []changeTarget{targetPayments, targetContacts},
		page:    2,
	}
	page, err := conn.ReadChanges(t.Context(), &services.ReadAccountingChangesRequest{
		Auth: testAuth(), Cursor: cursor.String(), Now: pollNow, Payments: true,
	})
	require.NoError(t, err)
	assert.True(t, page.More)
	next, err := parseChangeCursor(page.NextCursor, pollNow)
	require.NoError(t, err)
	assert.Equal(t, 1, next.index)
	assert.Equal(t, 2, next.page)

	done, err := conn.ReadChanges(t.Context(), &services.ReadAccountingChangesRequest{
		Auth: testAuth(), Cursor: page.NextCursor, Now: pollNow, Payments: true,
	})
	require.NoError(t, err)
	assert.False(t, done.More)
	assert.Equal(t, "2026-09-29T10:59:59Z", done.NextCursor)
}
