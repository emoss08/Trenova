package qboconnector

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func queryIDsOf(statement string) []string {
	start := strings.Index(statement, "('")
	end := strings.Index(statement, "')")
	if start < 0 || end < 0 {
		return nil
	}
	return strings.Split(statement[start+2:end], "', '")
}

func TestReadDocumentsReadsBillsAndVendorCreditsApartAndMarksWhatIsGone(t *testing.T) {
	t.Parallel()

	var statements []string
	fake := &fakeChangeFeed{respond: func(call changeCall) string {
		statements = append(statements, call.Query)
		switch {
		case strings.HasPrefix(call.Query, "select * from Bill "):
			return `{"QueryResponse":{"Bill":[
{"Id":"401","DocNumber":"CS-0042","TotalAmt":1400,"Balance":1400,"CurrencyRef":{"value":"USD"},
 "MetaData":{"LastUpdatedTime":"2026-09-24T16:12:30Z","LastModifiedByRef":{"value":"9","name":"J Doe"}}}]}}`
		case strings.HasPrefix(call.Query, "select * from VendorCredit "):
			return `{"QueryResponse":{"VendorCredit":[
{"Id":"402","DocNumber":"CS-0043","TotalAmt":0,"PrivateNote":"Voided","MetaData":{"LastUpdatedTime":"2026-09-24T17:00:00Z"}}]}}`
		default:
			t.Errorf("unexpected query %q", call.Query)
			return `{}`
		}
	}}
	conn := fake.connector(t)

	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: auth(),
		Kind: accountingsync.SyncObjectCarrierBill,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: "401", Refs: map[string]string{accountingsync.ExternalRefDocumentType: "Bill"}},
			{ExternalID: "402", Refs: map[string]string{accountingsync.ExternalRefDocumentType: "VendorCredit"}},
			{ExternalID: "403"},
		},
	})
	require.NoError(t, err)
	require.Len(t, statements, 2, "one read per provider entity")

	require.Len(t, states, 3, "one state per target, in order")
	bill := states[0]
	assert.Equal(t, "401", bill.ExternalID)
	assert.True(t, bill.Found)
	assert.False(t, bill.Voided)
	assert.True(t, decimal.NewFromInt(1400).Equal(bill.Total))
	require.NotNil(t, bill.Balance)
	assert.Equal(t, "J Doe", bill.ModifiedBy)
	assert.Equal(t, time.Date(2026, 9, 24, 16, 12, 30, 0, time.UTC).Unix(), bill.ModifiedAt)
	assert.Equal(t, "USD", bill.CurrencyCode)

	assert.True(t, states[1].Found)
	assert.True(t, states[1].Voided)

	assert.Equal(t, "403", states[2].ExternalID)
	assert.False(t, states[2].Found, "a document the provider no longer returns was deleted there")
}

func TestReadDocumentsMapsEachTrenovaKindToItsProviderEntity(t *testing.T) {
	t.Parallel()

	cases := map[accountingsync.SyncObjectType]string{
		accountingsync.SyncObjectInvoice:           "Invoice",
		accountingsync.SyncObjectDebitMemo:         "Invoice",
		accountingsync.SyncObjectCreditMemo:        "CreditMemo",
		accountingsync.SyncObjectCustomerPayment:   "Payment",
		accountingsync.SyncObjectCreditApplication: "Payment",
		accountingsync.SyncObjectDriverBill:        "Bill",
		accountingsync.SyncObjectCarrierBillPay:    "BillPayment",
		accountingsync.SyncObjectDriverBillPay:     "BillPayment",
	}
	for kind, entity := range cases {
		var statement string
		fake := &fakeChangeFeed{respond: func(call changeCall) string {
			statement = call.Query
			return `{"QueryResponse":{}}`
		}}
		conn := fake.connector(t)
		states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
			Auth:    auth(),
			Kind:    kind,
			Targets: []services.AccountingDocumentTarget{{ExternalID: "7"}},
		})
		require.NoError(t, err, kind)
		assert.Equal(t, "select * from "+entity+" where Id in ('7') maxresults 1", statement, kind)
		require.Len(t, states, 1)
		assert.False(t, states[0].Found)
	}

	_, err := (&fakeChangeFeed{respond: func(changeCall) string { return `{}` }}).connector(t).
		ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
			Auth:    auth(),
			Kind:    accountingsync.SyncObjectCustomer,
			Targets: []services.AccountingDocumentTarget{{ExternalID: "7"}},
		})
	require.ErrorIs(t, err, errDocumentKind)
}

