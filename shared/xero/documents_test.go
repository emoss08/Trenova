package xero_test

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freightInvoice() xero.InvoiceInput {
	return xero.InvoiceInput{
		Type:          xero.InvoiceTypeReceivable,
		ContactID:     testContact,
		InvoiceNumber: "INV-1001",
		Reference:     "PRO 55012",
		Date:          time.Date(2026, 9, 24, 15, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC),
		CurrencyCode:  "usd",
		Lines: []xero.LineItem{{
			Description: "Linehaul Chicago to Dallas",
			Quantity:    dec("1"),
			UnitAmount:  dec("1450.25"),
			LineAmount:  dec("1450.25"),
			AccountCode: "200",
		}, {
			Description: "Fuel surcharge",
			Quantity:    dec("812.5"),
			UnitAmount:  dec("0.1234"),
			ItemCode:    "FSC",
		}},
	}
}

func lineOf(t *testing.T, doc map[string]any, idx int) map[string]any {
	t.Helper()
	lines, ok := doc["LineItems"].([]any)
	require.True(t, ok)
	require.Greater(t, len(lines), idx)
	line, ok := lines[idx].(map[string]any)
	require.True(t, ok)
	return line
}

func TestCreateInvoicePutsAnAuthorisedNoTaxInvoice(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/Invoices")
		assert.Equal(t, "4", r.URL.Query().Get("unitdp"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_invoice.json"))
	})

	invoice, err := client.CreateInvoice(t.Context(), testKey, freightInvoice())
	require.NoError(t, err)

	sent := firstOf(t, body, "Invoices")
	assert.NotContains(t, sent, "InvoiceID")
	assert.Equal(t, "ACCREC", sent["Type"])
	assert.Equal(t, map[string]any{"ContactID": testContact}, sent["Contact"])
	assert.Equal(t, "INV-1001", sent["InvoiceNumber"])
	assert.Equal(t, "PRO 55012", sent["Reference"])
	assert.Equal(t, "2026-09-24", sent["Date"])
	assert.Equal(t, "2026-10-24", sent["DueDate"])
	assert.Equal(t, "USD", sent["CurrencyCode"])
	assert.NotContains(t, sent, "CurrencyRate", "a zero rate is left for Xero to fill")
	assert.Equal(t, "AUTHORISED", sent["Status"])
	assert.Equal(t, "NoTax", sent["LineAmountTypes"])

	first := lineOf(t, sent, 0)
	assert.Equal(t, "Linehaul Chicago to Dallas", first["Description"])
	assert.InDelta(t, 1.0, first["Quantity"], 0)
	assert.InDelta(t, 1450.25, first["UnitAmount"], 0)
	assert.InDelta(t, 1450.25, first["LineAmount"], 0)
	assert.Equal(t, "200", first["AccountCode"])
	assert.NotContains(t, first, "ItemCode")

	second := lineOf(t, sent, 1)
	assert.InDelta(t, 0.1234, second["UnitAmount"], 0)
	assert.Equal(t, "FSC", second["ItemCode"])
	assert.NotContains(t, second, "AccountCode")
	assert.NotContains(t, second, "LineAmount")

	assert.Equal(t, testInvoice, invoice.InvoiceID)
	assert.Equal(t, "INV-1001", invoice.InvoiceNumber)
	assert.Equal(t, "AUTHORISED", invoice.Status)
	assert.Equal(t, testContact, invoice.ContactID)
	assert.True(t, dec("1450.25").Equal(invoice.Total))
	assert.True(t, dec("1450.25").Equal(invoice.AmountDue))
	assert.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), invoice.Date)
	assert.Equal(t, time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), invoice.DueDate)
	assert.Equal(t, time.Date(2026, 9, 25, 1, 0, 30, 0, time.UTC), invoice.UpdatedAt)
}

func TestCreateInvoiceSendsARateOnlyWhenGiven(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_invoice.json"))
	})

	in := freightInvoice()
	in.Type = "accpay"
	in.CurrencyCode = "CAD"
	in.CurrencyRate = dec("1.364512")
	in.Status = xero.StatusDraft
	in.LineAmountTypes = "exclusive"
	_, err := client.CreateInvoice(t.Context(), testKey, in)
	require.NoError(t, err)

	sent := firstOf(t, body, "Invoices")
	assert.Equal(t, "ACCPAY", sent["Type"])
	assert.InDelta(t, 1.364512, sent["CurrencyRate"], 0)
	assert.Equal(t, "DRAFT", sent["Status"])
	assert.Equal(t, "Exclusive", sent["LineAmountTypes"])
}

