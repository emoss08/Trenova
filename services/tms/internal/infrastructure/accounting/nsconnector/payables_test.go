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

func billDoc() *accountingsyncports.AccountingPurchaseDocument {
	return &accountingsyncports.AccountingPurchaseDocument{
		Auth:             testAuth(),
		RequestID:        testRequestID,
		Kind:             accountingsync.SyncObjectCarrierBill,
		VendorExternalID: testVendor,
		DocNumber:        "BILL-77",
		TxnDate:          "2026-10-02",
		DueDate:          "2026-11-01",
		CurrencyCode:     "USD",
		PrivateNote:      "Carrier settlement CS-12",
		Lines: []accountingsyncports.AccountingPurchaseLine{
			{
				Description:       "Load 4411",
				AccountExternalID: testExpense,
				Amount:            decimal.NewFromInt(900),
			},
		},
	}
}

func TestCreateVendorBillUsesExpenseLinesAndTheBillNumber(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordVendorBill, testRequestID)) {
			return upserted(recordVendorBill, testBill)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	result, err := conn.CreatePurchaseDocument(t.Context(), billDoc())
	require.NoError(t, err)
	assert.Equal(t, testBill, result.ExternalID)
	assert.Equal(t, recordVendorBill, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, "false", result.Refs[accountingsync.ExternalRefCreditDocument])
	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/accounting/transactions/transaction.nl?id=1501",
		result.Refs[accountingsync.ExternalRefURL],
	)

	put, ok := fake.find(http.MethodPut, eidPath(recordVendorBill, testRequestID))
	require.True(t, ok)
	assert.Equal(t, testVendor, refID(t, put.Body, "entity"))
	assert.Equal(t, testSubsidiary, refID(t, put.Body, "subsidiary"))
	assert.Equal(t, "BILL-77", put.Body["tranId"])
	assert.Equal(t, "2026-11-01", put.Body["dueDate"])
	assert.NotContains(t, put.Body, "account")
	expense := items(t, put.Body, "expense")
	require.Len(t, expense, 1)
	assert.Equal(t, testExpense, expense[0]["account"].(map[string]any)["id"])
	assert.InDelta(t, 900.0, expense[0]["amount"], 0)
	assert.Equal(t, "Load 4411", expense[0]["memo"])
}

func TestCreateVendorCreditAndRefuseUnmappedLines(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordVendorCredit, testRequestID)) {
			return upserted(recordVendorCredit, testVendorCredit)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	doc := billDoc()
	doc.VendorCredit = true
	result, err := conn.CreatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, recordVendorCredit, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, "true", result.Refs[accountingsync.ExternalRefCreditDocument])
	put, _ := fake.find(http.MethodPut, eidPath(recordVendorCredit, testRequestID))
	assert.NotContains(t, put.Body, "dueDate")

	doc = billDoc()
	doc.Lines[0].AccountExternalID = ""
	_, err = conn.CreatePurchaseDocument(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)
	doc = billDoc()
	doc.Kind = accountingsync.SyncObjectInvoice
	_, err = conn.CreatePurchaseDocument(t.Context(), doc)
	require.ErrorIs(t, err, errDocumentKind)
}

func TestUpdatePurchaseDocumentReplacesTheExpenseLines(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	doc := billDoc()
	doc.ExternalID = testBill
	result, err := conn.UpdatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testBill, result.ExternalID)
	patch, ok := fake.find(http.MethodPatch, recordPath(recordVendorBill, testBill))
	require.True(t, ok)
	assert.Equal(t, "expense", patch.Query.Get("replace"))

	doc.ExternalID = ""
	_, err = conn.UpdatePurchaseDocument(t.Context(), doc)
	require.ErrorIs(t, err, errExternalID)
}

func billStateResponder(
	t *testing.T,
	overrides map[string]any,
	extra func(nsCall) reply,
) func(nsCall) reply {
	return func(call nsCall) reply {
		if call.queries("LEFT JOIN accountingperiod ap") {
			fields := map[string]any{
				"type":                   "VendBill",
				"foreigntotal":           "900.00",
				"foreignamountremaining": "900.00",
			}
			for key, value := range overrides {
				fields[key] = value
			}
			return reply{
				status: http.StatusOK,
				body:   rowsJSON(t, false, stateRow(testBill, fields)),
			}
		}
		if extra != nil {
			return extra(call)
		}
		return reply{}
	}
}