func TestReadDocumentsSplitsLargeReads(t *testing.T) {
	t.Parallel()

	var sizes []int
	fake := &fakeChangeFeed{respond: func(call changeCall) string {
		ids := queryIDsOf(call.Query)
		sizes = append(sizes, len(ids))
		rows := make([]string, 0, len(ids))
		for _, id := range ids {
			rows = append(rows, fmt.Sprintf(`{"Id":%q,"TotalAmt":1}`, id))
		}
		return `{"QueryResponse":{"Invoice":[` + strings.Join(rows, ",") + `]}}`
	}}
	conn := fake.connector(t)
	targets := make([]services.AccountingDocumentTarget, 0, quickbooks.MaxQueryIDs+5)
	for i := range quickbooks.MaxQueryIDs + 5 {
		targets = append(targets, services.AccountingDocumentTarget{ExternalID: strconv.Itoa(i + 1)})
	}

	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth:    auth(),
		Kind:    accountingsync.SyncObjectInvoice,
		Targets: targets,
	})
	require.NoError(t, err)
	assert.Equal(t, []int{quickbooks.MaxQueryIDs, 5}, sizes)
	require.Len(t, states, len(targets))
	for _, state := range states {
		assert.True(t, state.Found)
	}
	assert.Equal(t, quickbooks.MaxQueryIDs, conn.DocumentReadLimits().MaxPerRead)
}

func readFixture(kind string, id string) string {
	return `{"` + kind + `":{"Id":"` + id + `","SyncToken":"3","DocNumber":"D-1","TotalAmt":10}}`
}

