package businesscentral_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSalesInvoicesSinceAndShape(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/salesInvoices")
		assert.Equal(
			t,
			"lastModifiedDateTime gt 2026-09-01T00:00:00Z",
			r.URL.Query().Get("$filter"),
		)
		_, _ = w.Write(fixture(t, "sales_invoices.json"))
	})

	docs, err := client.Documents(t.Context(), businesscentral.DocumentSalesInvoice, &since)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	open := docs[0]
	assert.Equal(t, businesscentral.DocumentSalesInvoice, open.Kind)
	assert.Equal(t, testInvoice, open.ID)
	assert.Equal(t, "PS-INV103001", open.Number)
	assert.Equal(t, "TRN-1001", open.ExternalDocumentNumber)
	assert.Equal(t, "2026-09-01", open.DocumentDate)
	assert.Equal(t, "2026-09-01", open.PostingDate)
	assert.Equal(t, "2026-10-01", open.DueDate)
	assert.Equal(t, testCustomer, open.PartyID)
	assert.Equal(t, "C00010", open.PartyNumber)
	assert.Equal(t, "USD", open.CurrencyCode)
	assert.Equal(t, "66666666-7777-4888-8999-aaaaaaaaaaa1", open.PaymentTermsID)
	assert.Equal(t, businesscentral.StatusOpen, open.Status)
	assert.True(t, dec("1250.5").Equal(open.TotalAmountIncludingTax))
	assert.True(t, dec("250.5").Equal(open.RemainingAmount))
	assert.Equal(t, time.Date(2026, 9, 1, 18, 0, 0, 123000000, time.UTC), open.LastModified)
	assert.Equal(t, `W/"JzIwOzE="`, open.ETag)
	assert.True(t, docs[1].IsDraft())
	assert.Empty(t, docs[1].DueDate, "the zero BC date is unset")
	assert.Empty(t, docs[1].PaymentTermsID)
}

func TestFindSalesInvoiceByExternalNumberAndCustomer(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/salesInvoices")
		assert.Equal(t,
			"externalDocumentNumber eq 'TRN''1001' and customerId eq "+testCustomer,
			r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "sales_invoices.json"))
	})

	docs, err := client.FindDocuments(t.Context(), businesscentral.DocumentSalesInvoice,
		"TRN'1001", strings.ToUpper(testCustomer))
	require.NoError(t, err)
	assert.Len(t, docs, 2)

	_, err = client.FindDocuments(t.Context(), businesscentral.DocumentSalesInvoice,
		strings.Repeat("x", 36), "")
	require.ErrorIs(t, err, businesscentral.ErrInvalidFilter)
	_, err = client.FindDocuments(t.Context(), businesscentral.DocumentSalesInvoice,
		"TRN-1", "1 or 1 eq 1")
	require.ErrorIs(t, err, businesscentral.ErrInvalidID)
}

func TestFindPurchaseInvoiceByVendorInvoiceNumber(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/purchaseInvoices")
		assert.Equal(t, "vendorInvoiceNumber eq 'OO-77' and vendorId eq "+testVendor,
			r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "purchase_invoices.json"))
	})

	docs, err := client.FindDocuments(t.Context(), businesscentral.DocumentPurchaseInvoice,
		"OO-77", testVendor)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "OO-77", docs[0].ExternalDocumentNumber)
	assert.Equal(t, testVendor, docs[0].PartyID)
	assert.Equal(t, "V00010", docs[0].PartyNumber)
	assert.Equal(t, businesscentral.StatusInReview, docs[0].Status)
}

func TestDocumentsByIDUsesInFilter(t *testing.T) {
	t.Parallel()

	other := "99999999-aaaa-4bbb-8ccc-ddddddddddd2"
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "id in ("+testInvoice+","+other+")", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "sales_invoices.json"))
	})

	docs, err := client.DocumentsByID(t.Context(), businesscentral.DocumentSalesInvoice,
		[]string{testInvoice, strings.ToUpper(testInvoice), other})
	require.NoError(t, err)
	assert.Len(t, docs, 2)

	empty, err := client.DocumentsByID(t.Context(), businesscentral.DocumentSalesInvoice, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	ids := make([]string, 0, businesscentral.MaxIDsPerRead+1)
	for idx := range businesscentral.MaxIDsPerRead + 1 {
		ids = append(ids, "00000000-0000-4000-8000-0000000000"+twoDigits(idx))
	}
	_, err = client.DocumentsByID(t.Context(), businesscentral.DocumentSalesInvoice, ids)
	require.ErrorIs(t, err, businesscentral.ErrTooManyIDs)
}

