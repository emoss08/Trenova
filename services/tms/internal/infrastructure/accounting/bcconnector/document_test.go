package bcconnector

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	postAction   = "/Microsoft.NAV.post"
	cancelAction = "/Microsoft.NAV.cancel"
)

func entity(name, id string) string {
	return "/" + name + "(" + id + ")"
}

func salesDoc() *services.AccountingSalesDocument {
	return &services.AccountingSalesDocument{
		Auth:               testAuth(),
		RequestID:          testRequestID,
		Kind:               accountingsync.SyncObjectInvoice,
		CustomerExternalID: testCustomer,
		DocNumber:          "TRN-1003",
		TxnDate:            "2026-10-01",
		DueDate:            "2026-10-31",
		TermExternalID:     testTerm,
		CurrencyCode:       "USD",
		Lines: []services.AccountingDocumentLine{
			{
				Description:    "Linehaul",
				ItemExternalID: testItem,
				Quantity:       decimal.NewFromInt(2),
				UnitPrice:      decimal.RequireFromString("25.5"),
				Amount:         decimal.NewFromInt(51),
			},
			{
				ItemExternalID: testItem,
				Quantity:       decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromInt(1),
				Amount:         decimal.NewFromInt(10),
			},
		},
	}
}

type salesServer struct {
	t        *testing.T
	entity   string
	newID    string
	listed   string
	posted   docSpec
	existing map[string]docSpec
	extra    func(call bcCall) (int, string)
}

func (s *salesServer) respond(call bcCall) (int, string) {
	if s.extra != nil {
		if code, body := s.extra(call); code != 0 {
			return code, body
		}
	}
	switch {
	case call.is(http.MethodGet, "/"+s.entity):
		if s.listed == "" {
			return http.StatusOK, `{"value":[]}`
		}
		return http.StatusOK, s.listed
	case call.is(http.MethodPost, "/"+s.entity):
		return http.StatusCreated, docSpec{
			id: s.newID, number: "D-1", status: "Draft", reference: "TRN-1003",
		}.json(s.t)
	case call.is(http.MethodPost, entity(s.entity, s.newID)+postAction):
		return http.StatusNoContent, ""
	case call.is(http.MethodGet, entity(s.entity, s.newID)):
		return http.StatusOK, s.posted.json(s.t)
	}
	for id, doc := range s.existing {
		if call.is(http.MethodGet, entity(s.entity, id)) {
			return http.StatusOK, doc.json(s.t)
		}
		if call.is(http.MethodPost, entity(s.entity, id)+cancelAction) {
			return http.StatusNoContent, ""
		}
		if call.is(http.MethodDelete, entity(s.entity, id)) {
			return http.StatusNoContent, ""
		}
	}
	if call.is(http.MethodGet, "/salesCreditMemos") {
		return http.StatusOK, `{"value":[]}`
	}
	return 0, ""
}

func newSalesServer(t *testing.T) *salesServer {
	return &salesServer{
		t:      t,
		entity: "salesInvoices",
		newID:  testNewInvoice,
		posted: docSpec{
			id: testNewInvoice, number: "PS-INV-1003", status: "Open", reference: "TRN-1003",
			total: 61, remaining: 61,
		},
	}
}

func TestDocumentLimits(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	assert.Equal(t, services.AccountingDocumentLimits{
		MaxDocNumberLength:      35,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       30 * 24 * time.Hour,
	}, conn.DocumentLimits())
}

