package bcconnector

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	customerJournals = "/customerPaymentJournals"
	vendorJournals   = "/vendorPaymentJournals"
)

func journalJSON(code string) string {
	return `{"id":"` + testJournal + `","code":"` + code + `","displayName":"Trenova receipts",` +
		`"balancingAccountId":"` + testBank + `"}`
}

type journalServer struct {
	t        *testing.T
	journals string
	payments string
	exists   bool
	lines    string
	ledger   string
	postCode int
	racing   bool
	mu       sync.Mutex
	events   []string
	delay    time.Duration
}

func (s *journalServer) record(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *journalServer) respond(call bcCall) (int, string) {
	linesPath := entity(strings.TrimPrefix(s.journals, "/"), testJournal) + s.payments
	switch {
	case call.is(http.MethodGet, s.journals):
		s.mu.Lock()
		exists := s.exists
		s.mu.Unlock()
		if exists {
			return http.StatusOK, `{"value":[` + journalJSON(batchCode(testBank)) + `]}`
		}
		return http.StatusOK, `{"value":[]}`
	case call.is(http.MethodPost, s.journals):
		if s.racing {
			s.mu.Lock()
			s.exists = true
			s.mu.Unlock()
			return http.StatusBadRequest, readFixture(s.t, "error_duplicate.json")
		}
		return http.StatusCreated, journalJSON(batchCode(testBank))
	case call.is(http.MethodGet, "/generalLedgerEntries"):
		if s.ledger != "" {
			return http.StatusOK, s.ledger
		}
		return http.StatusOK, `{"value":[]}`
	case call.is(http.MethodGet, linesPath):
		if s.lines != "" {
			return http.StatusOK, s.lines
		}
		return http.StatusOK, `{"value":[]}`
	case call.is(http.MethodPost, linesPath):
		number, _ := call.Body["documentNumber"].(string)
		s.record("line:" + number)
		time.Sleep(s.delay)
		return http.StatusCreated, `{"id":"ffffffff-0000-4111-8222-333333333332","journalId":"` +
			testJournal + `","documentNumber":"` + number + `"}`
	case call.is(http.MethodPost, entity("journals", testJournal)+postAction):
		s.record("post")
		if s.postCode != 0 {
			return s.postCode, readFixture(s.t, "error_404.json")
		}
		return http.StatusNoContent, ""
	}
	return 0, ""
}

func newCustomerJournals(t *testing.T) *journalServer {
	return &journalServer{t: t, journals: customerJournals, payments: "/customerPayments"}
}

func paymentDoc() *services.AccountingPaymentDocument {
	return &services.AccountingPaymentDocument{
		Auth:                     testAuth(),
		RequestID:                testRequestID,
		CustomerExternalID:       testCustomer,
		TxnDate:                  "2026-10-01",
		CurrencyCode:             "USD",
		DepositAccountExternalID: testBank,
		ReferenceNumber:          "CHK 4411",
		TotalAmount:              decimal.NewFromInt(150),
		Applications: []services.AccountingPaymentApplication{
			{InvoiceExternalID: testInvoice, InvoiceNumber: "INV-1", AppliedAmount: decimal.NewFromInt(100)},
			{InvoiceExternalID: testNewInvoice, InvoiceNumber: "INV-2", AppliedAmount: decimal.NewFromInt(50)},
		},
	}
}