func billRef(refs map[string]string) *accountingsyncports.AccountingDocumentRef {
	ref := voidRef(accountingsync.SyncObjectCarrierBill, testBill)
	for key, value := range refs {
		ref.Refs[key] = value
	}
	return ref
}

func TestVoidBillInAnOpenPeriodDeletesIt(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: billStateResponder(t, nil, nil)}
	conn := testConnector(t, fake)
	_, err := conn.VoidPurchaseDocument(t.Context(), billRef(nil))
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"DELETE " + recordPath(recordVendorBill, testBill)},
		methods(fake.writes()),
	)
}

func TestVoidBillInAnAPLockedPeriodWritesAVendorCreditAppliedToIt(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{
		respond: billStateResponder(t, map[string]any{"aplocked": "T"}, func(call nsCall) reply {
			switch {
			case call.is(http.MethodGet, recordPath(recordVendorBill, testBill)):
				return fixtureReply(t, http.StatusOK, "vendor_bill.json")
			case call.is(http.MethodPut, eidPath(recordVendorCredit, testVoidRequest)):
				return upserted(recordVendorCredit, "1610")
			}
			return reply{}
		}),
	}
	conn := testConnector(t, fake)
	result, err := conn.VoidPurchaseDocument(t.Context(), billRef(nil))
	require.NoError(t, err)
	assert.Equal(t, "1610", result.Refs[refCorrective])
	assert.Equal(
		t,
		[]string{"PUT " + eidPath(recordVendorCredit, testVoidRequest)},
		methods(fake.writes()),
	)
	put, _ := fake.find(http.MethodPut, eidPath(recordVendorCredit, testVoidRequest))
	assert.Equal(t, testVendor, refID(t, put.Body, "entity"))
	expense := items(t, put.Body, "expense")
	require.Len(t, expense, 1)
	assert.Equal(t, testExpense, expense[0]["account"].(map[string]any)["id"])
	apply := items(t, put.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, testBill, apply[0]["doc"].(map[string]any)["id"])
	assert.InDelta(t, 900.0, apply[0]["amount"], 0)
	assert.NotContains(t, put.Body, "item")

	arLocked := &fakeNS{respond: billStateResponder(t, map[string]any{"arlocked": "T"}, nil)}
	conn = testConnector(t, arLocked)
	_, err = conn.VoidPurchaseDocument(t.Context(), billRef(nil))
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"DELETE " + recordPath(recordVendorBill, testBill)},
		methods(arLocked.writes()),
		"an A/R lock does not block a bill",
	)
}

func TestVoidPaidBillIsAConflict(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{
		respond: billStateResponder(t, map[string]any{"foreignamountremaining": "0"}, nil),
	}
	conn := testConnector(t, fake)
	_, err := conn.VoidPurchaseDocument(t.Context(), billRef(nil))
	assert.Equal(t, accountingsync.SyncErrorConflict, syncErrorOf(t, err).Category)
	assert.Empty(t, fake.writes())
}

func TestVoidVendorCreditUnappliesThenDeletes(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{
		respond: billStateResponder(t, map[string]any{"type": "VendCred"}, func(call nsCall) reply {
			if call.is(http.MethodGet, recordPath(recordVendorCredit, testBill)) {
				return reply{status: http.StatusOK, body: `{"id":"1501","apply":{"items":[]}}`}
			}
			return reply{}
		}),
	}
	conn := testConnector(t, fake)
	_, err := conn.VoidPurchaseDocument(t.Context(), billRef(map[string]string{
		accountingsync.ExternalRefCreditDocument: "true",
	}))
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"DELETE " + recordPath(recordVendorCredit, testBill)},
		methods(fake.writes()),
	)

	_, err = conn.VoidPurchaseDocument(t.Context(), billRef(map[string]string{
		accountingsync.ExternalRefCreditDocument: "maybe",
	}))
	require.ErrorIs(t, err, errPurchaseDocumentType)
}

func TestPurchaseIsCreditReadsEitherRef(t *testing.T) {
	t.Parallel()

	credit, err := purchaseIsCredit(
		map[string]string{accountingsync.ExternalRefDocumentType: recordVendorCredit},
	)
	require.NoError(t, err)
	assert.True(t, credit)
	credit, err = purchaseIsCredit(nil)
	require.NoError(t, err)
	assert.False(t, credit)
	_, err = purchaseIsCredit(map[string]string{accountingsync.ExternalRefDocumentType: "journal"})
	require.ErrorIs(t, err, errPurchaseDocumentType)
}