func TestDocumentURLLinksThePostedPageByNumber(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	auth := testAuth()
	link := conn.DocumentURL(auth, services.AccountingDocumentLink{
		Kind:       accountingsync.SyncObjectInvoice,
		ExternalID: testInvoice,
		DocNumber:  "PS-INV103001",
	})
	assert.Equal(t, testWebURL+"/"+testTenant+"/Production/?company=CRONUS+USA%2C+Inc."+
		"&page=132&filter=%27No.%27+IS+%27PS-INV103001%27", link)

	for kind, page := range map[accountingsync.SyncObjectType]string{
		accountingsync.SyncObjectDebitMemo:   "page=132",
		accountingsync.SyncObjectCreditMemo:  "page=134",
		accountingsync.SyncObjectCarrierBill: "page=138",
		accountingsync.SyncObjectDriverBill:  "page=138",
	} {
		got := conn.DocumentURL(auth, services.AccountingDocumentLink{Kind: kind, DocNumber: "N-1"})
		assert.Contains(t, got, page, kind)
	}

	assert.Empty(t, conn.DocumentURL(auth, services.AccountingDocumentLink{
		Kind: accountingsync.SyncObjectCustomerPayment, DocNumber: "T1",
	}))
	assert.Empty(t, conn.DocumentURL(auth, services.AccountingDocumentLink{
		Kind: accountingsync.SyncObjectInvoice, ExternalID: testInvoice,
	}))
	noName := auth
	noName.CompanyName = ""
	assert.Empty(t, conn.DocumentURL(noName, services.AccountingDocumentLink{
		Kind: accountingsync.SyncObjectInvoice, DocNumber: "N-1",
	}))
	badRealm := auth
	badRealm.RealmID = "realm"
	assert.Empty(t, conn.DocumentURL(badRealm, services.AccountingDocumentLink{
		Kind: accountingsync.SyncObjectInvoice, DocNumber: "N-1",
	}))
}

func TestCreateSalesInvoiceCreatesPostsAndRereadsTheNumber(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.CreateSalesDocument(t.Context(), salesDoc())
	require.NoError(t, err)
	assert.Equal(t, testNewInvoice, result.ExternalID)
	assert.Equal(t, "PS-INV-1003", result.DocNumber)
	assert.Equal(t, testNewInvoice, result.Refs[accountingsync.ExternalRefDocument])

	create, ok := fake.find(http.MethodPost, "/salesInvoices")
	require.True(t, ok)
	assert.Equal(t, "TRN-1003", create.Body["externalDocumentNumber"])
	assert.Equal(t, "2026-10-01", create.Body["invoiceDate"])
	assert.Equal(t, "2026-10-01", create.Body["postingDate"])
	assert.Equal(t, "2026-10-31", create.Body["dueDate"])
	assert.Equal(t, testCustomer, create.Body["customerId"])
	assert.Equal(t, testTerm, create.Body["paymentTermsId"])
	assert.NotContains(t, create.Body, "currencyCode")
	sent := lines(t, create.Body, "salesInvoiceLines")
	require.Len(t, sent, 2)
	assert.Equal(t, "Item", sent[0]["lineType"])
	assert.Equal(t, testItem, sent[0]["itemId"])
	assert.InDelta(t, 2, sent[0]["quantity"], 0)
	assert.InDelta(t, 25.5, sent[0]["unitPrice"], 0)
	assert.Equal(t, "Charges", sent[1]["description"])
	assert.InDelta(t, 1, sent[1]["quantity"], 0)
	assert.InDelta(t, 10, sent[1]["unitPrice"], 0)

	_, posted := fake.find(http.MethodPost, entity("salesInvoices", testNewInvoice)+postAction)
	assert.True(t, posted)
	search, ok := fake.find(http.MethodGet, "/salesInvoices")
	require.True(t, ok)
	assert.Equal(t, "externalDocumentNumber eq 'TRN-1003' and customerId eq "+testCustomer,
		search.filter())
}

func TestCreateSalesInvoiceResumesADraftWithTheSameReference(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	server.listed = listJSON(t,
		docSpec{id: testInvoice, number: "PS-1", status: "Canceled", reference: "TRN-1003"},
		docSpec{id: testNewInvoice, number: "D-1", status: "Draft", reference: "TRN-1003"},
	)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	result, err := conn.CreateSalesDocument(t.Context(), salesDoc())
	require.NoError(t, err)
	assert.Equal(t, testNewInvoice, result.ExternalID)
	assert.Equal(t, "PS-INV-1003", result.DocNumber)
	assert.Zero(t, fake.count(http.MethodPost, "/salesInvoices"))
	assert.Equal(t, 1, fake.count(http.MethodPost, entity("salesInvoices", testNewInvoice)+postAction))
}

func TestCreateSalesInvoiceResumesTheDocumentItsRefsName(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := salesDoc()
	doc.Refs = map[string]string{accountingsync.ExternalRefDocument: testNewInvoice}
	result, err := conn.CreateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, "PS-INV-1003", result.DocNumber)
	assert.Empty(t, fake.writes())
}

