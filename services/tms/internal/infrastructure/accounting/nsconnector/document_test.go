package nsconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func salesDoc(kind accountingsync.SyncObjectType) *accountingsyncports.AccountingSalesDocument {
	return &accountingsyncports.AccountingSalesDocument{
		Auth:               testAuth(),
		RequestID:          testRequestID,
		Kind:               kind,
		CustomerExternalID: testCustomer,
		DocNumber:          "TRN-INV-1001",
		TxnDate:            "2026-10-01",
		DueDate:            "2026-10-31",
		TermExternalID:     testTerm,
		CurrencyCode:       "USD",
		PrivateNote:        "Shipment S-1",
		CustomerMemo:       "Thank you",
		Lines: []accountingsyncports.AccountingDocumentLine{
			{
				Description:    "Linehaul",
				ItemExternalID: testItem,
				Quantity:       decimal.NewFromInt(2),
				UnitPrice:      decimal.NewFromInt(500),
				Amount:         decimal.NewFromInt(1000),
			},
			{
				ItemExternalID: "79",
				Quantity:       decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromInt(100),
				Amount:         decimal.NewFromInt(250),
			},
		},
		Refs: map[string]string{"kept": "yes"},
	}
}

func TestCreateInvoiceUpsertsByRequestIDAndReadsTheNumberBack(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordInvoice, testRequestID)) {
			return upserted(recordInvoice, testInvoice)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	result, err := conn.CreateSalesDocument(t.Context(), salesDoc(accountingsync.SyncObjectInvoice))
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)
	assert.Equal(t, "DOC-"+testInvoice, result.DocNumber)
	assert.Equal(t, testInvoice, result.Refs[accountingsync.ExternalRefDocument])
	assert.Equal(t, "yes", result.Refs["kept"])

	put, ok := fake.find(http.MethodPut, eidPath(recordInvoice, testRequestID))
	require.True(t, ok)
	body := put.Body
	assert.Equal(t, testCustomer, refID(t, body, "entity"))
	assert.Equal(t, testSubsidiary, refID(t, body, "subsidiary"))
	assert.Equal(t, testTerm, refID(t, body, "terms"))
	assert.Equal(t, "2026-10-01", body["tranDate"])
	assert.Equal(t, "2026-10-31", body["dueDate"])
	assert.Equal(t, "TRN-INV-1001", body["otherRefNum"])
	assert.Equal(t, "Shipment S-1", body["memo"])
	assert.Equal(t, "Thank you", body["message"])
	assert.NotContains(t, body, "currency")
	assert.NotContains(t, body, "apply")
	lines := items(t, body, "item")
	require.Len(t, lines, 2)
	assert.Equal(t, testItem, lines[0]["item"].(map[string]any)["id"])
	assert.InDelta(t, 2.0, lines[0]["quantity"], 0)
	assert.InDelta(t, 500.0, lines[0]["rate"], 0)
	assert.InDelta(t, 1000.0, lines[0]["amount"], 0)
	assert.InDelta(
		t,
		1.0,
		lines[1]["quantity"],
		0,
		"a quantity that does not multiply out is one line",
	)
	assert.InDelta(t, 250.0, lines[1]["rate"], 0)
	assert.Equal(t, defaultLineText, lines[1]["description"])

	read, ok := fake.find(http.MethodGet, recordPath(recordInvoice, testInvoice))
	require.True(t, ok)
	assert.Equal(t, fieldTranID, read.Query.Get("fields"))
}

func TestRetriedCreateLandsOnTheSameExternalID(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordInvoice, testRequestID)) {
			return upserted(recordInvoice, testInvoice)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	first, err := conn.CreateSalesDocument(
		t.Context(),
		salesDoc(accountingsync.SyncObjectDebitMemo),
	)
	require.NoError(t, err)
	second, err := conn.CreateSalesDocument(
		t.Context(),
		salesDoc(accountingsync.SyncObjectDebitMemo),
	)
	require.NoError(t, err)
	assert.Equal(t, first.ExternalID, second.ExternalID)
	assert.Equal(t, 2, fake.count(http.MethodPut, eidPath(recordInvoice, testRequestID)))
	for _, call := range fake.writes() {
		assert.Equal(t, http.MethodPut, call.Method, "a retry never posts a second record")
	}
	assert.Zero(t, conn.DocumentLimits().IdempotencyWindow)
}