func TestSavePaymentCreatesTheBatchAddsALinePerInvoiceAndPosts(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), paymentDoc())
	require.NoError(t, err)
	number := requestReference(testRequestID)
	assert.Len(t, number, 20)
	assert.Equal(t, number, result.ExternalID)
	assert.Equal(t, number, result.DocNumber)
	assert.Equal(t, testJournal, result.Refs[refJournal])
	assert.Equal(t, docTypeCustomerPayment, result.Refs[accountingsync.ExternalRefDocumentType])

	code := batchCode(testBank)
	assert.Len(t, code, 10)
	assert.True(t, strings.HasPrefix(code, "TRN"))
	lookup, ok := fake.find(http.MethodGet, customerJournals)
	require.True(t, ok)
	assert.Equal(t, "code eq '"+code+"'", lookup.filter())
	created, ok := fake.find(http.MethodPost, customerJournals)
	require.True(t, ok)
	assert.Equal(t, code, created.Body["code"])
	assert.Equal(t, testBank, created.Body["balancingAccountId"])

	linesPath := entity("customerPaymentJournals", testJournal) + "/customerPayments"
	var sent []bcCall
	for _, call := range fake.writes() {
		if call.Path == linesPath {
			sent = append(sent, call)
		}
	}
	require.Len(t, sent, 2)
	assert.Equal(t, testInvoice, sent[0].Body["appliesToInvoiceId"])
	assert.InDelta(t, -100, sent[0].Body["amount"], 0)
	assert.Equal(t, number, sent[0].Body["documentNumber"])
	assert.Equal(t, "CHK 4411", sent[0].Body["externalDocumentNumber"])
	assert.Equal(t, testCustomer, sent[0].Body["customerId"])
	assert.Equal(t, "2026-10-01", sent[0].Body["postingDate"])
	assert.InDelta(t, -50, sent[1].Body["amount"], 0)
	assert.Equal(t, []string{"line:" + number, "line:" + number, "post"}, server.events)
}

func TestSavePaymentResumesUnpostedLinesInsteadOfAddingMore(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.exists = true
	server.lines = `{"value":[{"id":"ffffffff-0000-4111-8222-333333333331","journalId":"` +
		testJournal + `","documentNumber":"` + requestReference(testRequestID) +
		`","appliesToInvoiceId":"` + strings.ToUpper(testInvoice) + `","amount":-100}]}`
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	_, err := conn.SavePayment(t.Context(), paymentDoc())
	require.NoError(t, err)
	_, created := fake.find(http.MethodPost, customerJournals)
	assert.False(t, created)
	number := requestReference(testRequestID)
	assert.Equal(t, []string{"line:" + number, "post"}, server.events)
	lookup, ok := fake.find(http.MethodGet,
		entity("customerPaymentJournals", testJournal)+"/customerPayments")
	require.True(t, ok)
	assert.Equal(t, "documentNumber eq '"+number+"'", lookup.filter())
}

func TestSavePaymentAlreadyPostedWritesNothingMore(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.exists = true
	server.ledger = readFixture(t, "general_ledger_entries.json")
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), paymentDoc())
	require.NoError(t, err)
	assert.Equal(t, requestReference(testRequestID), result.ExternalID)
	assert.Empty(t, fake.writes())
}

func TestSavePaymentWhenTheBatchCannotBePostedBlocksAsConfiguration(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.exists = true
	server.postCode = http.StatusNotFound
	conn := testConnector(t, &fakeBC{respond: server.respond})

	result, err := conn.SavePayment(t.Context(), paymentDoc())
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, classified.Category)
	assert.Equal(t, "journal-post", classified.Code)
	assert.Contains(t, classified.Resolution, "Post the "+batchCode(testBank)+" payment journal")
	assert.Equal(t, testJournal, result.Refs[refJournal])
}

