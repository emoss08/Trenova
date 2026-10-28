package nsconnector

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changesRequest(cursor string) *accountingconnectorports.ReadAccountingChangesRequest {
	return &accountingconnectorports.ReadAccountingChangesRequest{
		Auth:         testAuth(),
		Cursor:       cursor,
		Now:          testNow,
		Payments:     true,
		BillPayments: true,
		Documents:    true,
		ReferenceKinds: []accountingprovider.ReferenceKind{
			accountingprovider.ReferenceKindCustomer,
			accountingprovider.ReferenceKindTerm,
		},
	}
}

func typeOf(call nsCall) string {
	if len(call.Params) == 0 {
		return ""
	}
	value, _ := call.Params[0].(string)
	return value
}

func changesResponder(t *testing.T) func(call nsCall) reply {
	return func(call nsCall) reply {
		switch {
		case call.queries("FROM transaction t") && typeOf(call) == typeCustomerPayment:
			return fixtureReply(t, http.StatusOK, "changed_payments.json")
		case call.queries("FROM transaction t") && typeOf(call) == typeInvoice:
			return fixtureReply(t, http.StatusOK, "changed_invoices.json")
		case call.queries("FROM customer c"):
			return fixtureReply(t, http.StatusOK, "customers.json")
		case call.queries("FROM deletedrecord dr"):
			return fixtureReply(t, http.StatusOK, "deleted_records.json")
		case call.queries("FROM systemnote sn"):
			return fixtureReply(t, http.StatusOK, "system_notes.json")
		case call.is(http.MethodGet, recordPath(recordCustomerPayment, testPayment)):
			return fixtureReply(t, http.StatusOK, "customer_payment.json")
		}
		return reply{}
	}
}

func TestReadChangesReadsPaymentsBackWithTheirApplications(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: changesResponder(t)}
	conn := testConnector(t, fake)
	start := conn.ChangeCursorAt(testNow.Add(-time.Hour))
	page, err := conn.ReadChanges(t.Context(), changesRequest(start))
	require.NoError(t, err)

	require.Len(t, page.Payments, 1)
	payment := page.Payments[0]
	assert.Equal(t, accountinginbound.InboundCustomerPayment, payment.Kind)
	assert.Equal(t, accountingconnectorports.AccountingChangeUpsert, payment.Operation)
	assert.Equal(t, testPayment, payment.ExternalID)
	assert.Equal(t, "PYMT-88", payment.Number)
	assert.Equal(t, testCustomer, payment.PartyExternalID)
	assert.Equal(t, "Acme Freight", payment.PartyName)
	assert.Equal(t, "2026-10-06", payment.TxnDate)
	assert.Equal(t, "USD", payment.CurrencyCode)
	assert.True(t, payment.Amount.Equal(decimal.NewFromInt(1500)))
	assert.True(t, payment.Unapplied.Equal(decimal.NewFromInt(50)))
	assert.Equal(t, "ACH", payment.MethodName)
	assert.Equal(t, testDeposit, payment.AccountExternalID)
	assert.Equal(t, "ACH-5521", payment.ReferenceNumber)
	assert.Equal(t, "Lee Ortega", payment.ModifiedBy)
	assert.Equal(t, []accountingconnectorports.AccountingInboundLine{
		{
			DocumentKind:       accountinginbound.InboundDocInvoice,
			DocumentExternalID: testInvoice,
			Amount:             decimal.NewFromInt(1250),
		},
		{
			DocumentKind:       accountinginbound.InboundDocInvoice,
			DocumentExternalID: "1202",
			Amount:             decimal.NewFromInt(200),
		},
	}, normalizeLines(payment.Lines))

	read, ok := fake.find(http.MethodGet, recordPath(recordCustomerPayment, testPayment))
	require.True(t, ok)
	assert.Equal(t, "true", read.Query.Get("expandSubResources"))

	require.Len(t, page.Documents, 2)
	assert.Equal(t, testInvoice, page.Documents[0].ExternalID)
	assert.Equal(t, accountingconnectorports.AccountingChangeUpsert, page.Documents[0].Operation)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectDebitMemo,
	}, page.Documents[0].ObjectTypes)
	assert.Equal(t, "Dana Whitfield", page.Documents[0].ModifiedBy)
	assert.Equal(t, accountingconnectorports.AccountingChangeVoid, page.Documents[1].Operation)
	assert.Empty(t, page.Documents[1].ModifiedBy, "no system note names who voided 1205")
	assert.False(t, page.DeletesUnread)
	assert.True(t, page.More, "the invoice target said it has more")
	assert.Empty(t, page.References, "targets after an unfinished one wait for the next read")

	assert.True(t, strings.HasPrefix(page.NextCursor, cursorVersion+cursorSeparator))
	cursor, err := parseChangeCursor(page.NextCursor, testNow)
	require.NoError(t, err)
	invoices := cursor.positions[targetInvoices]
	assert.Equal(t, int64(1205), invoices.id)
	assert.Equal(t, time.Date(2026, 10, 13, 14, 6, 0, 0, time.UTC), invoices.at)
	assert.Equal(t, int64(1401), cursor.positions[targetCustPayments].id)
}