func twoDigits(value int) string {
	return string([]byte{byte('0' + value/10), byte('0' + value%10)})
}

func TestSalesCreditMemos(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/salesCreditMemos")
		_, _ = w.Write(fixture(t, "sales_credit_memos.json"))
	})

	docs, err := client.Documents(t.Context(), businesscentral.DocumentSalesCreditMemo, nil)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "2026-09-05", docs[0].DocumentDate)
	assert.Equal(t, testInvoice, docs[0].InvoiceID)
	assert.Equal(t, "PS-INV103001", docs[0].InvoiceNumber)
	assert.Equal(t, "TRN-CM-1", docs[0].ExternalDocumentNumber)
}

func TestPurchaseCreditMemos(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/purchaseCreditMemos")
		_, _ = w.Write(fixture(t, "purchase_credit_memos.json"))
	})

	docs, err := client.Documents(t.Context(), businesscentral.DocumentPurchaseCreditMemo, nil)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "OO-CM-1", docs[0].ExternalDocumentNumber)
	assert.Equal(t, businesscentral.StatusCanceled, docs[0].Status)
	assert.Equal(t, "2026-09-08", docs[0].DocumentDate)
}

func TestDocumentByID(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/purchaseInvoices(cccccccc-dddd-4eee-8fff-000000000002)")
		_, _ = w.Write(fixture(t, "create_purchase_invoice.json"))
	})

	doc, err := client.Document(t.Context(), businesscentral.DocumentPurchaseInvoice,
		"cccccccc-dddd-4eee-8fff-000000000002")
	require.NoError(t, err)
	assert.Equal(t, "OO-78", doc.ExternalDocumentNumber)
	assert.Equal(t, businesscentral.DocumentPurchaseInvoice, doc.Kind)
}

func freightInvoice() businesscentral.DocumentInput {
	return businesscentral.DocumentInput{
		PartyID:                testCustomer,
		ExternalDocumentNumber: "TRN-1003",
		DocumentDate:           "2026-10-01",
		PostingDate:            "2026-10-01",
		DueDate:                "2026-10-31",
		CurrencyCode:           "usd",
		PaymentTermsID:         "66666666-7777-4888-8999-aaaaaaaaaaa1",
		Lines: []businesscentral.LineInput{
			{
				ItemID:      "44444444-5555-4666-8777-888888888881",
				Description: "Linehaul ATL-SEA",
				Quantity:    dec("1"),
				UnitPrice:   dec("1250.50"),
			},
			{
				AccountID:   testAccount,
				Description: "Detention",
				Quantity:    dec("2"),
				UnitPrice:   dec("25.125"),
			},
		},
	}
}

func TestCreateSalesInvoiceDeepInsertsLines(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/salesInvoices")
		assert.Equal(t, map[string]any{
			"externalDocumentNumber": "TRN-1003",
			"invoiceDate":            "2026-10-01",
			"postingDate":            "2026-10-01",
			"dueDate":                "2026-10-31",
			"customerId":             testCustomer,
			"currencyCode":           "USD",
			"paymentTermsId":         "66666666-7777-4888-8999-aaaaaaaaaaa1",
			"salesInvoiceLines": []any{
				map[string]any{
					"lineType":    "Item",
					"itemId":      "44444444-5555-4666-8777-888888888881",
					"description": "Linehaul ATL-SEA",
					"quantity":    float64(1),
					"unitPrice":   1250.5,
				},
				map[string]any{
					"lineType":    "Account",
					"accountId":   testAccount,
					"description": "Detention",
					"quantity":    float64(2),
					"unitPrice":   25.125,
				},
			},
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_sales_invoice.json"))
	})

	in := freightInvoice()
	doc, err := client.CreateDocument(t.Context(), businesscentral.DocumentSalesInvoice, &in)
	require.NoError(t, err)
	assert.Equal(t, "S-INV-0003", doc.Number)
	assert.True(t, doc.IsDraft())
	assert.True(t, dec("1300.75").Equal(doc.TotalAmountIncludingTax))
}