func TestUpdatesSendTrenovasDocumentOverTheProvidersOne(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		switch {
		case call.Method == http.MethodGet && call.Path == "/invoice/145":
			return http.StatusOK, readFixture("Invoice", "145")
		case call.Method == http.MethodGet && call.Path == "/creditmemo/160":
			return http.StatusOK, readFixture("CreditMemo", "160")
		case call.Method == http.MethodGet && call.Path == "/vendorcredit/402":
			return http.StatusOK, readFixture("VendorCredit", "402")
		case call.Method == http.MethodGet && call.Path == "/billpayment/501":
			return http.StatusOK, readFixture("BillPayment", "501")
		case call.Path == "/invoice":
			return http.StatusOK, readFixture("Invoice", "145")
		case call.Path == "/creditmemo":
			return http.StatusOK, readFixture("CreditMemo", "160")
		case call.Path == "/vendorcredit":
			return http.StatusOK, readFixture("VendorCredit", "402")
		case call.Path == "/billpayment":
			return http.StatusOK, readFixture("BillPayment", "501")
		default:
			t.Errorf("unexpected call %s %s", call.Method, call.Path)
			return http.StatusNotFound, `{}`
		}
	}}
	conn := documentConnector(t, fake)

	sales := func(kind accountingsync.SyncObjectType, id string) *services.AccountingSalesDocument {
		return &services.AccountingSalesDocument{
			Auth:               auth(),
			RequestID:          docRequestID,
			ExternalID:         id,
			Kind:               kind,
			CustomerExternalID: "58",
			DocNumber:          "INV-1001",
			PrivateNote:        "Trenova INV-1001",
			Lines: []services.AccountingDocumentLine{
				{ItemExternalID: "21", Amount: decimal.NewFromInt(1500), Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1500)},
			},
			Refs: map[string]string{accountingsync.ExternalRefDocument: id},
		}
	}

	debit, err := conn.UpdateSalesDocument(t.Context(), sales(accountingsync.SyncObjectDebitMemo, "145"))
	require.NoError(t, err)
	assert.Equal(t, "145", debit.ExternalID)
	assert.Equal(t, "145", debit.Refs[accountingsync.ExternalRefDocument])
	_, err = conn.UpdateSalesDocument(t.Context(), sales(accountingsync.SyncObjectCreditMemo, "160"))
	require.NoError(t, err)

	credit, err := conn.UpdatePurchaseDocument(t.Context(), &services.AccountingPurchaseDocument{
		Auth:             auth(),
		RequestID:        docRequestID,
		ExternalID:       "402",
		Kind:             accountingsync.SyncObjectCarrierBill,
		VendorCredit:     true,
		VendorExternalID: "77",
		Lines:            []services.AccountingPurchaseLine{{AccountExternalID: "60", Amount: decimal.NewFromInt(40)}},
	})
	require.NoError(t, err)
	assert.Equal(t, "VendorCredit", credit.Refs[accountingsync.ExternalRefDocumentType])
	assert.Contains(t, credit.Refs[accountingsync.ExternalRefURL], "402")

	_, err = conn.UpdateBillPayment(t.Context(), &services.AccountingBillPaymentDocument{
		Auth:                  auth(),
		RequestID:             docRequestID,
		ExternalID:            "501",
		Kind:                  accountingsync.SyncObjectCarrierBillPay,
		VendorExternalID:      "77",
		BankAccountExternalID: "35",
		BillExternalID:        "401",
		Amount:                decimal.NewFromInt(1500),
	})
	require.NoError(t, err)

	writes := fake.writes()
	require.Len(t, writes, 4)
	assert.Equal(t, "/invoice", writes[0].Path)
	assert.Equal(t, "145", writes[0].Body["Id"])
	assert.Equal(t, "3", writes[0].Body["SyncToken"])
	assert.True(t, strings.HasPrefix(writes[0].Body["PrivateNote"].(string), debitMemoNotePrefix),
		"a debit memo stays marked as one")
	assert.Equal(t, "/creditmemo", writes[1].Path)
	assert.Equal(t, "/vendorcredit", writes[2].Path)
	assert.Equal(t, "402", writes[2].Body["Id"])
	assert.Equal(t, "/billpayment", writes[3].Path)
	assert.Equal(t, "501", writes[3].Body["Id"])
}

func TestUpdatesNeedTheProvidersDocumentAndTheRightKind(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) { return http.StatusOK, `{}` }}
	conn := documentConnector(t, fake)

	_, err := conn.UpdateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Auth: auth(), RequestID: docRequestID, Kind: accountingsync.SyncObjectInvoice,
	})
	require.ErrorIs(t, err, errExternalIDRequired)
	_, err = conn.UpdateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Auth: auth(), RequestID: docRequestID, ExternalID: "1", Kind: accountingsync.SyncObjectCarrierBill,
	})
	require.ErrorIs(t, err, errDocumentKind)
	_, err = conn.UpdatePurchaseDocument(t.Context(), &services.AccountingPurchaseDocument{
		Auth: auth(), RequestID: docRequestID, ExternalID: "1", Kind: accountingsync.SyncObjectInvoice,
	})
	require.ErrorIs(t, err, errDocumentKind)
	_, err = conn.UpdateBillPayment(t.Context(), &services.AccountingBillPaymentDocument{
		Auth: auth(), RequestID: docRequestID, Kind: accountingsync.SyncObjectCarrierBillPay,
	})
	require.ErrorIs(t, err, errExternalIDRequired)
	assert.Empty(t, fake.writes())
}

