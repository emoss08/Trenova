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

func paymentDoc() *accountingsyncports.AccountingPaymentDocument {
	return &accountingsyncports.AccountingPaymentDocument{
		Auth:                     testAuth(),
		RequestID:                testRequestID,
		CustomerExternalID:       testCustomer,
		TxnDate:                  "2026-10-06",
		CurrencyCode:             "USD",
		DepositAccountExternalID: testDeposit,
		ReferenceNumber:          "ACH-5521",
		PrivateNote:              "Payment P-9",
		TotalAmount:              decimal.NewFromInt(1450),
		ShortPayItemExternalID:   testWriteOffItem,
		Applications: []accountingsyncports.AccountingPaymentApplication{
			{
				InvoiceExternalID: testInvoice,
				InvoiceNumber:     "INV-1001",
				AppliedAmount:     decimal.NewFromInt(1200),
				ShortPayAmount:    decimal.NewFromInt(50),
			},
			{
				InvoiceExternalID: "1202",
				InvoiceNumber:     "INV-1002",
				AppliedAmount:     decimal.NewFromInt(250),
			},
		},
	}
}

func shortPayRequest() string {
	return accountingsync.SyncStepRequestID(testRequestID, stepShortPayBase)
}

func TestSavePaymentWritesTheShortPayCreditThenThePayment(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.is(http.MethodPut, eidPath(recordCreditMemo, shortPayRequest())):
			return upserted(recordCreditMemo, "1320")
		case call.is(http.MethodPut, eidPath(recordCustomerPayment, testRequestID)):
			return upserted(recordCustomerPayment, testPayment)
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	result, err := conn.SavePayment(t.Context(), paymentDoc())
	require.NoError(t, err)
	assert.Equal(t, testPayment, result.ExternalID)
	assert.Equal(t, "DOC-"+testPayment, result.DocNumber)
	assert.Equal(t, docTypeCustomerPayment, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, "1320", result.Refs[accountingsync.ExternalRefShortPayPrefix+testInvoice])

	writes := fake.writes()
	require.Equal(t, []string{
		"PUT " + eidPath(recordCreditMemo, shortPayRequest()),
		"PUT " + eidPath(recordCustomerPayment, testRequestID),
	}, methods(writes))

	memo := writes[0].Body
	assert.Equal(t, testCustomer, refID(t, memo, "entity"))
	assert.Equal(t, "Short pay on INV-1001", memo["memo"])
	memoLine := items(t, memo, "item")[0]
	assert.Equal(t, testWriteOffItem, memoLine["item"].(map[string]any)["id"])
	assert.InDelta(t, 50.0, memoLine["amount"], 0)
	memoApply := items(t, memo, "apply")[0]
	assert.Equal(t, testInvoice, memoApply["doc"].(map[string]any)["id"])
	assert.InDelta(t, 50.0, memoApply["amount"], 0)

	payment := writes[1].Body
	assert.Equal(t, testCustomer, refID(t, payment, "customer"))
	assert.Equal(t, testSubsidiary, refID(t, payment, "subsidiary"))
	assert.Equal(t, testDeposit, refID(t, payment, "account"))
	assert.InDelta(t, 1450.0, payment["payment"], 0)
	assert.Equal(t, "ACH-5521", payment["checkNum"])
	apply := items(t, payment, "apply")
	require.Len(t, apply, 2)
	assert.InDelta(t, 1200.0, apply[0]["amount"], 0)
	assert.Equal(t, "1202", apply[1]["doc"].(map[string]any)["id"])
}

func TestSavePaymentUpdateUnappliesInvoicesNoLongerPaid(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodGet, recordPath(recordCustomerPayment, testPayment)) &&
			call.Query.Get("expandSubResources") == "true" {
			return fixtureReply(t, http.StatusOK, "customer_payment.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	doc := paymentDoc()
	doc.ExternalID = testPayment
	doc.Applications[0].ShortPayCreditExternalID = "1320"
	doc.Applications = doc.Applications[:1]
	doc.TotalAmount = decimal.NewFromInt(1200)
	result, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testPayment, result.ExternalID)

	writes := fake.writes()
	require.Equal(
		t,
		[]string{"PATCH " + recordPath(recordCustomerPayment, testPayment)},
		methods(writes),
	)
	apply := items(t, writes[0].Body, "apply")
	require.Len(t, apply, 2)
	assert.Equal(t, testInvoice, apply[0]["doc"].(map[string]any)["id"])
	assert.Equal(t, true, apply[0]["apply"])
	assert.Equal(t, "1202", apply[1]["doc"].(map[string]any)["id"])
	assert.Equal(t, false, apply[1]["apply"])
	assert.Empty(t, writes[0].Query.Get("replace"))
}

