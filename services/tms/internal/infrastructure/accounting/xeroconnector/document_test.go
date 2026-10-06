package xeroconnector

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func TestDocumentLimits(t *testing.T) {
	t.Parallel()

	limits := testConnector(t, &fakeXero{}).DocumentLimits()
	assert.Equal(t, services.AccountingDocumentLimits{
		MaxDocNumberLength:      xero.MaxDocumentNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       6 * time.Minute,
	}, limits)
}

func TestDocumentURL(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	cases := []struct {
		kind accountingsync.SyncObjectType
		code string
		want string
	}{
		{accountingsync.SyncObjectInvoice, "!Ab12C", "https://go.xero.com/app/%21Ab12C/invoicing/view/" + testInvoice},
		{accountingsync.SyncObjectDebitMemo, "!Ab12C", "https://go.xero.com/app/%21Ab12C/invoicing/view/" + testInvoice},
		{accountingsync.SyncObjectCreditMemo, "!Ab12C", "https://go.xero.com/app/%21Ab12C/invoicing/view/" + testInvoice},
		{accountingsync.SyncObjectCarrierBill, "!Ab12C", "https://go.xero.com/app/%21Ab12C/bills/view/" + testInvoice},
		{accountingsync.SyncObjectDriverBill, "!Ab12C", "https://go.xero.com/app/%21Ab12C/bills/view/" + testInvoice},
		{accountingsync.SyncObjectCustomerPayment, "!Ab12C", ""},
		{accountingsync.SyncObjectCustomer, "!Ab12C", ""},
		{accountingsync.SyncObjectInvoice, "", ""},
	}
	for _, tc := range cases {
		auth := testAuth()
		auth.CompanyCode = tc.code
		assert.Equal(t, tc.want, conn.DocumentURL(auth, services.AccountingDocumentLink{Kind: tc.kind, ExternalID: testInvoice}), string(tc.kind))
	}
	assert.Empty(t, conn.DocumentURL(testAuth(), services.AccountingDocumentLink{Kind: accountingsync.SyncObjectInvoice, ExternalID: " "}))
}

func salesDoc(kind accountingsync.SyncObjectType) *services.AccountingSalesDocument {
	return &services.AccountingSalesDocument{
		Auth:               testAuth(),
		RequestID:          testRequestID,
		Kind:               kind,
		CustomerExternalID: testCustomer,
		DocNumber:          "INV-1001",
		TxnDate:            "2026-09-01",
		DueDate:            "2026-10-01",
		CurrencyCode:       "USD",
		PrivateNote:        "Load 42",
		Lines: []services.AccountingDocumentLine{
			{
				Description:       "Line haul",
				AccountExternalID: testRevenue,
				Quantity:          dec("2"),
				UnitPrice:         dec("250"),
				Amount:            dec("500"),
			},
			{
				Description:       "Fuel",
				AccountExternalID: testRevenue,
				Quantity:          dec("3"),
				UnitPrice:         dec("33.33"),
				Amount:            dec("100"),
			},
		},
	}
}

const invoiceWritten = `{"Invoices":[{"InvoiceID":"` + testInvoice + `","InvoiceNumber":"INV-1001","Type":"ACCREC","Status":"AUTHORISED","Total":600}]}`

func TestCreateInvoiceWritesAccountCodedLines(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodPut && call.Path == "/Invoices" {
			return http.StatusOK, invoiceWritten
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	result, err := conn.CreateSalesDocument(t.Context(), salesDoc(accountingsync.SyncObjectInvoice))
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)
	assert.Equal(t, "INV-1001", result.DocNumber)
	assert.Equal(t, testInvoice, result.Refs[accountingsync.ExternalRefDocument])

	call, ok := fake.find(http.MethodPut, "/Invoices")
	require.True(t, ok)
	assert.Equal(t, testRequestID, call.Key)
	assert.Equal(t, testTenant, call.Tenant)
	doc := firstDoc(t, call.Body, "Invoices")
	assert.Equal(t, "ACCREC", doc["Type"])
	assert.Equal(t, "AUTHORISED", doc["Status"])
	assert.Equal(t, "NoTax", doc["LineAmountTypes"])
	assert.Equal(t, "Load 42", doc["Reference"])
	assert.Equal(t, "2026-10-01", doc["DueDate"])
	lines := lineItems(t, call.Body, "Invoices")
	require.Len(t, lines, 2)
	assert.Equal(t, "200", lines[0]["AccountCode"])
	assert.InDelta(t, 2.0, lines[0]["Quantity"], 0)
	assert.InDelta(t, 250.0, lines[0]["UnitAmount"], 0)
	assert.InDelta(t, 1.0, lines[1]["Quantity"], 0)
	assert.InDelta(t, 100.0, lines[1]["UnitAmount"], 0)
	assert.InDelta(t, 100.0, lines[1]["LineAmount"], 0)
}