func TestReadChangesReportsChangedDocumentsWithTheirTrenovaKinds(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	fake := &fakeChangeFeed{respond: func(changeCall) string {
		return `{"CDCResponse":[{"QueryResponse":[
{"Invoice":[{"Id":"145","TotalAmt":1400,"MetaData":{"LastUpdatedTime":"2026-09-24T16:12:30Z","LastModifiedByRef":{"value":"9","name":"J Doe"}}}]},
{"CreditMemo":[{"Id":"160","status":"Deleted","MetaData":{"LastUpdatedTime":"2026-09-24T17:00:00Z"}}]},
{"VendorCredit":[{"Id":"402","TotalAmt":0,"PrivateNote":"Voided","MetaData":{"LastUpdatedTime":"2026-09-24T18:00:00Z"}}]}
]}]}`
	}}
	conn := fake.connector(t)

	page, err := conn.ReadChanges(t.Context(), &services.ReadAccountingChangesRequest{
		Auth:      auth(),
		Cursor:    conn.ChangeCursorAt(now.Add(-time.Hour)),
		Now:       now,
		Documents: true,
	})
	require.NoError(t, err)
	calls := fake.recorded()
	require.Len(t, calls, 1)
	assert.Equal(t, "Invoice,CreditMemo,Bill,VendorCredit", calls[0].Items)

	require.Len(t, page.Documents, 3)
	invoice := page.Documents[0]
	assert.Equal(t, "145", invoice.ExternalID)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectDebitMemo,
	}, invoice.ObjectTypes)
	assert.Equal(t, services.AccountingChangeUpsert, invoice.Operation)
	assert.Equal(t, "J Doe", invoice.ModifiedBy)

	assert.Equal(t, []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}, page.Documents[1].ObjectTypes)
	assert.Equal(t, services.AccountingChangeDelete, page.Documents[1].Operation)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectDriverBill,
	}, page.Documents[2].ObjectTypes)
	assert.Equal(t, services.AccountingChangeVoid, page.Documents[2].Operation)
	assert.Empty(t, page.Payments)
}

func TestReadChangesPagesDocumentsThatFilledTheChangeFeed(t *testing.T) {
	t.Parallel()

	var cdc strings.Builder
	cdc.WriteString(`{"CDCResponse":[{"QueryResponse":[{"Invoice":[`)
	for idx := range quickbooks.MaxChangesPerCall {
		if idx > 0 {
			cdc.WriteString(",")
		}
		cdc.WriteString(`{"Id":"i","TotalAmt":1}`)
	}
	cdc.WriteString(`]},{"Bill":[{"Id":"401","TotalAmt":1}]}]}]}`)

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	fake := &fakeChangeFeed{respond: func(call changeCall) string {
		if call.Path == "/cdc" {
			return cdc.String()
		}
		if strings.Contains(call.Query, "from Invoice") {
			return `{"QueryResponse":{"Invoice":[{"Id":"q1","TotalAmt":1}]}}`
		}
		return `{"QueryResponse":{}}`
	}}
	conn := fake.connector(t)
	request := func(cursor string) *services.ReadAccountingChangesRequest {
		return &services.ReadAccountingChangesRequest{
			Auth:      auth(),
			Cursor:    cursor,
			Now:       now,
			Documents: true,
		}
	}

	page, err := conn.ReadChanges(t.Context(), request(conn.ChangeCursorAt(now.Add(-time.Hour))))
	require.NoError(t, err)
	assert.True(t, page.More)
	require.Len(t, page.Documents, 1, "the capped invoices are re-read by query, the rest kept")
	assert.Equal(t, "401", page.Documents[0].ExternalID)

	page, err = conn.ReadChanges(t.Context(), request(page.NextCursor))
	require.NoError(t, err)
	assert.False(t, page.More)
	require.Len(t, page.Documents, 1)
	assert.Equal(t, "q1", page.Documents[0].ExternalID)
	assert.Contains(t, fake.recorded()[1].Query, "from Invoice")
}