func TestCreateInvoiceValidatesBeforeCallingXero(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	check := func(mutate func(*xero.InvoiceInput), want error) {
		t.Helper()
		in := freightInvoice()
		mutate(&in)
		_, err := client.CreateInvoice(t.Context(), testKey, in)
		require.ErrorIs(t, err, want)
	}
	check(func(in *xero.InvoiceInput) { in.Type = "ACCRECCREDIT" }, xero.ErrUnknownType)
	check(func(in *xero.InvoiceInput) { in.ContactID = "" }, xero.ErrContactRequired)
	check(func(in *xero.InvoiceInput) { in.ContactID = "58" }, xero.ErrInvalidID)
	check(func(in *xero.InvoiceInput) { in.Date = time.Time{} }, xero.ErrDateRequired)
	check(func(in *xero.InvoiceInput) { in.CurrencyCode = "US" }, xero.ErrCurrencyCodeInvalid)
	check(func(in *xero.InvoiceInput) { in.CurrencyRate = dec("-1") }, xero.ErrNegativeRate)
	check(func(in *xero.InvoiceInput) { in.Status = xero.StatusPaid }, xero.ErrUnknownStatus)
	check(func(in *xero.InvoiceInput) { in.LineAmountTypes = "Gross" }, xero.ErrUnknownAmountTypes)
	check(func(in *xero.InvoiceInput) { in.Lines = nil }, xero.ErrLinesRequired)
	check(func(in *xero.InvoiceInput) {
		in.Lines = []xero.LineItem{{Quantity: dec("1"), UnitAmount: dec("10")}}
	}, xero.ErrLineDescription)
	assert.Zero(t, calls.Load(), "nothing reaches Xero")
}

func TestUpdateInvoicePostsToTheInvoice(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/Invoices/"+testInvoice)
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_invoice.json"))
	})

	_, err := client.UpdateInvoice(t.Context(), testKey, testInvoice, freightInvoice())
	require.NoError(t, err)
	sent := firstOf(t, body, "Invoices")
	assert.Equal(t, testInvoice, sent["InvoiceID"])
	assert.Equal(t, "INV-1001", sent["InvoiceNumber"])

	_, err = client.UpdateInvoice(t.Context(), testKey, "", freightInvoice())
	require.ErrorIs(t, err, xero.ErrIDRequired)
}

func TestVoidInvoiceSendsOnlyTheIDAndStatus(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/Invoices/"+testInvoice)
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "void_invoice.json"))
	})

	invoice, err := client.VoidInvoice(t.Context(), testKey, testInvoice)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"InvoiceID": testInvoice, "Status": "VOIDED"}, firstOf(t, body, "Invoices"))
	assert.Equal(t, xero.StatusVoided, invoice.Status)
	assert.True(t, invoice.AmountDue.IsZero())
}

func TestInvoicesByIDReadsPaymentsAndCredits(t *testing.T) {
	t.Parallel()

	var ids string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Invoices")
		ids = r.URL.Query().Get("IDs")
		_, _ = w.Write(fixture(t, "invoices_by_id.json"))
	})

	invoices, err := client.InvoicesByID(t.Context(), []string{
		testInvoice, " " + testBill + " ", "FEE88EEA-F2AA-4A71-A372-33D6D83D3C45",
	})
	require.NoError(t, err)
	assert.Equal(t, testInvoice+","+testBill, ids, "ids are trimmed, lower cased and asked for once")

	require.Len(t, invoices, 2)
	open := invoices[0]
	assert.Equal(t, "ACCREC", open.Type)
	assert.True(t, dec("850.25").Equal(open.AmountDue))
	assert.True(t, dec("500").Equal(open.AmountPaid))
	assert.True(t, dec("100").Equal(open.AmountCredited))
	assert.True(t, dec("1").Equal(open.CurrencyRate))
	require.Len(t, open.Payments, 1)
	assert.Equal(t, testPayment, open.Payments[0].PaymentID)
	assert.True(t, dec("500").Equal(open.Payments[0].Amount))
	require.Len(t, open.CreditNotes, 1)
	assert.Equal(t, testCreditNote, open.CreditNotes[0].CreditNoteID)
	assert.True(t, dec("100").Equal(open.CreditNotes[0].Amount))

	bill := invoices[1]
	assert.Equal(t, "ACCPAY", bill.Type)
	assert.Equal(t, xero.StatusVoided, bill.Status)
	assert.Empty(t, bill.Payments)
	assert.True(t, bill.DueDate.IsZero())
}