func TestAccountCodesAreCachedAcrossDocuments(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodPut && call.Path == "/Invoices" {
			return http.StatusOK, invoiceWritten
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	for range 3 {
		_, err := conn.CreateSalesDocument(t.Context(), salesDoc(accountingsync.SyncObjectInvoice))
		require.NoError(t, err)
	}
	reads := 0
	for _, call := range fake.all() {
		if call.Path == "/Accounts" {
			reads++
		}
	}
	assert.Equal(t, 1, reads)
}

func TestAccountCodeLookupRefetchesAStaleIndex(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	codes := newAccountCodes(func() time.Time { return now })
	fake := &fakeXero{}
	conn := testConnector(t, fake)
	conn.codes = codes
	client, err := conn.client(testAuth())
	require.NoError(t, err)

	code, ok, err := codes.codeFor(t.Context(), client, testRevenue)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "200", code)

	_, ok, err = codes.codeFor(t.Context(), client, testUncoded)
	require.NoError(t, err)
	assert.False(t, ok)

	now = now.Add(accountCodesRefetch + time.Second)
	_, ok, err = codes.codeFor(t.Context(), client, "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2dff")
	require.NoError(t, err)
	assert.False(t, ok)

	id, ok, err := codes.idFor(t.Context(), client, "499")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, testWriteOff, id)

	reads := 0
	for _, call := range fake.all() {
		if call.Path == "/Accounts" {
			reads++
		}
	}
	assert.Equal(t, 2, reads)
}

func TestSalesLineWithoutAUsableAccountIsAMappingError(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{}
	conn := testConnector(t, fake)

	missing := salesDoc(accountingsync.SyncObjectInvoice)
	missing.Lines[0].AccountExternalID = ""
	_, err := conn.CreateSalesDocument(t.Context(), missing)
	require.Error(t, err)
	assert.Equal(t, accountingsync.SyncErrorMapping, conn.ClassifyDocumentError(err).Category)

	uncoded := salesDoc(accountingsync.SyncObjectInvoice)
	uncoded.Lines[1].AccountExternalID = testUncoded
	_, err = conn.CreateSalesDocument(t.Context(), uncoded)
	require.Error(t, err)
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorMapping, classified.Category)
	assert.Contains(t, classified.Message, "Line 2 (Fuel)")
	assert.Empty(t, fake.writes())
}

