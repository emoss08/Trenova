package qboconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const objectNotFoundFault = `{"Fault":{"type":"ValidationFault","Error":[{"Message":"Object Not Found","code":"610"}]}}`

func settlementBill() *services.AccountingPurchaseDocument {
	return &services.AccountingPurchaseDocument{
		Auth:                auth(),
		RequestID:           docRequestID,
		Kind:                accountingsync.SyncObjectCarrierBill,
		VendorExternalID:    "91",
		APAccountExternalID: "33",
		DocNumber:           "CINV-778",
		TxnDate:             "2026-09-20",
		DueDate:             "2026-10-05",
		CurrencyCode:        "USD",
		PrivateNote:         "Trenova carrier settlement CS-1042",
		Lines: []services.AccountingPurchaseLine{
			{
				Description:       "Purchased transportation",
				AccountExternalID: "61",
				Amount:            decimal.NewFromInt(1500),
			},
			{
				Description:       "Escrow withheld",
				AccountExternalID: "72",
				Amount:            decimal.NewFromInt(-150),
			},
		},
	}
}

func TestUpsertVendorCreatesA1099VendorWhenUnmapped(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"Vendor":{"Id":"95","DisplayName":"Dale Owner - Operator","Vendor1099":true}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.UpsertVendor(t.Context(), &services.AccountingVendorDocument{
		Auth:      auth(),
		RequestID: docRequestID,
		Party: services.AccountingPartyDraft{
			DisplayName: "Dale Owner: Operator",
			City:        "Reno",
			Is1099:      true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "95", result.ExternalID)
	assert.Equal(t, "Dale Owner - Operator", result.DocNumber)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/vendor", writes[0].Path)
	assert.Equal(t, docRequestID, writes[0].RequestID)
	assert.Equal(t, "Dale Owner - Operator", writes[0].Body["DisplayName"])
	assert.Equal(t, true, writes[0].Body["Vendor1099"])
	assert.NotContains(t, writes[0].Body, "Id")
}

func TestUpsertVendorUpdatesAMappedVendor(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{"Vendor":{"Id":"91","SyncToken":"4"}}`
		}
		return http.StatusOK, `{"Vendor":{"Id":"91","DisplayName":"Swift Haul LLC","SyncToken":"5"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.UpsertVendor(t.Context(), &services.AccountingVendorDocument{
		Auth:       auth(),
		RequestID:  docRequestID,
		ExternalID: "91",
		Party:      services.AccountingPartyDraft{DisplayName: "Swift Haul LLC"},
	})
	require.NoError(t, err)
	assert.Equal(t, "91", result.ExternalID)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/vendor", writes[0].Path)
	assert.Equal(t, "91", writes[0].Body["Id"])
	assert.Equal(t, "4", writes[0].Body["SyncToken"])
	assert.Equal(t, true, writes[0].Body["sparse"])
	assert.NotContains(t, writes[0].Body, "Vendor1099")
}

func TestCreatePurchaseDocumentWritesABill(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"Bill":{"Id":"210","DocNumber":"CINV-778","SyncToken":"0"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.CreatePurchaseDocument(t.Context(), settlementBill())
	require.NoError(t, err)
	assert.Equal(t, "210", result.ExternalID)
	assert.Equal(t, "CINV-778", result.DocNumber)
	assert.Equal(t, map[string]string{
		accountingsync.ExternalRefDocument: "210",
		accountingsync.ExternalRefDocumentType:                    "Bill",
		accountingsync.ExternalRefURL:                             conn.env.AppBaseURL() + "/app/bill?txnId=210",
	}, result.Refs)
	assert.Equal(t, result.Refs[accountingsync.ExternalRefURL],
		conn.DocumentURL(accountingsync.SyncObjectCarrierBill, "210"))

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/bill", writes[0].Path)
	assert.Equal(t, docRequestID, writes[0].RequestID)
	assert.Equal(t, map[string]any{"value": "91"}, writes[0].Body["VendorRef"])
	assert.Equal(t, map[string]any{"value": "33"}, writes[0].Body["APAccountRef"])
	assert.Equal(t, "2026-10-05", writes[0].Body["DueDate"])
	lines := writes[0].Body["Line"].([]any)
	require.Len(t, lines, 2)
	escrow := lines[1].(map[string]any)
	assert.InDelta(t, -150, escrow["Amount"], 0.001)
	assert.Equal(t, map[string]any{"AccountRef": map[string]any{"value": "72"}},
		escrow["AccountBasedExpenseLineDetail"])
}

func TestCreatePurchaseDocumentWritesAVendorCreditForANegativeNet(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"VendorCredit":{"Id":"211","DocNumber":"DS-77"}}`
	}}
	conn := documentConnector(t, fake)

	doc := settlementBill()
	doc.Kind = accountingsync.SyncObjectDriverBill
	doc.VendorCredit = true
	doc.DocNumber = "DS-77"
	doc.Lines = []services.AccountingPurchaseLine{
		{AccountExternalID: "72", Amount: decimal.NewFromInt(150)},
	}
	result, err := conn.CreatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, "211", result.ExternalID)
	assert.Equal(t, "VendorCredit", result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t,
		conn.env.AppBaseURL()+"/app/vendorcredit?txnId=211",
		result.Refs[accountingsync.ExternalRefURL],
		"a vendor credit links to its own screen, not the bill screen",
	)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/vendorcredit", writes[0].Path)
	assert.NotContains(t, writes[0].Body, "DueDate")
}

func TestCreatePurchaseDocumentRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		t.Fatal("nothing should be written")
		return 0, ""
	}}
	conn := documentConnector(t, fake)

	wrongKind := settlementBill()
	wrongKind.Kind = accountingsync.SyncObjectInvoice
	_, err := conn.CreatePurchaseDocument(t.Context(), wrongKind)
	require.ErrorIs(t, err, errDocumentKind)

	negative := settlementBill()
	negative.Lines[1].Amount = decimal.NewFromInt(-2000)
	_, err = conn.CreatePurchaseDocument(t.Context(), negative)
	require.ErrorIs(t, err, quickbooks.ErrNegativeAmount)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)
}

func TestVoidPurchaseDocumentDeletesByDocumentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		refs     map[string]string
		readPath string
		path     string
		reply    string
	}{
		{
			name:     "bill by default",
			refs:     map[string]string{accountingsync.ExternalRefDocument: "210"},
			readPath: "/bill/210",
			path:     "/bill",
			reply:    `{"Bill":{"Id":"210","status":"Deleted"}}`,
		},
		{
			name:     "vendor credit",
			refs:     map[string]string{accountingsync.ExternalRefDocumentType: "VendorCredit"},
			readPath: "/vendorcredit/210",
			path:     "/vendorcredit",
			reply:    `{"VendorCredit":{"Id":"210","status":"Deleted"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
				if call.Method == http.MethodGet {
					assert.Equal(t, tt.readPath, call.Path)
					return http.StatusOK, `{"Bill":{"Id":"210","SyncToken":"2"},"VendorCredit":{"Id":"210","SyncToken":"2"}}`
				}
				return http.StatusOK, tt.reply
			}}
			conn := documentConnector(t, fake)

			result, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
				Auth:       auth(),
				RequestID:  docRequestID,
				Kind:       accountingsync.SyncObjectCarrierBill,
				ExternalID: "210",
				Refs:       tt.refs,
			})
			require.NoError(t, err)
			assert.Equal(t, "210", result.ExternalID)
			assert.Equal(t, tt.refs, result.Refs)

			writes := fake.writes()
			require.Len(t, writes, 1)
			assert.Equal(t, tt.path, writes[0].Path)
			assert.Equal(t, "delete", writes[0].Operation)
			assert.Equal(t, docRequestID, writes[0].RequestID)
			assert.Equal(t, map[string]any{"Id": "210", "SyncToken": "2"}, writes[0].Body)
		})
	}
}