func TestInvoicesByIDRefusesWhatCannotBeAnID(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	invoices, err := client.InvoicesByID(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, invoices)

	_, err = client.InvoicesByID(t.Context(), []string{testInvoice, testBill + ",x"})
	require.ErrorIs(t, err, xero.ErrInvalidID)

	many := make([]string, 0, xero.MaxIDsPerRead+1)
	for i := range xero.MaxIDsPerRead + 1 {
		many = append(many, fmt.Sprintf("00000000-0000-4000-8000-%012d", i))
	}
	_, err = client.InvoicesByID(t.Context(), many)
	require.ErrorIs(t, err, xero.ErrTooManyIDs)
	assert.Zero(t, calls.Load())
}

func TestInvoicesPageUsesPaginationAndModifiedSince(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Invoices")
		query := r.URL.Query()
		assert.Equal(t, "2", query.Get("page"))
		assert.Equal(t, "1000", query.Get("pageSize"))
		assert.Empty(t, query.Get("Statuses"), "a sync reads every status")
		assert.Equal(t, "Fri, 25 Sep 2026 00:00:00 GMT", r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "invoices_page.json"))
	})

	page, err := client.Invoices(t.Context(), 2, &since)
	require.NoError(t, err)
	assert.Len(t, page.Invoices, 2)
	assert.False(t, page.More, "page 2 of 2")

	_, err = client.Invoices(t.Context(), -1, nil)
	require.ErrorIs(t, err, xero.ErrInvalidPage)
}

func TestFindInvoicesUsesParamsAndAWhereForTypeAndDate(t *testing.T) {
	t.Parallel()

	var pages []string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Invoices")
		query := r.URL.Query()
		pages = append(pages, query.Get("page"))
		assert.Equal(t, "INV-1001,BILL 88/2", query.Get("InvoiceNumbers"))
		assert.Equal(t, testContact, query.Get("ContactIDs"))
		assert.Equal(t, "AUTHORISED,PAID", query.Get("Statuses"))
		assert.Equal(t,
			`Type=="ACCPAY" AND Date>=DateTime(2026,01,02) AND Date<=DateTime(2026,09,30)`,
			query.Get("where"))
		assert.Contains(t, r.URL.RawQuery, "InvoiceNumbers=INV-1001%2CBILL+88%2F2")
		if query.Get("page") == "1" {
			_, _ = w.Write([]byte(`{"Invoices":[{"InvoiceID":"` + testInvoice +
				`"}],"pagination":{"page":1,"pageSize":1000,"pageCount":2,"itemCount":1001}}`))
			return
		}
		_, _ = w.Write([]byte(`{"Invoices":[{"InvoiceID":"` + testBill +
			`"}],"pagination":{"page":2,"pageSize":1000,"pageCount":2,"itemCount":1001}}`))
	})

	invoices, err := client.FindInvoices(t.Context(), xero.InvoiceFilter{
		Type:       "accpay",
		Numbers:    []string{"INV-1001", "BILL 88/2", "INV-1001"},
		ContactIDs: []string{testContact},
		Statuses:   []string{"authorised", "PAID"},
		DateFrom:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		DateTo:     time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, pages, "every page of the result is read")
	require.Len(t, invoices, 2)
	assert.Equal(t, testInvoice, invoices[0].InvoiceID)
	assert.Equal(t, testBill, invoices[1].InvoiceID)
}

