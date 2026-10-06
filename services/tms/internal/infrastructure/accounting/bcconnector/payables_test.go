package bcconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func billDoc() *services.AccountingPurchaseDocument {
	return &services.AccountingPurchaseDocument{
		Auth:             testAuth(),
		RequestID:        testRequestID,
		Kind:             accountingsync.SyncObjectCarrierBill,
		VendorExternalID: testVendor,
		DocNumber:        "OO-78",
		TxnDate:          "2026-10-02",
		DueDate:          "2026-11-01",
		CurrencyCode:     "USD",
		Lines: []services.AccountingPurchaseLine{
			{Description: "Linehaul", AccountExternalID: testExpense, Amount: decimal.NewFromInt(800)},
			{Description: "Fuel", AccountExternalID: strings.ToUpper(testExpense), Amount: decimal.NewFromInt(200)},
		},
	}
}

func newBillServer(t *testing.T) *salesServer {
	return &salesServer{
		t:      t,
		entity: "purchaseInvoices",
		newID:  testNewBill,
		posted: docSpec{id: testNewBill, number: "108002", status: "Open", total: 1000, remaining: 1000},
	}
}

func TestCreateBillPostsAccountLinesUnderTheVendorInvoiceNumber(t *testing.T) {
	t.Parallel()

	server := newBillServer(t)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.CreatePurchaseDocument(t.Context(), billDoc())
	require.NoError(t, err)
	assert.Equal(t, testNewBill, result.ExternalID)
	assert.Equal(t, "108002", result.DocNumber)
	assert.Equal(t, docTypePurchaseInvoice, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, "false", result.Refs[accountingsync.ExternalRefCreditDocument])
	assert.Equal(t, "1000", result.Refs[refAccountPrefix+testExpense])
	assert.Contains(t, result.Refs[accountingsync.ExternalRefURL], "page=138")
	assert.Contains(t, result.Refs[accountingsync.ExternalRefURL], "108002")

	create, ok := fake.find(http.MethodPost, "/purchaseInvoices")
	require.True(t, ok)
	assert.Equal(t, "OO-78", create.Body["vendorInvoiceNumber"])
	assert.Equal(t, testVendor, create.Body["vendorId"])
	assert.Equal(t, "2026-11-01", create.Body["dueDate"])
	sent := lines(t, create.Body, "purchaseInvoiceLines")
	require.Len(t, sent, 2)
	assert.Equal(t, "Account", sent[0]["lineType"])
	assert.Equal(t, testExpense, sent[0]["accountId"])
	assert.InDelta(t, 800, sent[0]["directUnitCost"], 0)
	assert.InDelta(t, 1, sent[0]["quantity"], 0)
	_, posted := fake.find(http.MethodPost, entity("purchaseInvoices", testNewBill)+postAction)
	assert.True(t, posted)
}

func TestCreateVendorCreditIsAPurchaseCreditMemo(t *testing.T) {
	t.Parallel()

	server := newBillServer(t)
	server.entity = "purchaseCreditMemos"
	server.newID = testVendorCredit
	server.posted = docSpec{id: testVendorCredit, number: "109001", status: "Open"}
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := billDoc()
	doc.VendorCredit = true
	result, err := conn.CreatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testVendorCredit, result.ExternalID)
	assert.Equal(t, "true", result.Refs[accountingsync.ExternalRefCreditDocument])
	assert.True(t, accountingsync.IsCreditDocument(result.Refs))
	assert.Contains(t, result.Refs[accountingsync.ExternalRefURL], "page=140")
	create, ok := fake.find(http.MethodPost, "/purchaseCreditMemos")
	require.True(t, ok)
	assert.Equal(t, "OO-78", create.Body["vendorCreditMemoNumber"])
	assert.NotContains(t, create.Body, "dueDate")

	unmapped := billDoc()
	unmapped.Lines[0].AccountExternalID = ""
	_, err = conn.CreatePurchaseDocument(t.Context(), unmapped)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)
	notABill := billDoc()
	notABill.Kind = accountingsync.SyncObjectInvoice
	_, err = conn.CreatePurchaseDocument(t.Context(), notABill)
	require.ErrorIs(t, err, errDocumentKind)
}

func voidCreditServer(t *testing.T, bill docSpec) *salesServer {
	server := newBillServer(t)
	server.entity = "purchaseCreditMemos"
	server.newID = testVendorCredit
	server.posted = docSpec{id: testVendorCredit, number: "109002", status: "Open"}
	server.extra = func(call bcCall) (int, string) {
		if call.is(http.MethodGet, entity("purchaseInvoices", testBill)) {
			return http.StatusOK, bill.json(t)
		}
		return 0, ""
	}
	return server
}

