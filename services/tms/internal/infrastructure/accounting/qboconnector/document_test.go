package qboconnector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const docRequestID = "trn-0123456789abcdef0123456789abcdef01234567"

type recordedCall struct {
	Method    string
	Path      string
	RequestID string
	Operation string
	Body      map[string]any
}

type fakeQBO struct {
	mu      sync.Mutex
	calls   []recordedCall
	respond func(call recordedCall) (int, string)
}

func (f *fakeQBO) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		call := recordedCall{
			Method:    r.Method,
			Path:      strings.TrimPrefix(r.URL.Path, "/v3/company/123"),
			RequestID: r.URL.Query().Get("requestid"),
			Operation: r.URL.Query().Get("operation"),
		}
		if r.Method == http.MethodPost {
			raw, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.NoError(t, sonic.Unmarshal(raw, &call.Body))
		}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()
		status, body := f.respond(call)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func (f *fakeQBO) writes() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedCall, 0, len(f.calls))
	for _, call := range f.calls {
		if call.Method == http.MethodPost {
			out = append(out, call)
		}
	}
	return out
}

func documentConnector(t *testing.T, fake *fakeQBO) *Connector {
	t.Helper()
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)
	conn := newConnector(t, config.QuickBooksConfig{ClientID: "id", ClientSecret: "secret"})
	conn.apiOpts = []quickbooks.Option{
		quickbooks.WithBaseURL(server.URL),
		quickbooks.WithRetry(restx.RetryConfig{}),
	}
	return conn
}

func auth() services.AccountingDocumentAuth {
	return services.AccountingDocumentAuth{RealmID: "123", AccessToken: "token"}
}

func shortPaidPayment() *services.AccountingPaymentDocument {
	return &services.AccountingPaymentDocument{
		Auth:                     auth(),
		RequestID:                docRequestID,
		CustomerExternalID:       "58",
		TxnDate:                  "2026-09-22",
		PaymentMethodExternalID:  "4",
		DepositAccountExternalID: "35",
		ReferenceNumber:          "ACH-1",
		TotalAmount:              decimal.NewFromInt(975),
		ShortPayItemExternalID:   "77",
		Applications: []services.AccountingPaymentApplication{
			{
				InvoiceExternalID: "145",
				InvoiceNumber:     "INV-1001",
				AppliedAmount:     decimal.NewFromInt(975),
				ShortPayAmount:    decimal.NewFromInt(25),
			},
		},
	}
}