func TestSavePaymentRefusesUnmappedAccountsAndItems(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	doc := paymentDoc()
	doc.DepositAccountExternalID = ""
	_, err := conn.SavePayment(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)

	doc = paymentDoc()
	doc.ShortPayItemExternalID = ""
	_, err = conn.SavePayment(t.Context(), doc)
	syncErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErr.Category)
	assert.Equal(t, accountingsync.ItemRoleShortPayWriteOff, syncErr.Code)

	doc = paymentDoc()
	doc.Applications[1].AppliedAmount = decimal.NewFromInt(-1)
	_, err = conn.SavePayment(t.Context(), doc)
	require.ErrorIs(t, err, errAmountInvalid)

	doc = paymentDoc()
	doc.Applications[1].InvoiceExternalID = ""
	_, err = conn.SavePayment(t.Context(), doc)
	require.ErrorIs(t, err, errApplyTarget)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)
	assert.Empty(t, fake.writes())
}

func paymentVoidResponder(t *testing.T, state string) func(call nsCall) reply {
	return func(call nsCall) reply {
		switch {
		case call.queries("LEFT JOIN accountingperiod ap"):
			return reply{
				status: http.StatusOK,
				body: rowsJSON(t, false, stateRow(testPayment, map[string]any{
					"type": "CustPymt", "closed": state,
				})),
			}
		case call.is(http.MethodGet, recordPath(recordCustomerPayment, testPayment)):
			return fixtureReply(t, http.StatusOK, "customer_payment.json")
		case call.is(http.MethodGet, recordPath(recordCreditMemo, "1320")):
			return reply{status: http.StatusOK, body: `{"id":"1320","apply":{"items":[` +
				`{"doc":{"id":"1201"},"apply":true,"amount":50.0}]}}`}
		}
		return reply{}
	}
}

func TestVoidPaymentUnappliesThenDeletes(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: paymentVoidResponder(t, "F")}
	conn := testConnector(t, fake)
	ref := voidRef(accountingsync.SyncObjectCustomerPayment, testPayment)
	ref.Refs[accountingsync.ExternalRefShortPayPrefix+testInvoice] = "1320"
	result, err := conn.VoidPayment(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, testPayment, result.ExternalID)

	writes := fake.writes()
	require.Equal(t, []string{
		"PATCH " + recordPath(recordCustomerPayment, testPayment),
		"DELETE " + recordPath(recordCustomerPayment, testPayment),
		"PATCH " + recordPath(recordCreditMemo, "1320"),
		"DELETE " + recordPath(recordCreditMemo, "1320"),
	}, methods(writes))
	apply := items(t, writes[0].Body, "apply")
	require.Len(t, apply, 2)
	for _, line := range apply {
		assert.Equal(t, false, line["apply"])
	}
	assert.Equal(t, testInvoice, apply[0]["doc"].(map[string]any)["id"])
	assert.Equal(t, "1202", apply[1]["doc"].(map[string]any)["id"])
}

func TestVoidPaymentInAClosedPeriodAsksForAReversal(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: paymentVoidResponder(t, "T")}
	conn := testConnector(t, fake)
	_, err := conn.VoidPayment(
		t.Context(),
		voidRef(accountingsync.SyncObjectCustomerPayment, testPayment),
	)
	syncErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, syncErr.Category)
	assert.Equal(
		t,
		"Reverse the payment in NetSuite, then mark this record resolved",
		syncErr.Resolution,
	)
	assert.Empty(t, fake.writes())
}

func TestVoidPaymentWhoseDeleteIsRefusedForThePeriodAsksForAReversal(t *testing.T) {
	t.Parallel()

	responder := paymentVoidResponder(t, "F")
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodDelete, recordPath(recordCustomerPayment, testPayment)) {
			return fixtureReply(t, http.StatusBadRequest, "error_period_closed.json")
		}
		return responder(call)
	}}
	conn := testConnector(t, fake)
	_, err := conn.VoidPayment(
		t.Context(),
		voidRef(accountingsync.SyncObjectCustomerPayment, testPayment),
	)
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, classified.Category)
	assert.Equal(t, reversePayment, classified.Resolution)
}