func TestForeignCurrencyDocumentsCarryTheCurrencyAndRate(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.is(http.MethodPut, eidPath(recordCreditMemo, testRequestID)):
			return upserted(recordCreditMemo, testCreditMemo)
		case call.queries("FROM currency c"):
			return fixtureReply(t, http.StatusOK, "currencies.json")
		case call.queries("FROM subsidiary s"):
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": testSubsidiary, "name": "Acme US", "currency": "USD",
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	doc := salesDoc(accountingsync.SyncObjectCreditMemo)
	doc.Auth.CompanyCode = ""
	doc.CurrencyCode = "cad"
	doc.ExchangeRate = decimal.RequireFromString("0.7312")
	doc.ApplyToExternalID = testInvoice
	doc.ApplyAmount = decimal.NewFromInt(1250)
	result, err := conn.CreateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.Refs[accountingsync.ExternalRefApplication])

	put, ok := fake.find(http.MethodPut, eidPath(recordCreditMemo, testRequestID))
	require.True(t, ok)
	assert.Equal(t, "3", refID(t, put.Body, "currency"))
	assert.InDelta(t, 0.7312, put.Body["exchangeRate"], 0)
	assert.NotContains(t, put.Body, "dueDate")
	assert.NotContains(t, put.Body, "terms")
	apply := items(t, put.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, testInvoice, apply[0]["doc"].(map[string]any)["id"])
	assert.Equal(t, true, apply[0]["apply"])
	assert.InDelta(t, 1250.0, apply[0]["amount"], 0)

	doc.CurrencyCode = "JPY"
	_, err = conn.CreateSalesDocument(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorCurrency, syncErrorOf(t, err).Category)
	assert.Equal(t, 1, countQueries(fake, "FROM currency c"), "the currency book is cached")
}

func countQueries(fake *fakeNS, fragment string) int {
	n := 0
	for _, call := range fake.all() {
		if call.queries(fragment) {
			n++
		}
	}
	return n
}

func TestSalesDocumentsRefuseUnmappedAndNegativeLines(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	doc := salesDoc(accountingsync.SyncObjectInvoice)
	doc.Lines[0].ItemExternalID = ""
	_, err := conn.CreateSalesDocument(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)

	doc = salesDoc(accountingsync.SyncObjectInvoice)
	doc.Lines[1].Amount = decimal.NewFromInt(-5)
	_, err = conn.CreateSalesDocument(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorValidation, syncErrorOf(t, err).Category)

	doc.Lines = nil
	_, err = conn.CreateSalesDocument(t.Context(), doc)
	require.ErrorIs(t, err, errLinesRequired)

	doc = salesDoc(accountingsync.SyncObjectCustomerPayment)
	_, err = conn.CreateSalesDocument(t.Context(), doc)
	require.ErrorIs(t, err, errDocumentKind)

	doc = salesDoc(accountingsync.SyncObjectInvoice)
	doc.RequestID = ""
	_, err = conn.CreateSalesDocument(t.Context(), doc)
	require.ErrorIs(t, err, errRequestID)
	assert.Empty(t, fake.writes())
}

func TestUpdateSalesDocumentPatchesAndReplacesTheLines(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	doc := salesDoc(accountingsync.SyncObjectInvoice)
	doc.ExternalID = testInvoice
	result, err := conn.UpdateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)
	assert.Equal(t, "DOC-"+testInvoice, result.DocNumber)

	patch, ok := fake.find(http.MethodPatch, recordPath(recordInvoice, testInvoice))
	require.True(t, ok)
	assert.Equal(t, "item", patch.Query.Get("replace"))
	assert.Len(t, items(t, patch.Body, "item"), 2)
	assert.Zero(t, fake.count(http.MethodPut, eidPath(recordInvoice, testRequestID)))

	doc.ExternalID = ""
	_, err = conn.UpdateSalesDocument(t.Context(), doc)
	require.ErrorIs(t, err, errExternalID)
}