func TestSavePaymentHoldsTheBatchLockAcrossLinesAndPost(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.exists = true
	server.delay = 5 * time.Millisecond
	conn := testConnector(t, &fakeBC{respond: server.respond})

	first := paymentDoc()
	second := paymentDoc()
	second.RequestID = testRequestID + "-2"
	var wg sync.WaitGroup
	for _, doc := range []*services.AccountingPaymentDocument{first, second} {
		wg.Go(func() {
			_, err := conn.SavePayment(t.Context(), doc)
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	require.Len(t, server.events, 6)
	for start := 0; start < len(server.events); start += 3 {
		group := server.events[start : start+3]
		assert.Equal(t, group[0], group[1])
		assert.Equal(t, "post", group[2])
	}
}

func TestSavePaymentRefusalsAndValidation(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)

	changed := paymentDoc()
	changed.ExternalID = "T123"
	_, err := conn.SavePayment(t.Context(), changed)
	refused := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, refused.Category)
	assert.Contains(t, refused.Resolution, "Reverse transaction")

	unapplied := paymentDoc()
	unapplied.TotalAmount = decimal.NewFromInt(200)
	_, err = conn.SavePayment(t.Context(), unapplied)
	assert.Equal(t, accountingsync.SyncErrorValidation, syncErrorOf(t, err).Category)

	nothing := paymentDoc()
	nothing.Applications = nil
	_, err = conn.SavePayment(t.Context(), nothing)
	assert.Equal(t, accountingsync.SyncErrorValidation, syncErrorOf(t, err).Category)

	noDeposit := paymentDoc()
	noDeposit.DepositAccountExternalID = ""
	_, err = conn.SavePayment(t.Context(), noDeposit)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)

	closed := paymentDoc()
	closed.TxnDate = "2025-06-01"
	_, err = conn.SavePayment(t.Context(), closed)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, syncErrorOf(t, err).Category)

	ref := &services.AccountingDocumentRef{Auth: testAuth(), ExternalID: "T123"}
	_, err = conn.VoidPayment(t.Context(), ref)
	assert.Equal(t, "reverse-transaction", syncErrorOf(t, err).Code)
	_, err = conn.UpdateBillPayment(t.Context(), &services.AccountingBillPaymentDocument{})
	assert.Equal(t, "reverse-transaction", syncErrorOf(t, err).Code)
	_, err = conn.CreateCreditApplication(t.Context(), &services.AccountingCreditApplicationDocument{})
	creditErr := syncErrorOf(t, err)
	assert.Equal(t, accountingsync.SyncErrorConfiguration, creditErr.Category)
	assert.Contains(t, creditErr.Resolution, "Apply the credit memo in Business Central")
	_, err = conn.VoidCreditApplication(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), Refs: map[string]string{},
	})
	assert.Equal(t, "credit-application", syncErrorOf(t, err).Code)
	assert.Empty(t, fake.writes())
}

func TestSavePaymentWritesAShortPayCreditMemoAppliedToTheInvoice(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.exists = true
	memos := newSalesServer(t)
	memos.entity = "salesCreditMemos"
	memos.newID = testCreditMemo
	memos.posted = docSpec{id: testCreditMemo, number: "S-CR-9", status: "Open", total: 5}
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if code, body := server.respond(call); code != 0 {
			return code, body
		}
		return memos.respond(call)
	}}
	conn := testConnector(t, fake)

	doc := paymentDoc()
	doc.Applications[0].ShortPayAmount = decimal.NewFromInt(5)
	doc.ShortPayItemExternalID = testWriteOffItem
	result, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testCreditMemo,
		result.Refs[accountingsync.ExternalRefShortPayPrefix+testInvoice])

	create, ok := fake.find(http.MethodPost, "/salesCreditMemos")
	require.True(t, ok)
	assert.Equal(t, testInvoice, create.Body["invoiceId"])
	assert.Equal(t, testCustomer, create.Body["customerId"])
	sent := lines(t, create.Body, "salesCreditMemoLines")
	require.Len(t, sent, 1)
	assert.Equal(t, testWriteOffItem, sent[0]["itemId"])
	assert.InDelta(t, 5, sent[0]["unitPrice"], 0)
	assert.Equal(t, "Short pay on INV-1", sent[0]["description"])
	_, posted := fake.find(http.MethodPost, entity("salesCreditMemos", testCreditMemo)+postAction)
	assert.True(t, posted)

	held := paymentDoc()
	held.Applications[0].ShortPayAmount = decimal.NewFromInt(5)
	held.Applications[0].ShortPayCreditExternalID = testCreditMemo
	again := &fakeBC{respond: server.respond}
	_, err = testConnector(t, again).SavePayment(t.Context(), held)
	require.NoError(t, err)
	_, wrote := again.find(http.MethodPost, "/salesCreditMemos")
	assert.False(t, wrote)

	unmapped := paymentDoc()
	unmapped.Applications[0].ShortPayAmount = decimal.NewFromInt(5)
	_, err = conn.SavePayment(t.Context(), unmapped)
	assert.Equal(t, accountingsync.ItemRoleShortPayWriteOff, syncErrorOf(t, err).Code)
}