func TestVoidPaymentOffsetsAShortPayCreditMemoInAClosedPeriod(t *testing.T) {
	t.Parallel()

	offsetID := testVoidRequest + "-1320"
	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.queries("LEFT JOIN accountingperiod ap"):
			if len(call.Params) == 1 && call.Params[0] == float64(1320) {
				return reply{status: http.StatusOK, body: rowsJSON(t, false, stateRow("1320",
					map[string]any{"type": "CustCred", "arlocked": "T"}))}
			}
			return reply{status: http.StatusOK, body: rowsJSON(t, false, stateRow(testPayment,
				map[string]any{"type": "CustPymt"}))}
		case call.is(http.MethodGet, recordPath(recordCustomerPayment, testPayment)):
			return fixtureReply(t, http.StatusOK, "customer_payment.json")
		case call.is(http.MethodGet, recordPath(recordCreditMemo, "1320")):
			return fixtureReply(t, http.StatusOK, "credit_memo.json")
		case call.is(http.MethodPut, eidPath(recordInvoice, offsetID)):
			return upserted(recordInvoice, "1211")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	ref := voidRef(accountingsync.SyncObjectCustomerPayment, testPayment)
	ref.Refs[accountingsync.ExternalRefShortPayPrefix+testInvoice] = "1320"
	result, err := conn.VoidPayment(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, "1211", result.Refs[refCorrective+"-"+testInvoice])
	writes := methods(fake.writes())
	assert.Contains(t, writes, "DELETE "+recordPath(recordCustomerPayment, testPayment))
	assert.Contains(t, writes, "PUT "+eidPath(recordInvoice, offsetID))
	assert.NotContains(t, writes, "DELETE "+recordPath(recordCreditMemo, "1320"))
}

func TestCreateCreditApplicationPatchesTheCreditMemoApplyList(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	result, err := conn.CreateCreditApplication(
		t.Context(),
		&accountingsyncports.AccountingCreditApplicationDocument{
			Auth:                 testAuth(),
			RequestID:            testRequestID,
			CustomerExternalID:   testCustomer,
			InvoiceExternalID:    testInvoice,
			CreditMemoExternalID: testCreditMemo,
			Amount:               decimal.NewFromInt(75),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, testCreditMemo, result.ExternalID)
	assert.Equal(t, testCreditMemo, result.Refs[accountingsync.ExternalRefDocument])
	assert.Equal(t, testInvoice, result.Refs[accountingsync.ExternalRefApplication])

	patch, ok := fake.find(http.MethodPatch, recordPath(recordCreditMemo, testCreditMemo))
	require.True(t, ok)
	apply := items(t, patch.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, true, apply[0]["apply"])
	assert.InDelta(t, 75.0, apply[0]["amount"], 0)
	assert.Empty(t, patch.Query.Get("replace"))

	_, err = conn.CreateCreditApplication(
		t.Context(),
		&accountingsyncports.AccountingCreditApplicationDocument{
			Auth:   testAuth(),
			Amount: decimal.Zero,
		},
	)
	require.ErrorIs(t, err, errAmountInvalid)
}

func TestVoidCreditApplicationTurnsTheApplyLineOff(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	ref := voidRef(accountingsync.SyncObjectCreditApplication, testCreditMemo)
	ref.Refs[accountingsync.ExternalRefApplication] = testInvoice
	_, err := conn.VoidCreditApplication(t.Context(), ref)
	require.NoError(t, err)
	patch, ok := fake.find(http.MethodPatch, recordPath(recordCreditMemo, testCreditMemo))
	require.True(t, ok)
	apply := items(t, patch.Body, "apply")
	require.Len(t, apply, 1)
	assert.Equal(t, testInvoice, apply[0]["doc"].(map[string]any)["id"])
	assert.Equal(t, false, apply[0]["apply"])
	assert.NotContains(t, apply[0], "amount")

	unknown := voidRef(accountingsync.SyncObjectCreditApplication, testCreditMemo)
	_, err = conn.VoidCreditApplication(t.Context(), unknown)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, syncErrorOf(t, err).Category)
	assert.Equal(t, 1, fake.count(http.MethodPatch, recordPath(recordCreditMemo, testCreditMemo)))
}