func TestReadChangesAsksSystemNotesForEachTransactionPage(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: changesResponder(t)}
	conn := testConnector(t, fake)
	_, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)

	var notes []nsCall
	for _, call := range fake.all() {
		if call.queries("FROM systemnote sn") {
			notes = append(notes, call)
		}
	}
	require.Len(t, notes, 2)
	assert.Contains(t, notes[0].SQL, "n.recordtypeid = -30")
	assert.Contains(t, notes[0].SQL, "SELECT MAX(n.id) FROM systemnote n")
	assert.Equal(t, []any{float64(1401)}, notes[0].Params)
	assert.Equal(t, []any{float64(1201), float64(1205)}, notes[1].Params)
}

func TestReadChangesLeavesEditorsEmptyWhenSystemNotesAreUnreadable(t *testing.T) {
	t.Parallel()

	responder := changesResponder(t)
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("FROM systemnote sn") {
			return fixtureReply(t, http.StatusForbidden, "error_permission.json")
		}
		return responder(call)
	}}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)
	require.Len(t, page.Payments, 1)
	assert.Empty(t, page.Payments[0].ModifiedBy)
	require.Len(t, page.Documents, 2)
	assert.Empty(t, page.Documents[0].ModifiedBy)

	asked := 0
	for _, call := range fake.all() {
		if call.queries("FROM systemnote sn") {
			asked++
		}
	}
	assert.Equal(t, 1, asked, "a refused system note read is not repeated within a read")
}

func TestReadChangesFailsWhenSystemNotesAreBusy(t *testing.T) {
	t.Parallel()

	responder := changesResponder(t)
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("FROM systemnote sn") {
			return fixtureReply(t, http.StatusTooManyRequests, "error_429_concurrency.json")
		}
		return responder(call)
	}}
	conn := testConnector(t, fake)
	_, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.Error(t, err)
	assert.True(t, netsuite.IsConcurrencyLimit(err))
}

func normalizeLines(
	lines []accountingconnectorports.AccountingInboundLine,
) []accountingconnectorports.AccountingInboundLine {
	out := make([]accountingconnectorports.AccountingInboundLine, 0, len(lines))
	for _, line := range lines {
		line.Amount = decimal.RequireFromString(line.Amount.String())
		out = append(out, line)
	}
	return out
}

func TestReadChangesUsesAKeysetCursorNeverAnOffset(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: changesResponder(t)}
	conn := testConnector(t, fake)
	since := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	cursor := &changeCursor{
		since: since,
		positions: map[string]keyset{
			targetInvoices: {at: time.Date(2026, 10, 13, 14, 4, 5, 0, time.UTC), id: 1201},
		},
	}
	_, err := conn.ReadChanges(t.Context(), changesRequest(cursor.String()))
	require.NoError(t, err)

	var invoiceCall nsCall
	for _, call := range fake.all() {
		if call.queries("FROM transaction t") && typeOf(call) == typeInvoice {
			invoiceCall = call
		}
	}
	require.NotEmpty(t, invoiceCall.SQL)
	key := "TO_CHAR(t.lastmodifieddate, 'YYYY-MM-DD HH24:MI:SS')"
	assert.Contains(t, invoiceCall.SQL, "("+key+" > ? OR ("+key+" = ? AND t.id > ?))")
	assert.True(t, strings.HasSuffix(invoiceCall.SQL, "ORDER BY "+key+", t.id"))
	assert.Contains(t, invoiceCall.SQL, "tl.subsidiary = ?")
	assert.Equal(t, []any{
		typeInvoice,
		"2026-10-13 14:04:05",
		"2026-10-13 14:04:05",
		float64(1201),
		testSubsidiary,
	}, invoiceCall.Params)
	assert.Equal(t, "999", invoiceCall.Query.Get("limit"), "the payment read spent one of the 1000")
	assert.Empty(t, invoiceCall.Query.Get("offset"))

	payments, _ := fake.query("FROM transaction t")
	assert.Equal(t, typeCustomerPayment, typeOf(payments))
	assert.Equal(t, netsuite.FormatQueryTime(since.Add(-timezoneMargin)), payments.Params[1])
	assert.Equal(t, "100", payments.Query.Get("limit"))
}