func TestSavePaymentWritesTheShortPayCreditThenLinksItInThePayment(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Path == "/creditmemo" {
			return http.StatusOK, `{"CreditMemo":{"Id":"146","SyncToken":"0"}}`
		}
		return http.StatusOK, `{"Payment":{"Id":"147","SyncToken":"0"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), shortPaidPayment())
	require.NoError(t, err)
	assert.Equal(t, "147", result.ExternalID)
	assert.Equal(t, map[string]string{
		accountingsync.ExternalRefDocument:              "147",
		accountingsync.ExternalRefShortPayPrefix + "145": "146",
	}, result.Refs)

	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, "/creditmemo", writes[0].Path)
	assert.Equal(t, docRequestID+"-1", writes[0].RequestID)
	assert.Equal(t, "/payment", writes[1].Path)
	assert.Equal(t, docRequestID, writes[1].RequestID)
	lines := writes[1].Body["Line"].([]any)
	require.Len(t, lines, 2)
	assert.InDelta(t, 1000, lines[0].(map[string]any)["Amount"], 0.001, "the invoice is settled in full")
	assert.InDelta(t, 25, lines[1].(map[string]any)["Amount"], 0.001)
	assert.InDelta(t, 975, writes[1].Body["TotalAmt"], 0.001)
}

func TestSavePaymentReplayReusesTheCreditItAlreadyWrote(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Path == "/creditmemo" {
			return http.StatusOK, `{"CreditMemo":{"Id":"146"}}`
		}
		return http.StatusServiceUnavailable, `{"Fault":{"type":"SystemFault","Error":[{"Message":"down","code":"10000"}]}}`
	}}
	conn := documentConnector(t, fake)

	first, err := conn.SavePayment(t.Context(), shortPaidPayment())
	require.Error(t, err)
	require.NotNil(t, first, "a partial write reports what it created")
	assert.Equal(t, "146", first.Refs[accountingsync.ExternalRefShortPayPrefix+"145"])
	assert.Equal(t, accountingsync.SyncErrorTransient, conn.ClassifyDocumentError(err).Category)

	fake.respond = func(recordedCall) (int, string) {
		return http.StatusOK, `{"Payment":{"Id":"147"}}`
	}
	retry := shortPaidPayment()
	retry.Refs = first.Refs
	second, err := conn.SavePayment(t.Context(), retry)
	require.NoError(t, err)
	assert.Equal(t, "147", second.ExternalID)

	creditWrites := 0
	for _, call := range fake.writes() {
		if call.Path == "/creditmemo" {
			creditWrites++
		}
	}
	assert.Equal(t, 1, creditWrites, "the retry replays nothing already done")
}

func TestSavePaymentBlocksWhenTheShortPayItemIsUnmapped(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		t.Fatal("nothing should be written")
		return 0, ""
	}}
	conn := documentConnector(t, fake)

	doc := shortPaidPayment()
	doc.ShortPayItemExternalID = ""
	_, err := conn.SavePayment(t.Context(), doc)
	classified := conn.ClassifyDocumentError(err)
	require.NotNil(t, classified)
	assert.Equal(t, accountingsync.SyncErrorMapping, classified.Category)
	assert.Contains(t, classified.Resolution, "QuickBooks Online item")
}

func TestSavePaymentUpdatesAnExistingPayment(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{"Payment":{"Id":"147","SyncToken":"2"}}`
		}
		return http.StatusOK, `{"Payment":{"Id":"147","SyncToken":"3"}}`
	}}
	conn := documentConnector(t, fake)

	doc := shortPaidPayment()
	doc.ExternalID = "147"
	doc.Applications[0].ShortPayCreditExternalID = "146"
	_, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)

	writes := fake.writes()
	require.Len(t, writes, 1, "a credit written by an earlier record is linked, not written again")
	assert.Equal(t, "147", writes[0].Body["Id"])
	assert.Equal(t, "2", writes[0].Body["SyncToken"])
}

func TestAdjustmentCreditMemoIsAppliedToItsInvoice(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Path == "/creditmemo" {
			return http.StatusOK, `{"CreditMemo":{"Id":"146","DocNumber":"CM-1"}}`
		}
		return http.StatusOK, `{"Payment":{"Id":"148"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.CreateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Auth:               auth(),
		RequestID:          docRequestID,
		Kind:               accountingsync.SyncObjectCreditMemo,
		CustomerExternalID: "58",
		DocNumber:          "CM-1",
		TxnDate:            "2026-09-22",
		DueDate:            "2026-10-22",
		TermExternalID:     "3",
		Lines: []services.AccountingDocumentLine{{
			ItemExternalID: "21",
			Amount:         decimal.NewFromInt(400),
		}},
		ApplyToExternalID: "145",
		ApplyAmount:       decimal.NewFromInt(400),
	})
	require.NoError(t, err)
	assert.Equal(t, "146", result.ExternalID)
	assert.Equal(t, "148", result.Refs[accountingsync.ExternalRefApplication])

	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.NotContains(t, writes[0].Body, "DueDate", "a credit memo has no terms")
	assert.NotContains(t, writes[0].Body, "SalesTermRef")
	assert.Equal(t, docRequestID+"-1", writes[1].RequestID)
	assert.InDelta(t, 0, writes[1].Body["TotalAmt"], 0.001)
}

func TestDebitMemoIsSentAsAnInvoiceThatSaysSo(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"Invoice":{"Id":"149","DocNumber":"DM-1"}}`
	}}
	conn := documentConnector(t, fake)

	_, err := conn.CreateSalesDocument(t.Context(), &services.AccountingSalesDocument{
		Auth:               auth(),
		RequestID:          docRequestID,
		Kind:               accountingsync.SyncObjectDebitMemo,
		CustomerExternalID: "58",
		DocNumber:          "DM-1",
		PrivateNote:        "Rebills INV-1001",
		Lines: []services.AccountingDocumentLine{{
			ItemExternalID: "21",
			Amount:         decimal.NewFromInt(80),
		}},
	})
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/invoice", writes[0].Path)
	assert.Equal(t, "Debit memo. Rebills INV-1001", writes[0].Body["PrivateNote"])
}