func TestVoidBillPostsAFullCreditMemoAppliedToIt(t *testing.T) {
	t.Parallel()

	server := voidCreditServer(t, docSpec{
		id: testBill, number: "108001", status: "Open", total: 900, remaining: 900,
		party: testVendor, currency: "CAD",
	})
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, entity("purchaseInvoices", testBill)) {
			return http.StatusOK, `{"id":"` + testBill + `","number":"108001","status":"Open",` +
				`"vendorId":"` + testVendor + `","currencyCode":"CAD",` +
				`"totalAmountIncludingTax":900,"remainingAmount":900}`
		}
		return server.respond(call)
	}}
	conn := testConnector(t, fake)

	result, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       testAuth(),
		RequestID:  testRequestID,
		Kind:       accountingsync.SyncObjectCarrierBill,
		ExternalID: testBill,
		Refs: map[string]string{
			accountingsync.ExternalRefCreditDocument: "false",
			refAccountPrefix + testExpense:           "700",
			refAccountPrefix + testBank:              "200",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, testVendorCredit, result.Refs[voidCreditKey(testBill)])

	create, ok := fake.find(http.MethodPost, "/purchaseCreditMemos")
	require.True(t, ok)
	assert.Equal(t, testBill, create.Body["invoiceId"])
	assert.Equal(t, testVendor, create.Body["vendorId"])
	assert.Equal(t, "CAD", create.Body["currencyCode"])
	assert.Equal(t, requestReference(testRequestID), create.Body["vendorCreditMemoNumber"])
	sent := lines(t, create.Body, "purchaseCreditMemoLines")
	require.Len(t, sent, 2)
	assert.Equal(t, testBank, sent[0]["accountId"])
	assert.InDelta(t, 200, sent[0]["directUnitCost"], 0)
	assert.InDelta(t, 700, sent[1]["directUnitCost"], 0)
	_, posted := fake.find(http.MethodPost, entity("purchaseCreditMemos", testVendorCredit)+postAction)
	assert.True(t, posted)
}

func TestVoidBillRefusalsAndResume(t *testing.T) {
	t.Parallel()

	ref := &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectDriverBill,
		ExternalID: testBill, Refs: map[string]string{refAccountPrefix + testExpense: "900"},
	}

	paid := voidCreditServer(t, docSpec{
		id: testBill, status: "Paid", total: 900, remaining: 0, party: testVendor,
	})
	fake := &fakeBC{respond: paid.respond}
	_, err := testConnector(t, fake).VoidPurchaseDocument(t.Context(), ref)
	assert.Equal(t, accountingsync.SyncErrorConflict, syncErrorOf(t, err).Category)
	assert.Empty(t, fake.writes())

	done := voidCreditServer(t, docSpec{
		id: testBill, status: "Paid", total: 900, remaining: 0, party: testVendor,
	})
	done.listed = listJSON(t, docSpec{
		id: testVendorCredit, number: "109002", status: "Open",
		reference: requestReference(testRequestID),
	})
	fake = &fakeBC{respond: done.respond}
	result, err := testConnector(t, fake).VoidPurchaseDocument(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, testVendorCredit, result.Refs[voidCreditKey(testBill)])
	assert.Empty(t, fake.writes())

	unknown := voidCreditServer(t, docSpec{
		id: testBill, status: "Open", total: 900, remaining: 900, party: testVendor,
	})
	fake = &fakeBC{respond: unknown.respond}
	noLines := *ref
	noLines.Refs = map[string]string{}
	_, err = testConnector(t, fake).VoidPurchaseDocument(t.Context(), &noLines)
	assert.Equal(t, "bill-lines", syncErrorOf(t, err).Code)

	payment := *ref
	payment.Kind = accountingsync.SyncObjectCarrierBillPay
	_, err = testConnector(t, fake).VoidPurchaseDocument(t.Context(), &payment)
	assert.Equal(t, "reverse-transaction", syncErrorOf(t, err).Code)
}

func TestVoidVendorCreditCancelsIt(t *testing.T) {
	t.Parallel()

	server := newBillServer(t)
	server.entity = "purchaseCreditMemos"
	server.existing = map[string]docSpec{
		testVendorCredit: {id: testVendorCredit, status: "Open", total: 50},
	}
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	_, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       testAuth(),
		Kind:       accountingsync.SyncObjectCarrierBill,
		ExternalID: testVendorCredit,
		Refs:       map[string]string{accountingsync.ExternalRefDocumentType: docTypePurchaseCreditMemo},
	})
	require.NoError(t, err)
	_, cancelled := fake.find(http.MethodPost, entity("purchaseCreditMemos", testVendorCredit)+cancelAction)
	assert.True(t, cancelled)

	_, err = conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCarrierBill, ExternalID: testVendorCredit,
		Refs: map[string]string{accountingsync.ExternalRefDocumentType: "Other"},
	})
	require.ErrorIs(t, err, errPurchaseDocumentType)
}