func TestVoidPurchaseDocumentTreatsAMissingDocumentAsDeleted(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusBadRequest, objectNotFoundFault
	}}
	conn := documentConnector(t, fake)

	_, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectDriverBill,
		ExternalID: "210",
		Refs:       map[string]string{accountingsync.ExternalRefDocumentType: "Bill"},
	})
	require.NoError(t, err, "a bill already gone counts as deleted")
	assert.Empty(t, fake.writes())
}

func TestVoidPurchaseDocumentReportsOtherFaults(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{"Bill":{"Id":"210","SyncToken":"2"}}`
		}
		return http.StatusBadRequest, `{"Fault":{"type":"ValidationFault","Error":[{"Message":"Closed","code":"6210"}]}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectCarrierBill,
		ExternalID: "210",
	})
	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, conn.ClassifyDocumentError(err).Category)
}

func TestVoidPurchaseDocumentRejectsUnknownKinds(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		t.Fatal("nothing should be sent")
		return 0, ""
	}}
	conn := documentConnector(t, fake)

	_, err := conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectCarrierBill,
		ExternalID: "210",
		Refs:       map[string]string{accountingsync.ExternalRefDocumentType: "Invoice"},
	})
	require.ErrorIs(t, err, errPurchaseDocumentType)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)

	_, err = conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectCarrierBillPay,
		ExternalID: "210",
	})
	require.ErrorIs(t, err, errDocumentKind)
}