func TestCreateSalesInvoiceKeepsTheDraftWhenPostingFails(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	server.extra = func(call bcCall) (int, string) {
		if call.is(http.MethodPost, entity("salesInvoices", testNewInvoice)+postAction) {
			return http.StatusBadRequest, readFixture(t, "error_posting_date.json")
		}
		return 0, ""
	}
	conn := testConnector(t, &fakeBC{respond: server.respond})

	result, err := conn.CreateSalesDocument(t.Context(), salesDoc())
	require.Error(t, err)
	assert.Equal(t, testNewInvoice, result.Refs[accountingsync.ExternalRefDocument])
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, classified.Category)
	assert.Equal(t, businesscentral.CodeDialogException, classified.Code)
}

func TestCreateSalesDocumentValidatesBeforeWriting(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)

	closed := salesDoc()
	closed.TxnDate = "2025-12-15"
	_, err := conn.CreateSalesDocument(t.Context(), closed)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, syncErrorOf(t, err).Category)

	unmapped := salesDoc()
	unmapped.Lines[1].ItemExternalID = ""
	_, err = conn.CreateSalesDocument(t.Context(), unmapped)
	assert.Equal(t, accountingsync.SyncErrorMapping, syncErrorOf(t, err).Category)

	negative := salesDoc()
	negative.Lines[0].Amount = decimal.NewFromInt(-5)
	_, err = conn.CreateSalesDocument(t.Context(), negative)
	assert.Equal(t, accountingsync.SyncErrorValidation, syncErrorOf(t, err).Category)

	empty := salesDoc()
	empty.Lines = nil
	_, err = conn.CreateSalesDocument(t.Context(), empty)
	require.ErrorIs(t, err, errLinesRequired)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)

	wrong := salesDoc()
	wrong.Kind = accountingsync.SyncObjectCustomerPayment
	_, err = conn.CreateSalesDocument(t.Context(), wrong)
	require.ErrorIs(t, err, errDocumentKind)
	assert.Empty(t, fake.writes())
}

func TestCreateSalesDocumentSendsAForeignCurrencyAndAFallbackReference(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := salesDoc()
	doc.CurrencyCode = "cad"
	doc.DocNumber = ""
	_, err := conn.CreateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	create, ok := fake.find(http.MethodPost, "/salesInvoices")
	require.True(t, ok)
	assert.Equal(t, "CAD", create.Body["currencyCode"])
	reference, isString := create.Body["externalDocumentNumber"].(string)
	require.True(t, isString)
	assert.Len(t, reference, 20)
	assert.True(t, strings.HasPrefix(reference, "T"))
	assert.Equal(t, requestReference(testRequestID), reference)
	assert.Equal(t, reference, docReference(strings.Repeat("x", 36), testRequestID))
}

func TestCreateCreditMemoAppliesToTheInvoice(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	server.entity = "salesCreditMemos"
	server.newID = testCreditMemo
	server.posted = docSpec{id: testCreditMemo, number: "S-CR-1", status: "Open"}
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := salesDoc()
	doc.Kind = accountingsync.SyncObjectCreditMemo
	doc.ApplyToExternalID = testInvoice
	doc.ApplyAmount = decimal.NewFromInt(61)
	result, err := conn.CreateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testCreditMemo, result.ExternalID)
	assert.Equal(t, testInvoice, result.Refs[accountingsync.ExternalRefApplication])

	create, ok := fake.find(http.MethodPost, "/salesCreditMemos")
	require.True(t, ok)
	assert.Equal(t, testInvoice, create.Body["invoiceId"])
	assert.Equal(t, "2026-10-01", create.Body["creditMemoDate"])
	assert.NotContains(t, create.Body, "dueDate")
	assert.NotContains(t, create.Body, "paymentTermsId")
	assert.Len(t, lines(t, create.Body, "salesCreditMemoLines"), 2)
}