func TestCreatePurchaseInvoiceUsesDirectUnitCost(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/purchaseInvoices")
		body := readJSON(t, r)
		assert.Equal(t, "OO-78", body["vendorInvoiceNumber"])
		assert.Equal(t, testVendor, body["vendorId"])
		assert.NotContains(t, body, "customerId")
		assert.NotContains(t, body, "externalDocumentNumber")
		lines, ok := body["purchaseInvoiceLines"].([]any)
		require.True(t, ok)
		require.Len(t, lines, 1)
		assert.Equal(t, map[string]any{
			"lineType":       "Account",
			"accountId":      testAccount,
			"description":    "Settlement",
			"quantity":       float64(1),
			"directUnitCost": float64(1000),
		}, lines[0])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_purchase_invoice.json"))
	})

	doc, err := client.CreateDocument(t.Context(), businesscentral.DocumentPurchaseInvoice,
		&businesscentral.DocumentInput{
			PartyID:                testVendor,
			ExternalDocumentNumber: "OO-78",
			DocumentDate:           "2026-10-02",
			Lines: []businesscentral.LineInput{{
				AccountID: testAccount, Description: "Settlement", Quantity: dec("1"),
				UnitPrice: dec("1000"),
			}},
		})
	require.NoError(t, err)
	assert.Equal(t, "108002", doc.Number)
}