func TestCreateBillPaymentPaysTheSettlementBill(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"BillPayment":{"Id":"212","DocNumber":"ACH-1"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.CreateBillPayment(t.Context(), &services.AccountingBillPaymentDocument{
		Auth:                  auth(),
		RequestID:             docRequestID,
		Kind:                  accountingsync.SyncObjectCarrierBillPay,
		VendorExternalID:      "91",
		BankAccountExternalID: "35",
		BillExternalID:        "210",
		DocNumber:             "ACH-1",
		TxnDate:               "2026-09-25",
		PrivateNote:           "Paid by ACH",
		Amount:                decimal.NewFromInt(1350),
	})
	require.NoError(t, err)
	assert.Equal(t, "212", result.ExternalID)
	assert.Equal(t, "ACH-1", result.DocNumber)
	assert.Equal(t, map[string]string{accountingsync.ExternalRefDocument: "212"}, result.Refs)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/billpayment", writes[0].Path)
	assert.Equal(t, docRequestID, writes[0].RequestID)
	assert.Equal(t, "Check", writes[0].Body["PayType"])
	assert.Equal(t, map[string]any{"BankAccountRef": map[string]any{"value": "35"}},
		writes[0].Body["CheckPayment"])
	assert.Equal(t, []any{map[string]any{
		"Amount":    float64(1350),
		"LinkedTxn": []any{map[string]any{"TxnId": "210", "TxnType": "Bill"}},
	}}, writes[0].Body["Line"])
}

func TestCreateBillPaymentRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		t.Fatal("nothing should be written")
		return 0, ""
	}}
	conn := documentConnector(t, fake)

	doc := &services.AccountingBillPaymentDocument{
		Auth:                  auth(),
		RequestID:             docRequestID,
		Kind:                  accountingsync.SyncObjectDriverBillPay,
		VendorExternalID:      "91",
		BankAccountExternalID: "35",
		BillExternalID:        "210",
		Amount:                decimal.Zero,
	}
	_, err := conn.CreateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, quickbooks.ErrNonPositiveAmount)

	doc.Kind = accountingsync.SyncObjectDriverBill
	_, err = conn.CreateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, errDocumentKind)
}

func TestPayablesValidationErrorsClassifyAsValidation(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	for _, err := range []error{
		quickbooks.ErrVendorRequired,
		quickbooks.ErrAccountRequired,
		quickbooks.ErrBankAccountRequired,
		quickbooks.ErrBillRequired,
		quickbooks.ErrNonPositiveAmount,
		quickbooks.ErrLinesRequired,
		errPurchaseDocumentType,
	} {
		classified := conn.ClassifyDocumentError(err)
		require.NotNil(t, classified, err)
		assert.Equal(t, accountingsync.SyncErrorValidation, classified.Category, err)
	}
}

func TestPayablesDocumentURLsAndLimits(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{Environment: "sandbox"})
	base := "https://app.sandbox.qbo.intuit.com"
	tests := []struct {
		kind accountingsync.SyncObjectType
		want string
	}{
		{kind: accountingsync.SyncObjectCarrierVendor, want: base + "/app/vendordetail?nameId=91"},
		{kind: accountingsync.SyncObjectDriverVendor, want: base + "/app/vendordetail?nameId=91"},
		{kind: accountingsync.SyncObjectCarrierBill, want: base + "/app/bill?txnId=91"},
		{kind: accountingsync.SyncObjectDriverBill, want: base + "/app/bill?txnId=91"},
		{kind: accountingsync.SyncObjectCarrierBillPay, want: base + "/app/billpayment?txnId=91"},
		{kind: accountingsync.SyncObjectDriverBillPay, want: base + "/app/billpayment?txnId=91"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, conn.DocumentURL(tt.kind, " 91 "), tt.kind)
	}
	assert.Empty(t, conn.DocumentURL(accountingsync.SyncObjectCarrierBill, ""))
	assert.Empty(t, conn.DocumentURL(accountingsync.SyncObjectType("Estimate"), "91"))
	assert.False(t, conn.DocumentLimits().CanVoidPurchaseDocument)
}