func TestUpdateSalesInvoiceCancelsThenCreatesANewOne(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	server.existing = map[string]docSpec{
		testInvoice: {
			id: testInvoice, number: "PS-INV103001", status: "Open", reference: "TRN-1003",
			total: 61, remaining: 61, modified: "2026-10-05T10:00:00Z",
		},
	}
	server.listed = listJSON(t, docSpec{
		id: testInvoice, number: "PS-INV103001", status: "Canceled", reference: "TRN-1003",
	})
	server.extra = func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/salesCreditMemos") {
			return http.StatusOK, listJSON(t,
				docSpec{id: testCreditMemo, status: "Open", invoiceID: testNewBill},
				docSpec{id: testCorrective, status: "Corrective", invoiceID: testInvoice},
			)
		}
		return 0, ""
	}
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := salesDoc()
	doc.ExternalID = testInvoice
	doc.Refs = map[string]string{accountingsync.ExternalRefDocument: testInvoice}
	result, err := conn.UpdateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testNewInvoice, result.ExternalID)
	assert.Equal(t, "PS-INV-1003", result.DocNumber)
	assert.Equal(t, testNewInvoice, result.Refs[accountingsync.ExternalRefDocument])
	assert.Equal(t, testInvoice, result.Refs[refReplaced])
	assert.Equal(t, testCorrective, result.Refs[refCorrective])

	_, cancelled := fake.find(http.MethodPost, entity("salesInvoices", testInvoice)+cancelAction)
	assert.True(t, cancelled)
	memos, ok := fake.find(http.MethodGet, "/salesCreditMemos")
	require.True(t, ok)
	assert.Equal(t, "lastModifiedDateTime gt 2026-10-05T09:55:00Z", memos.filter())
	assert.Equal(t, 1, fake.count(http.MethodPost, "/salesInvoices"))

	_, err = conn.UpdateSalesDocument(t.Context(), salesDoc())
	require.ErrorIs(t, err, errExternalID)
}

func TestUpdateSalesInvoiceRetriesWithoutCancellingTwice(t *testing.T) {
	t.Parallel()

	server := newSalesServer(t)
	server.existing = map[string]docSpec{
		testInvoice: {id: testInvoice, status: "Canceled", total: 61},
	}
	server.listed = listJSON(t,
		docSpec{id: testInvoice, status: "Canceled", reference: "TRN-1003"},
		docSpec{id: testNewInvoice, number: "PS-INV-1003", status: "Open", reference: "TRN-1003"},
	)
	fake := &fakeBC{respond: server.respond}
	conn := testConnector(t, fake)

	doc := salesDoc()
	doc.ExternalID = testInvoice
	result, err := conn.UpdateSalesDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testNewInvoice, result.ExternalID)
	assert.Empty(t, fake.writes())
}

func TestVoidSalesDocumentPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     accountingsync.SyncObjectType
		entity   string
		doc      docSpec
		write    string
		method   string
		category accountingsync.SyncErrorCategory
	}{
		{
			name: "open invoice is cancelled", kind: accountingsync.SyncObjectInvoice,
			entity: "salesInvoices", doc: docSpec{status: "Open", total: 10, remaining: 10},
			write: cancelAction, method: http.MethodPost,
		},
		{
			name: "credit memo is cancelled", kind: accountingsync.SyncObjectCreditMemo,
			entity: "salesCreditMemos", doc: docSpec{status: "Open", total: 10},
			write: cancelAction, method: http.MethodPost,
		},
		{
			name: "draft is deleted", kind: accountingsync.SyncObjectInvoice,
			entity: "salesInvoices", doc: docSpec{status: "Draft", etag: `W/"JzE5OzE="`},
			write: "", method: http.MethodDelete,
		},
		{
			name: "paid invoice is refused", kind: accountingsync.SyncObjectInvoice,
			entity: "salesInvoices", doc: docSpec{status: "Open", total: 10, remaining: 4},
			category: accountingsync.SyncErrorConflict,
		},
		{
			name: "cancelled invoice is left alone", kind: accountingsync.SyncObjectInvoice,
			entity: "salesInvoices", doc: docSpec{status: "Canceled"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.doc.id = testInvoice
			server := newSalesServer(t)
			server.entity = tc.entity
			server.existing = map[string]docSpec{testInvoice: tc.doc}
			fake := &fakeBC{respond: server.respond}
			conn := testConnector(t, fake)

			result, err := conn.VoidSalesDocument(t.Context(), &services.AccountingDocumentRef{
				Auth: testAuth(), RequestID: testRequestID, Kind: tc.kind, ExternalID: testInvoice,
			})
			if tc.category != "" {
				assert.Equal(t, tc.category, syncErrorOf(t, err).Category)
				assert.Empty(t, fake.writes())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testInvoice, result.ExternalID)
			if tc.method == "" {
				assert.Empty(t, fake.writes())
				return
			}
			call, ok := fake.find(tc.method, entity(tc.entity, testInvoice)+tc.write)
			require.True(t, ok)
			if tc.method == http.MethodDelete {
				assert.Equal(t, `W/"JzE5OzE="`, call.IfMatch)
			}
		})
	}
}