func TestReadChangesFinishesWithReferencesAndDeletes(t *testing.T) {
	t.Parallel()

	responder := changesResponder(t)
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("FROM transaction t") && typeOf(call) == typeInvoice {
			return reply{status: http.StatusOK, body: emptyRows}
		}
		return responder(call)
	}}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)
	assert.False(t, page.More)
	assert.False(t, page.DeletesUnread)

	require.Len(t, page.References, 3)
	assert.Equal(t, "Acme Freight", page.References[0].Object.Name)
	assert.False(t, page.References[0].Deleted)
	deleted := page.References[2]
	assert.True(t, deleted.Deleted)
	assert.Equal(t, accountingprovider.ReferenceKindCustomer, deleted.Object.Kind)
	assert.Equal(t, "350", deleted.Object.ExternalID)

	require.Len(t, page.Documents, 1)
	assert.Equal(t, accountingconnectorports.AccountingChangeDelete, page.Documents[0].Operation)
	assert.Equal(t, "1209", page.Documents[0].ExternalID)
	require.Len(t, page.Payments, 2)
	assert.Equal(t, accountingconnectorports.AccountingChangeDelete, page.Payments[1].Operation)
	assert.Equal(t, "1410", page.Payments[1].ExternalID)

	_, termsRead := fake.query("FROM term")
	assert.False(t, termsRead, "terms are not a change target")
	cursor, err := parseChangeCursor(page.NextCursor, testNow)
	require.NoError(t, err)
	assert.Equal(t, int64(350), cursor.positions[targetDeleted].id)
	assert.Equal(t, int64(342), cursor.positions[targetCustomers].id)
	_, kept := cursor.positions[targetInvoices]
	assert.True(t, kept)
}

func TestReadChangesSkipsDeletedRecordsWhenTheRoleCannotReadThem(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("FROM deletedrecord dr") {
			return reply{status: http.StatusBadRequest, body: strings.ReplaceAll(
				readFixture(t, "error_unsupported_table.json"), "'subsidiary'", "'deletedrecord'",
			)}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)
	assert.False(t, page.More)
	assert.True(t, page.DeletesUnread)
	assert.True(t, conn.ChangeFeedLimits().ReportsDeletes)
	assert.Equal(t, 1000, conn.ChangeFeedLimits().MaxPerRead)
}

func TestReadChangesWithNothingToReadKeepsTheCursor(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	start := conn.ChangeCursorAt(testNow)
	page, err := conn.ReadChanges(
		t.Context(),
		&accountingconnectorports.ReadAccountingChangesRequest{
			Auth:   testAuth(),
			Cursor: start,
			Now:    testNow,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, start, page.NextCursor)
	assert.Empty(t, fake.all())
	assert.NotNil(t, page.Payments)
}

func TestReadChangesRefusesACursorItDidNotWrite(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	for _, raw := range []string{
		"t:123",
		"n1",
		"n1;abc",
		"n1;100;bogus=1|2",
		"n1;100;invoices=1",
		"n1;100;invoices=x|2",
	} {
		_, err := conn.ReadChanges(t.Context(), changesRequest(raw))
		require.ErrorIs(t, err, errChangeCursor, raw)
	}
}

func TestVendorPaymentsReadBackAsBillPayments(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.queries("FROM transaction t") && typeOf(call) == typeVendorPayment:
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": testBillPayment, "type": "VendPymt", "voided": "T", "currency": "USD",
				"lastmodified": "2026-10-13 14:00:00",
			})}
		case call.is(http.MethodGet, recordPath(recordVendorPayment, testBillPayment)):
			return fixtureReply(t, http.StatusOK, "vendor_payment.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(
		t.Context(),
		&accountingconnectorports.ReadAccountingChangesRequest{
			Auth:         testAuth(),
			Now:          testNow,
			BillPayments: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, page.Payments, 1)
	payment := page.Payments[0]
	assert.Equal(t, accountinginbound.InboundBillPayment, payment.Kind)
	assert.Equal(t, accountingconnectorports.AccountingChangeVoid, payment.Operation)
	assert.Equal(t, testVendor, payment.PartyExternalID)
	assert.True(t, payment.Amount.Equal(decimal.NewFromInt(900)))
	assert.Equal(t, "CHK-1009", payment.ReferenceNumber)
	require.Len(t, payment.Lines, 1)
	assert.Equal(t, accountinginbound.InboundDocBill, payment.Lines[0].DocumentKind)
	assert.Equal(t, testBill, payment.Lines[0].DocumentExternalID)
}

func TestInboundDocumentKindNamesNetSuiteTypes(t *testing.T) {
	t.Parallel()

	customer := accountinginbound.InboundCustomerPayment
	bill := accountinginbound.InboundBillPayment
	assert.Equal(t, accountinginbound.InboundDocInvoice, inboundDocumentKind(customer, "Invoice"))
	assert.Equal(
		t,
		accountinginbound.InboundDocCreditMemo,
		inboundDocumentKind(customer, "Credit Memo"),
	)
	assert.Equal(t, accountinginbound.InboundDocInvoice, inboundDocumentKind(customer, ""))
	assert.Equal(t, accountinginbound.InboundDocBill, inboundDocumentKind(bill, ""))
	assert.Equal(
		t,
		accountinginbound.InboundDocVendorCredit,
		inboundDocumentKind(bill, "Bill Credit"),
	)
	assert.Equal(t, accountinginbound.InboundDocOther, inboundDocumentKind(customer, "Journal"))
}