func TestUpdateInAClosedPeriodIsAClosedPeriodFailure(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{respond: func(call nsCall) reply {
		if call.Method == http.MethodPatch {
			return fixtureReply(t, http.StatusBadRequest, "error_period_closed.json")
		}
		return reply{}
	}})
	doc := salesDoc(accountingsync.SyncObjectInvoice)
	doc.ExternalID = testInvoice
	_, err := conn.UpdateSalesDocument(t.Context(), doc)
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, classified.Category)
	assert.Equal(t, closedPeriodResolved, classified.Resolution)
}

func voidRef(
	kind accountingsync.SyncObjectType,
	id string,
) *accountingsyncports.AccountingDocumentRef {
	return &accountingsyncports.AccountingDocumentRef{
		Auth:       testAuth(),
		RequestID:  testVoidRequest,
		Kind:       kind,
		ExternalID: id,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: id},
	}
}

func stateResponder(
	t *testing.T,
	state string,
	extra func(call nsCall) reply,
) func(call nsCall) reply {
	return func(call nsCall) reply {
		if call.queries("LEFT JOIN accountingperiod ap") {
			return fixtureReply(t, http.StatusOK, state)
		}
		if extra != nil {
			return extra(call)
		}
		return reply{}
	}
}

func TestVoidInvoiceInAnOpenPeriodDeletesIt(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: stateResponder(t, "state_open.json", nil)}
	conn := testConnector(t, fake)
	result, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)
	assert.Empty(t, result.Refs[refCorrective])
	assert.Equal(t,
		[]string{"DELETE " + recordPath(recordInvoice, testInvoice)},
		methods(fake.writes()),
	)
	state, ok := fake.query("LEFT JOIN accountingperiod ap")
	require.True(t, ok)
	assert.Equal(t, []any{float64(1201)}, state.Params)
}

func reversalResponder(t *testing.T, state string) func(call nsCall) reply {
	return stateResponder(t, state, func(call nsCall) reply {
		switch {
		case call.is(http.MethodGet, recordPath(recordInvoice, testInvoice)) &&
			call.Query.Get("expandSubResources") == "true":
			return fixtureReply(t, http.StatusOK, "invoice.json")
		case call.is(http.MethodPut, eidPath(recordCreditMemo, testVoidRequest)):
			return upserted(recordCreditMemo, "1310")
		case call.is(http.MethodDelete, recordPath(recordInvoice, testInvoice)):
			return fixtureReply(t, http.StatusBadRequest, "error_user_error.json")
		}
		return reply{}
	})
}

func TestVoidInvoiceInAClosedPeriodCreditsItInstead(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: reversalResponder(t, "state_closed.json")}
	conn := testConnector(t, fake)
	result, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)
	assert.Equal(t, "1310", result.Refs[refCorrective])
	assert.Zero(t, fake.count(http.MethodDelete, recordPath(recordInvoice, testInvoice)))

	put, ok := fake.find(http.MethodPut, eidPath(recordCreditMemo, testVoidRequest))
	require.True(t, ok)
	assert.Equal(t, testCustomer, refID(t, put.Body, "entity"))
	assert.Equal(t, testSubsidiary, refID(t, put.Body, "subsidiary"))
	assert.Equal(t, "1", refID(t, put.Body, "currency"))
	assert.Equal(t, testNow.Format(dateLayout), put.Body["tranDate"])
	assert.Equal(t, "Reverses INV-1001", put.Body["memo"])
	lines := items(t, put.Body, "item")
	require.Len(t, lines, 2)
	assert.Equal(t, "Linehaul DAL-HOU", lines[0]["description"])
	assert.InDelta(t, 2.0, lines[1]["quantity"], 0)
	apply := items(t, put.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, testInvoice, apply[0]["doc"].(map[string]any)["id"])
	assert.InDelta(t, 1250.0, apply[0]["amount"], 0)
}

func TestVoidInvoiceWhoseDeleteIsRefusedCreditsIt(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: reversalResponder(t, "state_open.json")}
	conn := testConnector(t, fake)
	result, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)
	assert.Equal(t, "1310", result.Refs[refCorrective])
	assert.Equal(t, []string{
		"DELETE " + recordPath(recordInvoice, testInvoice),
		"PUT " + eidPath(recordCreditMemo, testVoidRequest),
	}, methods(fake.writes()))
}