func TestCreditMemoIsAllocatedToTheInvoiceItNames(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodPut && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","CreditNoteNumber":"CM-7","Type":"ACCRECCREDIT","Status":"AUTHORISED"}]}`
		case call.Method == http.MethodPut && call.Path == "/CreditNotes/"+testCreditNote+"/Allocations":
			return http.StatusOK, `{"Allocations":[{"AllocationID":"` + testAllocation + `","Amount":100,"Invoice":{"InvoiceID":"` + testInvoice + `"}}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	doc := salesDoc(accountingsync.SyncObjectCreditMemo)
	doc.DocNumber = "CM-7"
	doc.ApplyToExternalID = testInvoice
	doc.ApplyAmount = dec("100")
	result, err := conn.CreateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testCreditNote, result.ExternalID)
	assert.Equal(t, "CM-7", result.DocNumber)
	assert.Equal(t, testAllocation, result.Refs[accountingsync.ExternalRefApplication])

	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, testRequestID, writes[0].Key)
	note := firstDoc(t, writes[0].Body, "CreditNotes")
	assert.Equal(t, "ACCRECCREDIT", note["Type"])
	assert.NotContains(t, note, "DueDate")
	assert.Equal(t, testRequestID+"-1", writes[1].Key)

	again, err := conn.CreateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectCreditMemo,
		ApplyToExternalID: testInvoice, ApplyAmount: dec("100"), Refs: result.Refs,
	})
	require.NoError(t, err)
	assert.Equal(t, testCreditNote, again.ExternalID)
	assert.Len(t, fake.writes(), 2)
}

func TestDebitMemoIsAnInvoiceWithAMarkedReference(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodPut && call.Path == "/Invoices" {
			return http.StatusOK, invoiceWritten
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	_, err := conn.CreateSalesDocument(t.Context(), salesDoc(accountingsync.SyncObjectDebitMemo))
	require.NoError(t, err)
	call, _ := fake.find(http.MethodPut, "/Invoices")
	assert.Equal(t, "Debit memo. Load 42", firstDoc(t, call.Body, "Invoices")["Reference"])

	_, err = conn.CreateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Kind: accountingsync.SyncObjectCustomerPayment,
	})
	require.ErrorIs(t, err, errDocumentKind)
}

func TestUpdateSalesDocumentPostsToTheInvoice(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodPost && call.Path == "/Invoices/"+testInvoice {
			return http.StatusOK, invoiceWritten
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	doc := salesDoc(accountingsync.SyncObjectInvoice)
	doc.ExternalID = testInvoice
	result, err := conn.UpdateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)
	call, ok := fake.find(http.MethodPost, "/Invoices/"+testInvoice)
	require.True(t, ok)
	assert.Equal(t, testInvoice, firstDoc(t, call.Body, "Invoices")["InvoiceID"])

	doc.ExternalID = ""
	_, err = conn.UpdateSalesDocument(t.Context(), doc)
	require.ErrorIs(t, err, errExternalID)
}

func TestVoidInvoiceRemovesPaymentsAndAllocationsFirst(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodGet && call.Path == "/Invoices":
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testInvoice + `","Status":"AUTHORISED","Payments":[{"PaymentID":"` + testPayment + `","Amount":50}],"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","AppliedAmount":25}]}]}`
		case call.Method == http.MethodGet && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"AUTHORISED","Allocations":[{"AllocationID":"` + testAllocation + `","Amount":25,"Invoice":{"InvoiceID":"` + testInvoice + `"}},{"AllocationID":"b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0b","Amount":10,"Invoice":{"InvoiceID":"` + testInvoice2 + `"}}]}]}`
		case call.Method == http.MethodPost && call.Path == "/Payments/"+testPayment:
			return http.StatusOK, `{"Payments":[]}`
		case call.Method == http.MethodDelete:
			return http.StatusOK, `{}`
		case call.Method == http.MethodPost && call.Path == "/Invoices/"+testInvoice:
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testInvoice + `","Status":"VOIDED"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	result, err := conn.VoidSalesDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectInvoice,
		ExternalID: testInvoice, Refs: map[string]string{"document": testInvoice},
	})
	require.NoError(t, err)
	assert.Equal(t, testInvoice, result.ExternalID)

	writes := fake.writes()
	require.Len(t, writes, 3)
	assert.Equal(t, "/Payments/"+testPayment, writes[0].Path)
	assert.Equal(t, testRequestID+"-1", writes[0].Key)
	assert.Equal(t, http.MethodDelete, writes[1].Method)
	assert.Equal(t, "/CreditNotes/"+testCreditNote+"/Allocations/"+testAllocation, writes[1].Path)
	assert.Equal(t, "/Invoices/"+testInvoice, writes[2].Path)
	assert.Equal(t, testRequestID, writes[2].Key)
	assert.Equal(t, "VOIDED", firstDoc(t, writes[2].Body, "Invoices")["Status"])
}

func TestVoidOfAnAlreadyVoidedOrMissingDocumentWritesNothing(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Invoices":
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testInvoice + `","Status":"VOIDED"}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)
	for _, kind := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCreditMemo,
	} {
		_, err := conn.VoidSalesDocument(t.Context(), &services.AccountingDocumentRef{
			Auth: testAuth(), RequestID: testRequestID, Kind: kind, ExternalID: testInvoice,
		})
		require.NoError(t, err)
	}
	assert.Empty(t, fake.writes())
}

func TestVoidCreditMemoRemovesItsAllocations(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodGet && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"AUTHORISED","Allocations":[{"AllocationID":"` + testAllocation + `","Amount":25,"Invoice":{"InvoiceID":"` + testInvoice + `"}},{"AllocationID":"b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0c","Amount":5,"IsDeleted":true}]}]}`
		case call.Method == http.MethodDelete:
			return http.StatusOK, `{}`
		case call.Method == http.MethodPost && call.Path == "/CreditNotes/"+testCreditNote:
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"VOIDED"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectCreditMemo,
		ExternalID: testCreditNote,
	})
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, http.MethodDelete, writes[0].Method)
	assert.Equal(t, "VOIDED", firstDoc(t, writes[1].Body, "CreditNotes")["Status"])
}