func billPaymentDoc() *accountingsyncports.AccountingBillPaymentDocument {
	return &accountingsyncports.AccountingBillPaymentDocument{
		Auth:                  testAuth(),
		RequestID:             testRequestID,
		Kind:                  accountingsync.SyncObjectCarrierBillPay,
		VendorExternalID:      testVendor,
		BankAccountExternalID: testBank,
		BillExternalID:        testBill,
		DocNumber:             "CHK-1009",
		TxnDate:               "2026-10-07",
		CurrencyCode:          "USD",
		PrivateNote:           "Carrier payment",
		Amount:                decimal.NewFromInt(900),
	}
}

func TestCreateBillPaymentAppliesItToTheBill(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordVendorPayment, testRequestID)) {
			return upserted(recordVendorPayment, testBillPayment)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	result, err := conn.CreateBillPayment(t.Context(), billPaymentDoc())
	require.NoError(t, err)
	assert.Equal(t, testBillPayment, result.ExternalID)
	assert.Equal(t, docTypeVendorPayment, result.Refs[accountingsync.ExternalRefDocumentType])

	put, _ := fake.find(http.MethodPut, eidPath(recordVendorPayment, testRequestID))
	assert.Equal(t, testVendor, refID(t, put.Body, "entity"))
	assert.Equal(t, testBank, refID(t, put.Body, "account"))
	assert.Equal(t, "CHK-1009", put.Body["tranId"])
	apply := items(t, put.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, testBill, apply[0]["doc"].(map[string]any)["id"])
	assert.InDelta(t, 900.0, apply[0]["amount"], 0)

	doc := billPaymentDoc()
	doc.BankAccountExternalID = ""
	_, err = conn.CreateBillPayment(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)
	doc = billPaymentDoc()
	doc.Amount = decimal.Zero
	_, err = conn.CreateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, errAmountInvalid)
	doc = billPaymentDoc()
	doc.Kind = accountingsync.SyncObjectCarrierBill
	_, err = conn.CreateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, errDocumentKind)
}

func TestUpdateBillPaymentPatchesIt(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodGet, recordPath(recordVendorPayment, testBillPayment)) &&
			call.Query.Get("expandSubResources") == "true" {
			return fixtureReply(t, http.StatusOK, "vendor_payment.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	doc := billPaymentDoc()
	doc.ExternalID = testBillPayment
	doc.Amount = decimal.NewFromInt(800)
	result, err := conn.UpdateBillPayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testBillPayment, result.ExternalID)
	assert.Equal(
		t,
		[]string{"PATCH " + recordPath(recordVendorPayment, testBillPayment)},
		methods(fake.writes()),
	)
	patch, _ := fake.find(http.MethodPatch, recordPath(recordVendorPayment, testBillPayment))
	apply := items(t, patch.Body, "apply")
	require.Len(t, apply, 1)
	assert.InDelta(t, 800.0, apply[0]["amount"], 0)

	doc.ExternalID = ""
	_, err = conn.UpdateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, errExternalID)
}

func TestVoidBillPaymentUnappliesThenDeletesOrAsksForAReversal(t *testing.T) {
	t.Parallel()

	open := &fakeNS{
		respond: billStateResponder(t, map[string]any{"type": "VendPymt"}, func(call nsCall) reply {
			if call.is(http.MethodGet, recordPath(recordVendorPayment, testBillPayment)) {
				return fixtureReply(t, http.StatusOK, "vendor_payment.json")
			}
			return reply{}
		}),
	}
	conn := testConnector(t, open)
	ref := voidRef(accountingsync.SyncObjectDriverBillPay, testBillPayment)
	_, err := conn.VoidPurchaseDocument(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"PATCH " + recordPath(recordVendorPayment, testBillPayment),
		"DELETE " + recordPath(recordVendorPayment, testBillPayment),
	}, methods(open.writes()))

	locked := &fakeNS{respond: billStateResponder(t, map[string]any{"aplocked": "T"}, nil)}
	conn = testConnector(t, locked)
	_, err = conn.VoidPurchaseDocument(t.Context(), ref)
	syncErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, syncErr.Category)
	assert.Equal(t, reversePayment, syncErr.Resolution)
	assert.Empty(t, locked.writes())

	ref.Kind = accountingsync.SyncObjectInvoice
	_, err = conn.VoidPurchaseDocument(t.Context(), ref)
	require.ErrorIs(t, err, errDocumentKind)
}