func TestVoidInvoiceWithAPaymentAppliedIsAConflict(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: stateResponder(t, "state_applied.json", nil)}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	syncErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConflict, syncErr.Category)
	assert.Equal(t, "Unapply the payment in NetSuite, then retry", syncErr.Resolution)
	assert.Empty(t, fake.writes())

	_, ok := fake.query("t.externalid = ?")
	assert.True(t, ok, "a reversal this void already wrote is looked for first")
}

func TestVoidInvoiceResumesAReversalItAlreadyWrote(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: stateResponder(t, "state_applied.json", func(call nsCall) reply {
		if call.queries("t.externalid = ?") {
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "1310", "type": "CustCred", "externalid": testVoidRequest,
			})}
		}
		return reply{}
	})}
	conn := testConnector(t, fake)
	result, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)
	assert.Equal(t, "1310", result.Refs[refCorrective])
	assert.Empty(t, fake.writes())
	call, _ := fake.query("t.externalid = ?")
	assert.Equal(t, []any{typeCreditMemo, testVoidRequest}, call.Params)
}

func TestVoidOfAMissingOrVoidedInvoiceIsDone(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)

	voided := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("LEFT JOIN accountingperiod ap") {
			return reply{
				status: http.StatusOK,
				body: rowsJSON(t, false, stateRow(testInvoice, map[string]any{
					"voided": "T",
				})),
			}
		}
		return reply{}
	}}
	conn = testConnector(t, voided)
	_, err = conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectInvoice, testInvoice),
	)
	require.NoError(t, err)
	assert.Empty(t, fake.writes())
	assert.Empty(t, voided.writes())
}

func TestVoidCreditMemoUnappliesThenDeletes(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: stateResponder(t, "state_open.json", func(call nsCall) reply {
		if call.is(http.MethodGet, recordPath(recordCreditMemo, testCreditMemo)) {
			return reply{status: http.StatusOK, body: `{"id":"1301","apply":{"items":[` +
				`{"doc":1201,"apply":true,"amount":100.0},{"doc":1202,"apply":false,"amount":0}]}}`}
		}
		return reply{}
	})}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectCreditMemo, testCreditMemo),
	)
	require.NoError(t, err)
	writes := fake.writes()
	assert.Equal(t, []string{
		"PATCH " + recordPath(recordCreditMemo, testCreditMemo),
		"DELETE " + recordPath(recordCreditMemo, testCreditMemo),
	}, methods(writes))
	apply := items(t, writes[0].Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, "1201", apply[0]["doc"].(map[string]any)["id"])
	assert.Equal(t, false, apply[0]["apply"])
}

func TestVoidCreditMemoInAClosedPeriodIsRefused(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: stateResponder(t, "state_closed.json", nil)}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(
		t.Context(),
		voidRef(accountingsync.SyncObjectCreditMemo, testCreditMemo),
	)
	syncErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, syncErr.Category)
	assert.Contains(t, syncErr.Resolution, "Reverse the credit memo in NetSuite")
	assert.Empty(t, fake.writes())
}

func TestDocumentURLAndLimits(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	auth := testAuth()
	link := func(kind accountingsync.SyncObjectType, id string) string {
		return conn.DocumentURL(
			auth,
			accountingsyncports.AccountingDocumentLink{Kind: kind, ExternalID: id},
		)
	}
	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/accounting/transactions/transaction.nl?id=1201",
		link(accountingsync.SyncObjectInvoice, testInvoice),
	)
	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/accounting/transactions/transaction.nl?id=1701",
		link(accountingsync.SyncObjectCarrierBillPay, testBillPayment),
	)
	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/common/entity/custjob.nl?id=341",
		link(accountingsync.SyncObjectCustomer, testCustomer),
	)
	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/common/entity/vendor.nl?id=512",
		link(accountingsync.SyncObjectDriverVendor, testVendor),
	)
	assert.Empty(t, link(accountingsync.SyncObjectJournalEntry, "1"))
	assert.Empty(t, link(accountingsync.SyncObjectInvoice, "eid:x"))
	assert.Empty(t, link(accountingsync.SyncObjectCustomer, "abc"))
	auth.RealmID = "bad realm"
	assert.Empty(t, link(accountingsync.SyncObjectInvoice, testInvoice))

	limits := conn.DocumentLimits()
	assert.Equal(t, accountingsyncports.AccountingDocumentLimits{
		MaxDocNumberLength:      45,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       0,
	}, limits)
}