func TestCreditApplicationAllocatesAndVoidDeletesTheAllocation(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodPut:
			return http.StatusOK, `{"Allocations":[{"AllocationID":"` + testAllocation + `","Amount":40,"Invoice":{"InvoiceID":"` + testInvoice + `"}}]}`
		case call.Method == http.MethodDelete:
			return http.StatusOK, `{}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	result, err := conn.CreateCreditApplication(t.Context(), &services.AccountingCreditApplicationDocument{
		Auth: testAuth(), RequestID: testRequestID, InvoiceExternalID: testInvoice,
		CreditMemoExternalID: testCreditNote, Amount: dec("40"), TxnDate: "2026-09-02",
	})
	require.NoError(t, err)
	assert.Equal(t, testAllocation, result.ExternalID)
	assert.Equal(t, testCreditNote, result.Refs[refCreditNote])
	call, ok := fake.find(http.MethodPut, "/CreditNotes/"+testCreditNote+"/Allocations")
	require.True(t, ok)
	assert.Equal(t, "2026-09-02", firstDoc(t, call.Body, "Allocations")["Date"])

	_, err = conn.VoidCreditApplication(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: "void", Kind: accountingsync.SyncObjectCreditApplication,
		ExternalID: testAllocation, Refs: result.Refs,
	})
	require.NoError(t, err)
	_, ok = fake.find(http.MethodDelete, "/CreditNotes/"+testCreditNote+"/Allocations/"+testAllocation)
	assert.True(t, ok)

	_, err = conn.VoidCreditApplication(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), ExternalID: testAllocation, Refs: map[string]string{},
	})
	require.ErrorIs(t, err, errCreditNoteID)

	_, err = conn.CreateCreditApplication(t.Context(), &services.AccountingCreditApplicationDocument{
		Auth: testAuth(), Amount: decimal.Zero,
	})
	require.ErrorIs(t, err, xero.ErrAmountNotPositive)
}

func TestFindSalesDocumentByNumberAndTotal(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Invoices":
			return http.StatusOK, `{"Invoices":[
			 {"InvoiceID":"` + testInvoice2 + `","InvoiceNumber":"INV-1001","Type":"ACCREC","Status":"VOIDED","Total":600},
			 {"InvoiceID":"` + testInvoice + `","InvoiceNumber":"INV-1001","Type":"ACCREC","Status":"AUTHORISED","Total":600}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","CreditNoteNumber":"CM-7","Type":"ACCRECCREDIT","Status":"PAID","Total":100}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	find := func(kind accountingsync.SyncObjectType, number, total string) (*services.AccountingDocumentResult, bool) {
		t.Helper()
		found, ok, err := conn.FindDocument(t.Context(), &services.AccountingFindDocumentRequest{
			Auth: testAuth(), Kind: kind, DocNumber: number, Total: dec(total),
			RequestID: testRequestID,
		})
		require.NoError(t, err)
		return found, ok
	}

	found, ok := find(accountingsync.SyncObjectInvoice, "INV-1001", "600")
	require.True(t, ok)
	assert.Equal(t, testInvoice, found.ExternalID)
	call, _ := fake.find(http.MethodGet, "/Invoices")
	assert.Equal(t, "INV-1001", call.Query.Get("InvoiceNumbers"))
	assert.Contains(t, call.Query.Get("where"), `Type=="ACCREC"`)

	_, ok = find(accountingsync.SyncObjectInvoice, "INV-1001", "601")
	assert.False(t, ok)
	_, ok = find(accountingsync.SyncObjectInvoice, "INV-1001", "0")
	assert.True(t, ok)
	_, ok = find(accountingsync.SyncObjectInvoice, "", "600")
	assert.False(t, ok)
	_, ok = find(accountingsync.SyncObjectInvoice, `bad"number`, "600")
	assert.False(t, ok)

	found, ok = find(accountingsync.SyncObjectCreditMemo, "CM-7", "100")
	require.True(t, ok)
	assert.Equal(t, testCreditNote, found.ExternalID)

	_, _, err := conn.FindDocument(t.Context(), &services.AccountingFindDocumentRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCustomer,
	})
	require.ErrorIs(t, err, errDocumentKind)
}

func apiError(status int, typ string, messages ...string) *xero.APIError {
	return &xero.APIError{Status: status, Type: typ, ValidationMessages: messages}
}