func TestVoidCreditApplicationCancelsAMemoCreatedApplied(t *testing.T) {
	t.Parallel()

	memos := newSalesServer(t)
	memos.entity = "salesCreditMemos"
	memos.existing = map[string]docSpec{
		testCreditMemo: {id: testCreditMemo, status: "Open", total: 10},
	}
	fake := &fakeBC{respond: memos.respond}
	conn := testConnector(t, fake)

	result, err := conn.VoidCreditApplication(t.Context(), &services.AccountingDocumentRef{
		Auth:       testAuth(),
		Kind:       accountingsync.SyncObjectCreditApplication,
		ExternalID: "app-1",
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:    testCreditMemo,
			accountingsync.ExternalRefApplication: testInvoice,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "app-1", result.ExternalID)
	_, cancelled := fake.find(http.MethodPost, entity("salesCreditMemos", testCreditMemo)+cancelAction)
	assert.True(t, cancelled)
}

func TestCreateBillPaymentPostsAVendorPaymentAppliedToTheBill(t *testing.T) {
	t.Parallel()

	server := &journalServer{t: t, journals: vendorJournals, payments: "/vendorPayments"}
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := &services.AccountingBillPaymentDocument{
		Auth:                  testAuth(),
		RequestID:             testRequestID,
		Kind:                  accountingsync.SyncObjectCarrierBillPay,
		VendorExternalID:      testVendor,
		BankAccountExternalID: testBank,
		BillExternalID:        testBill,
		DocNumber:             "ACH-77",
		TxnDate:               "2026-10-02",
		PrivateNote:           "Settlement",
		Amount:                decimal.NewFromInt(900),
	}
	result, err := conn.CreateBillPayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, requestReference(testRequestID), result.ExternalID)
	assert.Equal(t, docTypeVendorPayment, result.Refs[accountingsync.ExternalRefDocumentType])

	created, ok := fake.find(http.MethodPost, vendorJournals)
	require.True(t, ok)
	assert.Equal(t, "Trenova payments "+batchCode(testBank), created.Body["displayName"])
	line, ok := fake.find(http.MethodPost, entity("vendorPaymentJournals", testJournal)+"/vendorPayments")
	require.True(t, ok)
	assert.InDelta(t, 900, line.Body["amount"], 0)
	assert.Equal(t, testBill, line.Body["appliesToInvoiceId"])
	assert.Equal(t, testVendor, line.Body["vendorId"])
	assert.Equal(t, "ACH-77", line.Body["externalDocumentNumber"])
	assert.Equal(t, []string{"line:" + requestReference(testRequestID), "post"}, server.events)

	doc.Kind = accountingsync.SyncObjectCarrierBill
	_, err = conn.CreateBillPayment(t.Context(), doc)
	require.ErrorIs(t, err, errDocumentKind)
	doc.Kind = accountingsync.SyncObjectDriverBillPay
	doc.BankAccountExternalID = ""
	_, err = conn.CreateBillPayment(t.Context(), doc)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)
}

func TestSavePaymentFindsABatchCreatedConcurrently(t *testing.T) {
	t.Parallel()

	server := newCustomerJournals(t)
	server.racing = true
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), paymentDoc())
	require.NoError(t, err)
	assert.Equal(t, testJournal, result.Refs[refJournal])
	assert.Equal(t, 2, fake.count(http.MethodGet, customerJournals))
	assert.Equal(t, 1, fake.count(http.MethodPost, customerJournals))
}