func TestVoidSalesDocumentOfAMissingDocumentSucceeds(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)
	_, err := conn.VoidSalesDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), Kind: accountingsync.SyncObjectInvoice, ExternalID: testInvoice,
	})
	require.NoError(t, err)
	assert.Empty(t, fake.writes())
}

func TestClassifyDocumentError(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	held := &accountingsync.SyncError{Category: accountingsync.SyncErrorMapping}
	cases := []struct {
		name     string
		err      error
		category accountingsync.SyncErrorCategory
		retry    time.Duration
	}{
		{"rate limited", &businesscentral.APIError{Status: 429}, accountingsync.SyncErrorRateLimited, 30 * time.Second},
		{"rate limited with wait", &businesscentral.APIError{Status: 429, RetryAfter: 7 * time.Second},
			accountingsync.SyncErrorRateLimited, 7 * time.Second},
		{"auth", &businesscentral.APIError{Status: 401, Code: "Authentication_InvalidCredentials"},
			accountingsync.SyncErrorAuth, 0},
		{"revoked", &businesscentral.OAuthError{Code: "invalid_grant"}, accountingsync.SyncErrorAuth, 0},
		{"forbidden", &businesscentral.APIError{Status: 403, Code: "Authorization_InsufficientPermissions"},
			accountingsync.SyncErrorConfiguration, 0},
		{"not configured", ErrNotConfigured, accountingsync.SyncErrorConfiguration, 0},
		{"unavailable", &businesscentral.APIError{Status: 503}, accountingsync.SyncErrorTransient, 0},
		{"timeout", &businesscentral.APIError{Status: 408}, accountingsync.SyncErrorTransient, 0},
		{"duplicate", &businesscentral.APIError{Status: 400, Code: businesscentral.CodeEntityWithSameKey},
			accountingsync.SyncErrorDuplicate, 0},
		{"duplicate name", errDuplicateName, accountingsync.SyncErrorDuplicate, 0},
		{"changed", &businesscentral.APIError{Status: 400, Code: businesscentral.CodeEntityChanged},
			accountingsync.SyncErrorConflict, 0},
		{"posting date", &businesscentral.APIError{
			Status: 400, Code: businesscentral.CodeDialogException,
			Message: "Posting Date is not within your range of allowed posting dates",
		}, accountingsync.SyncErrorClosedPeriod, 0},
		{"missing", &businesscentral.APIError{Status: 404, Code: businesscentral.CodeRecordNotFound},
			accountingsync.SyncErrorNotFound, 0},
		{"dialog", &businesscentral.APIError{
			Status: 400, Code: businesscentral.CodeDialogException, Message: "Item is blocked",
		}, accountingsync.SyncErrorValidation, 0},
		{"bad request", &businesscentral.APIError{Status: 400, Code: "BadRequest_PropertyNotFound"},
			accountingsync.SyncErrorValidation, 0},
		{"local", businesscentral.ErrFieldTooLong, accountingsync.SyncErrorValidation, 0},
		{"unknown", errors.New("boom"), accountingsync.SyncErrorTransient, 0},
		{"held", held, accountingsync.SyncErrorMapping, 0},
	}
	for _, tc := range cases {
		got := conn.ClassifyDocumentError(tc.err)
		require.NotNil(t, got, tc.name)
		assert.Equal(t, tc.category, got.Category, tc.name)
		assert.Equal(t, tc.retry, got.RetryAfter, tc.name)
	}
	assert.Nil(t, conn.ClassifyDocumentError(nil))
	assert.Same(t, held, conn.ClassifyDocumentError(held))

	dialog := conn.ClassifyDocumentError(&businesscentral.APIError{
		Status: 400, Code: businesscentral.CodeDialogException, Message: "Item is blocked",
	})
	assert.Equal(t, "Item is blocked", dialog.Message)
	assert.Equal(t, businesscentral.CodeDialogException, dialog.Code)
}