func TestClassifyDocumentError(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	mapping := &accountingsync.SyncError{Category: accountingsync.SyncErrorMapping, Message: "m"}
	cases := []struct {
		name string
		err  error
		want accountingsync.SyncErrorCategory
	}{
		{"sync error passes through", mapping, accountingsync.SyncErrorMapping},
		{"not configured", ErrNotConfigured, accountingsync.SyncErrorConfiguration},
		{"revoked", &xero.OAuthError{Status: 400, Code: "invalid_grant"}, accountingsync.SyncErrorAuth},
		{"unauthorized", apiError(401, ""), accountingsync.SyncErrorAuth},
		{"scope", apiError(401, xero.TypeInsufficientScope), accountingsync.SyncErrorAuth},
		{"disconnected", apiError(403, xero.TypeAuthUnsuccessful), accountingsync.SyncErrorAuth},
		{"not found", apiError(404, ""), accountingsync.SyncErrorNotFound},
		{"rate limited", apiError(429, ""), accountingsync.SyncErrorRateLimited},
		{"offline", apiError(503, ""), accountingsync.SyncErrorTransient},
		{"server", apiError(500, ""), accountingsync.SyncErrorTransient},
		{"transport", &restx.TransportError{Err: errors.New("reset")}, accountingsync.SyncErrorTransient},
		{"deadline", context.DeadlineExceeded, accountingsync.SyncErrorTransient},
		{"duplicate number", apiError(400, xero.TypeValidation, "Invoice # must be unique."), accountingsync.SyncErrorDuplicate},
		{"lock date", apiError(400, xero.TypeValidation, "The document date cannot be before the period lock date of 30 June 2026"), accountingsync.SyncErrorClosedPeriod},
		{"archived account", apiError(400, xero.TypeValidation, "Account code '310' has been archived, or deleted."), accountingsync.SyncErrorMapping},
		{"validation", apiError(400, xero.TypeValidation, "Email address must be valid."), accountingsync.SyncErrorValidation},
		{"local", xero.ErrLinesRequired, accountingsync.SyncErrorValidation},
		{"bad date", errInvalidDate, accountingsync.SyncErrorValidation},
		{"unknown", errors.New("strange"), accountingsync.SyncErrorTransient},
	}
	for _, tc := range cases {
		classified := conn.ClassifyDocumentError(tc.err)
		require.NotNil(t, classified, tc.name)
		assert.Equal(t, tc.want, classified.Category, tc.name)
	}
	assert.Nil(t, conn.ClassifyDocumentError(nil))

	validation := conn.ClassifyDocumentError(apiError(400, xero.TypeValidation, "A", "B"))
	assert.Equal(t, "A; B", validation.Message)
	assert.Equal(t, xero.TypeValidation, validation.Code)

	offline := conn.ClassifyDocumentError(apiError(503, ""))
	assert.Equal(t, "organisation-offline", offline.Code)
	assert.Contains(t, offline.Message, "5m0s")
}

func TestDailyRateLimitCarriesTheRetryDelay(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	daily := conn.ClassifyDocumentError(&xero.APIError{
		Status:           429,
		RateLimitProblem: xero.RateLimitDay,
		RetryAfter:       3 * time.Hour,
	})
	assert.Equal(t, accountingsync.SyncErrorRateLimited, daily.Category)
	assert.Equal(t, xero.RateLimitDay, daily.Code)
	assert.Contains(t, daily.Message, "3h0m0s")

	minute := conn.ClassifyDocumentError(&xero.APIError{
		Status:           429,
		RateLimitProblem: xero.RateLimitMinute,
	})
	assert.Equal(t, accountingsync.SyncErrorRateLimited, minute.Category)
	assert.NotEqual(t, xero.RateLimitDay, minute.Code)
}

func TestUpsertCustomerCreatesThenUpdates(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path == "/Contacts" || call.Path == "/Contacts/"+testCustomer {
			return http.StatusOK, `{"Contacts":[{"ContactID":"` + testCustomer + `","Name":"Globex"}]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	doc := &services.AccountingCustomerDocument{
		Auth: testAuth(), RequestID: testRequestID,
		Party: services.AccountingPartyDraft{DisplayName: "Globex"},
	}
	created, err := conn.UpsertCustomer(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testCustomer, created.ExternalID)

	doc.ExternalID = testCustomer
	_, err = conn.UpsertCustomer(t.Context(), doc)
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, http.MethodPut, writes[0].Method)
	assert.Equal(t, http.MethodPost, writes[1].Method)
	assert.Equal(t, "/Contacts/"+testCustomer, writes[1].Path)
}