func TestCreateCreditMemoAppliesToInvoice(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/salesCreditMemos")
		body := readJSON(t, r)
		assert.Equal(t, "2026-09-05", body["creditMemoDate"])
		assert.Equal(t, testInvoice, body["invoiceId"])
		assert.Contains(t, body, "salesCreditMemoLines")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"bbbbbbbb-cccc-4ddd-8eee-fffffffffff1","status":"Draft"}`))
	})

	doc, err := client.CreateDocument(t.Context(), businesscentral.DocumentSalesCreditMemo,
		&businesscentral.DocumentInput{
			PartyID:      testCustomer,
			DocumentDate: "2026-09-05",
			InvoiceID:    testInvoice,
			Lines: []businesscentral.LineInput{{
				AccountID: testAccount, Quantity: dec("1"), UnitPrice: dec("100"),
			}},
		})
	require.NoError(t, err)
	assert.Equal(t, businesscentral.DocumentSalesCreditMemo, doc.Kind)
}

func TestCreateDocumentValidatesInput(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("invalid input is never sent")
	})
	cases := map[string]struct {
		kind   businesscentral.DocumentKind
		mutate func(*businesscentral.DocumentInput)
		want   error
	}{
		"no party": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.PartyID = "" },
			businesscentral.ErrPartyRequired},
		"bad date": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.PostingDate = "2026-13-01" },
			businesscentral.ErrInvalidDate},
		"long reference": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) {
				in.ExternalDocumentNumber = strings.Repeat("x", 36)
			},
			businesscentral.ErrFieldTooLong},
		"line without target": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) {
				in.Lines[0].ItemID = ""
				in.Lines[0].AccountID = ""
			},
			businesscentral.ErrLinesRequired},
		"line with both targets": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.Lines[0].AccountID = testAccount },
			businesscentral.ErrLinesRequired},
		"zero quantity": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.Lines[0].Quantity = dec("0") },
			businesscentral.ErrQuantityInvalid},
		"negative price": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.Lines[0].UnitPrice = dec("-1") },
			businesscentral.ErrPriceInvalid},
		"long description": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) {
				in.Lines[0].Description = strings.Repeat("d", 101)
			},
			businesscentral.ErrFieldTooLong},
		"invoice id on an invoice": {businesscentral.DocumentSalesInvoice,
			func(in *businesscentral.DocumentInput) { in.InvoiceID = testInvoice },
			businesscentral.ErrFieldNotSupported},
		"payment terms on a purchase": {businesscentral.DocumentPurchaseInvoice,
			func(*businesscentral.DocumentInput) {}, businesscentral.ErrFieldNotSupported},
		"unknown kind": {businesscentral.DocumentKind(0),
			func(*businesscentral.DocumentInput) {}, businesscentral.ErrUnknownKind},
	}
	for name, tc := range cases {
		in := freightInvoice()
		tc.mutate(&in)
		_, err := client.CreateDocument(t.Context(), tc.kind, &in)
		require.ErrorIs(t, err, tc.want, name)
	}
}

func TestCreateDocumentLine(t *testing.T) {
	t.Parallel()

	doc := "99999999-aaaa-4bbb-8ccc-ddddddddddd3"
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(
			t,
			r,
			http.MethodPost,
			testCompanyPath+"/salesInvoices("+doc+")/salesInvoiceLines",
		)
		assert.Equal(t, map[string]any{
			"lineType": "Account", "accountId": testAccount, "description": "Detention",
			"quantity": float64(2), "unitPrice": 25.125,
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_sales_invoice_line.json"))
	})

	line, err := client.CreateDocumentLine(t.Context(), businesscentral.DocumentSalesInvoice, doc,
		&businesscentral.LineInput{
			AccountID: testAccount, Description: "Detention", Quantity: dec("2"),
			UnitPrice: dec("25.125"),
		})
	require.NoError(t, err)
	assert.Equal(t, doc, line.DocumentID)
	assert.Equal(t, 10000, line.Sequence)
	assert.Equal(t, businesscentral.LineTypeAccount, line.LineType)
	assert.Empty(t, line.ItemID)
	assert.Equal(t, testAccount, line.AccountID)
	assert.True(t, dec("25.125").Equal(line.UnitPrice))
	assert.True(t, dec("50.25").Equal(line.AmountIncludingTax))
}

func TestDocumentActions(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, r.URL.Path)
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := t.Context()
	require.NoError(t, client.PostDocument(ctx, businesscentral.DocumentSalesInvoice, testInvoice))
	require.NoError(
		t,
		client.CancelDocument(ctx, businesscentral.DocumentSalesInvoice, testInvoice),
	)
	require.NoError(
		t,
		client.PostDocument(ctx, businesscentral.DocumentSalesCreditMemo, testInvoice),
	)
	require.NoError(
		t,
		client.CancelDocument(ctx, businesscentral.DocumentSalesCreditMemo, testInvoice),
	)
	require.NoError(
		t,
		client.PostDocument(ctx, businesscentral.DocumentPurchaseInvoice, testInvoice),
	)
	require.NoError(
		t,
		client.PostDocument(ctx, businesscentral.DocumentPurchaseCreditMemo, testInvoice),
	)
	require.NoError(
		t,
		client.CancelDocument(ctx, businesscentral.DocumentPurchaseCreditMemo, testInvoice),
	)
	require.ErrorIs(t,
		client.CancelDocument(ctx, businesscentral.DocumentPurchaseInvoice, testInvoice),
		businesscentral.ErrUnsupportedAction)
	require.ErrorIs(t, client.PostDocument(ctx, businesscentral.DocumentSalesInvoice, "1"),
		businesscentral.ErrInvalidID)

	mu.Lock()
	defer mu.Unlock()
	id := "(" + testInvoice + ")/Microsoft.NAV."
	assert.Equal(t, []string{
		testCompanyPath + "/salesInvoices" + id + "post",
		testCompanyPath + "/salesInvoices" + id + "cancel",
		testCompanyPath + "/salesCreditMemos" + id + "post",
		testCompanyPath + "/salesCreditMemos" + id + "cancel",
		testCompanyPath + "/purchaseInvoices" + id + "post",
		testCompanyPath + "/purchaseCreditMemos" + id + "post",
		testCompanyPath + "/purchaseCreditMemos" + id + "cancel",
	}, paths)
}

func TestDeleteDocumentSendsIfMatch(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodDelete, testCompanyPath+"/salesInvoices("+testInvoice+")")
		assert.Equal(t, `W/"JzIwOzE="`, r.Header.Get("If-Match"))
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, client.DeleteDocument(t.Context(), businesscentral.DocumentSalesInvoice,
		testInvoice, `W/"JzIwOzE="`))
}