func TestFindInvoicesRejectsUnsafeOrEmptyFilters(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	_, err := client.FindInvoices(t.Context(), xero.InvoiceFilter{})
	require.ErrorIs(t, err, xero.ErrEmptyFilter)
	_, err = client.FindInvoices(t.Context(), xero.InvoiceFilter{Numbers: []string{`A" OR Type=="ACCPAY`}})
	require.ErrorIs(t, err, xero.ErrInvalidFilter)
	_, err = client.FindInvoices(t.Context(), xero.InvoiceFilter{Numbers: []string{"A,B"}})
	require.ErrorIs(t, err, xero.ErrInvalidFilter)
	_, err = client.FindInvoices(t.Context(), xero.InvoiceFilter{Type: "SALES"})
	require.ErrorIs(t, err, xero.ErrUnknownType)
	_, err = client.FindInvoices(t.Context(), xero.InvoiceFilter{Statuses: []string{"OPEN"}})
	require.ErrorIs(t, err, xero.ErrUnknownStatus)
	_, err = client.FindInvoices(t.Context(), xero.InvoiceFilter{ContactIDs: []string{"x"}})
	require.ErrorIs(t, err, xero.ErrInvalidID)
	assert.Zero(t, calls.Load())
}

func TestDuplicateInvoiceNumberIsADuplicate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_duplicate_number.json"))
	})

	_, err := client.CreateInvoice(t.Context(), testKey, freightInvoice())
	require.Error(t, err)
	assert.True(t, xero.IsValidation(err))
	assert.True(t, xero.IsDuplicateNumber(err))
	assert.False(t, xero.IsLockDate(err))
	assert.Contains(t, err.Error(), "Invoice # must be unique.")
}

func TestInvoiceBeforeTheLockDateIsALockDate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_lock_date.json"))
	})

	_, err := client.CreateInvoice(t.Context(), testKey, freightInvoice())
	require.Error(t, err)
	assert.True(t, xero.IsLockDate(err))
	assert.False(t, xero.IsDuplicateNumber(err))
	assert.False(t, xero.IsTransient(err))
}

func creditNote() xero.CreditNoteInput {
	return xero.CreditNoteInput{
		Type:             xero.CreditNoteTypeReceivable,
		ContactID:        testContact,
		CreditNoteNumber: "CN-0007",
		Reference:        "Rate correction PRO 55012",
		Date:             time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		Lines: []xero.LineItem{{
			Description: "Rate correction",
			Quantity:    dec("1"),
			UnitAmount:  dec("250"),
			AccountCode: "200",
		}},
	}
}

func TestCreateCreditNotePutsTheCredit(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/CreditNotes")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_credit_note.json"))
	})

	note, err := client.CreateCreditNote(t.Context(), testKey, creditNote())
	require.NoError(t, err)
	sent := firstOf(t, body, "CreditNotes")
	assert.Equal(t, "ACCRECCREDIT", sent["Type"])
	assert.Equal(t, "CN-0007", sent["CreditNoteNumber"])
	assert.Equal(t, "2026-09-24", sent["Date"])
	assert.Equal(t, "AUTHORISED", sent["Status"])
	assert.Equal(t, "NoTax", sent["LineAmountTypes"])
	assert.NotContains(t, sent, "DueDate")
	assert.Equal(t, testCreditNote, note.CreditNoteID)
	assert.True(t, dec("250").Equal(note.Total))
	assert.True(t, dec("250").Equal(note.RemainingCredit))
	assert.Empty(t, note.Allocations)

	in := creditNote()
	in.Type = xero.InvoiceTypeReceivable
	_, err = client.CreateCreditNote(t.Context(), testKey, in)
	require.ErrorIs(t, err, xero.ErrUnknownType)
}

func TestUpdateAndVoidCreditNote(t *testing.T) {
	t.Parallel()

	var bodies []map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/CreditNotes/"+testCreditNote)
		body := readJSON(t, r)
		bodies = append(bodies, body)
		if firstOf(t, body, "CreditNotes")["Status"] == "VOIDED" {
			_, _ = w.Write(fixture(t, "void_credit_note.json"))
			return
		}
		_, _ = w.Write(fixture(t, "create_credit_note.json"))
	})

	_, err := client.UpdateCreditNote(t.Context(), testKey, testCreditNote, creditNote())
	require.NoError(t, err)
	voided, err := client.VoidCreditNote(t.Context(), testKey, testCreditNote)
	require.NoError(t, err)

	require.Len(t, bodies, 2)
	assert.Equal(t, testCreditNote, firstOf(t, bodies[0], "CreditNotes")["CreditNoteID"])
	assert.Equal(t, map[string]any{"CreditNoteID": testCreditNote, "Status": "VOIDED"},
		firstOf(t, bodies[1], "CreditNotes"))
	assert.Equal(t, xero.StatusVoided, voided.Status)
}