func TestVoidPaymentAlsoRemovesItsShortPayCredits(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		switch {
		case call.Path == "/payment/147":
			return http.StatusOK, `{"Payment":{"Id":"147","SyncToken":"3"}}`
		case call.Path == "/creditmemo/146":
			return http.StatusBadRequest, `{"Fault":{"type":"ValidationFault","Error":[{"Message":"Object Not Found","code":"610"}]}}`
		default:
			return http.StatusOK, `{"Payment":{"Id":"147"}}`
		}
	}}
	conn := documentConnector(t, fake)

	_, err := conn.VoidPayment(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectCustomerPayment,
		ExternalID: "147",
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:              "147",
			accountingsync.ExternalRefShortPayPrefix + "145": "146",
		},
	})
	require.NoError(t, err, "a credit already gone counts as removed")

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "update", writes[0].Operation)
}

func TestClassifyDocumentErrorNamesTheCategory(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	fault := func(status int, code string) error {
		return &quickbooks.FaultError{
			StatusCode: status,
			Type:       "ValidationFault",
			Errors:     []quickbooks.FaultDetail{{Message: "m", Detail: "detail " + code, Code: code}},
		}
	}

	tests := []struct {
		err      error
		category accountingsync.SyncErrorCategory
	}{
		{err: ErrNotConfigured, category: accountingsync.SyncErrorConfiguration},
		{err: fault(http.StatusUnauthorized, "3200"), category: accountingsync.SyncErrorAuth},
		{err: fault(http.StatusTooManyRequests, "003001"), category: accountingsync.SyncErrorRateLimited},
		{err: fault(http.StatusBadRequest, "5010"), category: accountingsync.SyncErrorTransient},
		{err: fault(http.StatusBadRequest, "6140"), category: accountingsync.SyncErrorDuplicate},
		{err: fault(http.StatusBadRequest, "6240"), category: accountingsync.SyncErrorDuplicate},
		{err: fault(http.StatusBadRequest, "6210"), category: accountingsync.SyncErrorClosedPeriod},
		{err: fault(http.StatusBadRequest, "610"), category: accountingsync.SyncErrorNotFound},
		{err: fault(http.StatusBadRequest, "2500"), category: accountingsync.SyncErrorMapping},
		{err: fault(http.StatusBadRequest, "6000"), category: accountingsync.SyncErrorValidation},
		{err: quickbooks.ErrDocNumberTooLong, category: accountingsync.SyncErrorValidation},
		{err: fault(http.StatusInternalServerError, "10000"), category: accountingsync.SyncErrorTransient},
	}
	for _, tt := range tests {
		classified := conn.ClassifyDocumentError(tt.err)
		require.NotNil(t, classified, tt.err)
		assert.Equal(t, tt.category, classified.Category, tt.err)
	}

	duplicate := conn.ClassifyDocumentError(fault(http.StatusBadRequest, "6140"))
	assert.Equal(t, "6140", duplicate.Code)
	assert.Equal(t, "detail 6140", duplicate.Message)
	assert.NotEmpty(t, duplicate.Resolution)
	assert.Nil(t, conn.ClassifyDocumentError(nil))
}

func TestDocumentURLPointsAtTheRightScreen(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{Environment: "sandbox"})
	assert.Equal(t,
		"https://app.sandbox.qbo.intuit.com/app/invoice?txnId=145",
		conn.DocumentURL(accountingsync.SyncObjectDebitMemo, "145"),
	)
	assert.Equal(t,
		"https://app.sandbox.qbo.intuit.com/app/recvpayment?txnId=148",
		conn.DocumentURL(accountingsync.SyncObjectCreditApplication, "148"),
	)
	assert.Equal(t,
		"https://app.sandbox.qbo.intuit.com/app/customerdetail?nameId=58",
		conn.DocumentURL(accountingsync.SyncObjectCustomer, "58"),
	)
	assert.Empty(t, conn.DocumentURL(accountingsync.SyncObjectInvoice, " "))
	assert.Equal(t, quickbooks.MaxDocNumberLength, conn.DocumentLimits().MaxDocNumberLength)
}