func TestUpdateBillCreditsTheOldBillThenPostsTheNewOne(t *testing.T) {
	t.Parallel()

	bills := newBillServer(t)
	bills.listed = listJSON(t, docSpec{
		id: testBill, number: "108001", status: "Open", reference: "OO-78", total: 900,
	})
	memos := voidCreditServer(t, docSpec{
		id: testBill, number: "108001", status: "Open", total: 900, remaining: 900, party: testVendor,
	})
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if code, body := memos.respond(call); code != 0 {
			return code, body
		}
		return bills.respond(call)
	}}
	conn := testConnector(t, fake)

	doc := billDoc()
	doc.ExternalID = testBill
	result, err := conn.UpdatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testNewBill, result.ExternalID)
	assert.Equal(t, testBill, result.Refs[refReplaced])
	assert.Equal(t, testVendorCredit, result.Refs[refCorrective])

	memo, ok := fake.find(http.MethodPost, "/purchaseCreditMemos")
	require.True(t, ok)
	sent := lines(t, memo.Body, "purchaseCreditMemoLines")
	require.Len(t, sent, 1)
	assert.Equal(t, testExpense, sent[0]["accountId"])
	assert.InDelta(t, 900, sent[0]["directUnitCost"], 0)
	created, ok := fake.find(http.MethodPost, "/purchaseInvoices")
	require.True(t, ok)
	assert.Equal(t, revisedNumber(doc.DocNumber, doc.RequestID), created.Body["vendorInvoiceNumber"])
	assert.NotEqual(t, doc.DocNumber, revisedNumber(doc.DocNumber, doc.RequestID))

	_, err = conn.UpdatePurchaseDocument(t.Context(), billDoc())
	require.ErrorIs(t, err, errExternalID)
}

func TestUpsertVendorUpdatesWithTheETagOrRefusesADuplicateName(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodGet, entity("vendors", testVendor)):
			return http.StatusOK, `{"@odata.etag":"W/\"v1\"","id":"` + testVendor +
				`","displayName":"Owner Operator Inc","currencyCode":"CAD"}`
		case call.is(http.MethodPatch, entity("vendors", testVendor)):
			return http.StatusOK, `{"id":"` + testVendor + `","displayName":"Owner Operator LLC"}`
		case call.is(http.MethodGet, "/vendors"):
			return http.StatusOK, readFixture(t, "vendors.json")
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	result, err := conn.UpsertVendor(t.Context(), &services.AccountingVendorDocument{
		Auth:       testAuth(),
		ExternalID: testVendor,
		Party:      services.AccountingPartyDraft{DisplayName: "Owner Operator LLC", City: "Dallas"},
	})
	require.NoError(t, err)
	assert.Equal(t, testVendor, result.ExternalID)
	assert.Equal(t, "Owner Operator LLC", result.DocNumber)
	patch, ok := fake.find(http.MethodPatch, entity("vendors", testVendor))
	require.True(t, ok)
	assert.Equal(t, `W/"v1"`, patch.IfMatch)
	assert.Equal(t, "CAD", patch.Body["currencyCode"])
	assert.Equal(t, "Dallas", patch.Body["city"])

	_, err = conn.UpsertVendor(t.Context(), &services.AccountingVendorDocument{
		Auth:  testAuth(),
		Party: services.AccountingPartyDraft{DisplayName: "Owner Operator Inc"},
	})
	require.ErrorIs(t, err, errDuplicateName)
	assert.Equal(t, accountingsync.SyncErrorDuplicate, conn.ClassifyDocumentError(err).Category)
}

func TestVoidBillDeletesADraftAndIgnoresAMissingBill(t *testing.T) {
	t.Parallel()

	ref := &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectCarrierBill,
		ExternalID: testBill,
	}
	draft := voidCreditServer(t, docSpec{id: testBill, status: "Draft", etag: `W/"d1"`})
	draft.existing = map[string]docSpec{}
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodDelete, entity("purchaseInvoices", testBill)) {
			return http.StatusNoContent, ""
		}
		return draft.respond(call)
	}}
	_, err := testConnector(t, fake).VoidPurchaseDocument(t.Context(), ref)
	require.NoError(t, err)
	deleted, ok := fake.find(http.MethodDelete, entity("purchaseInvoices", testBill))
	require.True(t, ok)
	assert.Equal(t, `W/"d1"`, deleted.IfMatch)

	missing := &fakeBC{}
	_, err = testConnector(t, missing).VoidPurchaseDocument(t.Context(), ref)
	require.NoError(t, err)
	assert.Empty(t, missing.writes())
}

func TestARevisedBillNumberFitsAndDiffersFromTheOriginal(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("9", businesscentral.MaxExternalDocumentNumberLength)
	revised := revisedNumber(long, "req-1")
	assert.Len(t, revised, businesscentral.MaxExternalDocumentNumberLength)
	assert.True(t, strings.HasSuffix(revised, revisionSeparator+revised[len(revised)-revisionHashLength:]))
	assert.NotEqual(t, revisedNumber("OO-78", "req-1"), revisedNumber("OO-78", "req-2"))
	assert.Equal(t, revisedNumber("OO-78", "req-1"), revisedNumber("OO-78", "req-1"))
	assert.Empty(t, revisedNumber(" ", "req-1"))
}