func TestCreditNotesByIDReadAllocations(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/CreditNotes")
		assert.Equal(t, testCreditNote, r.URL.Query().Get("IDs"))
		_, _ = w.Write(fixture(t, "credit_notes_by_id.json"))
	})

	notes, err := client.CreditNotesByID(t.Context(), []string{testCreditNote})
	require.NoError(t, err)
	require.Len(t, notes, 1)
	note := notes[0]
	assert.Equal(t, "CN-0007", note.CreditNoteNumber)
	assert.Equal(t, testContact, note.ContactID)
	assert.True(t, dec("150").Equal(note.RemainingCredit))
	require.Len(t, note.Allocations, 1)
	allocation := note.Allocations[0]
	assert.Equal(t, testAllocation, allocation.AllocationID)
	assert.Equal(t, testInvoice, allocation.InvoiceID)
	assert.Equal(t, testCreditNote, allocation.CreditNoteID)
	assert.True(t, dec("100").Equal(allocation.Amount))
	assert.Equal(t, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), allocation.Date)
}

func TestCreditNotesPageWithoutPaginationIsTheLastWhenShort(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/CreditNotes")
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "1000", r.URL.Query().Get("pageSize"))
		assert.NotEmpty(t, r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "credit_notes_page.json"))
	})

	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	page, err := client.CreditNotes(t.Context(), 1, &since)
	require.NoError(t, err)
	assert.Len(t, page.CreditNotes, 1)
	assert.False(t, page.More)
}

func TestFindCreditNotesBuildsAWhere(t *testing.T) {
	t.Parallel()

	var where string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/CreditNotes")
		where = r.URL.Query().Get("where")
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		_, _ = w.Write(fixture(t, "credit_notes_page.json"))
	})

	notes, err := client.FindCreditNotes(t.Context(), xero.InvoiceFilter{
		Type:       xero.CreditNoteTypeReceivable,
		Numbers:    []string{"CN-0007", "CN-0008"},
		ContactIDs: []string{testContact},
		Statuses:   []string{xero.StatusAuthorised},
		DateFrom:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, notes, 1)
	assert.Equal(t,
		`Type=="ACCRECCREDIT" AND (CreditNoteNumber=="CN-0007" OR CreditNoteNumber=="CN-0008") AND `+
			`Contact.ContactID==guid("`+testContact+`") AND Status=="AUTHORISED" AND Date>=DateTime(2026,09,01)`,
		where)
}

func TestAllocateAppliesTheCreditToAnInvoice(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/CreditNotes/"+testCreditNote+"/Allocations")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "allocation.json"))
	})

	allocation, err := client.Allocate(t.Context(), testKey, testCreditNote, xero.AllocationInput{
		InvoiceID: testInvoice,
		Amount:    dec("100"),
		Date:      time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	sent := firstOf(t, body, "Allocations")
	assert.Equal(t, map[string]any{"InvoiceID": testInvoice}, sent["Invoice"])
	assert.InDelta(t, 100.0, sent["Amount"], 0)
	assert.Equal(t, "2026-09-25", sent["Date"])
	assert.Equal(t, testAllocation, allocation.AllocationID)
	assert.Equal(t, testInvoice, allocation.InvoiceID)
	assert.Equal(t, testCreditNote, allocation.CreditNoteID)

	_, err = client.Allocate(t.Context(), testKey, testCreditNote, xero.AllocationInput{
		InvoiceID: testInvoice,
		Amount:    dec("0"),
	})
	require.ErrorIs(t, err, xero.ErrAmountNotPositive)
}

func TestDeleteAllocation(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api.xro/2.0/CreditNotes/"+testCreditNote+"/Allocations/"+testAllocation, r.URL.Path)
		assertAPIHeaders(t, r)
		_, _ = w.Write(fixture(t, "delete_allocation.json"))
	})

	require.NoError(t, client.DeleteAllocation(t.Context(), testCreditNote, testAllocation))
	require.ErrorIs(t, client.DeleteAllocation(t.Context(), testCreditNote, "1/../2"), xero.ErrInvalidID)
}
