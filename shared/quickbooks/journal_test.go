package quickbooks_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoiceJournal() *quickbooks.JournalTxn {
	return &quickbooks.JournalTxn{
		DocNumber:   "JE-1042",
		TxnDate:     "2026-04-10",
		PrivateNote: "Trenova journal entry JE-1042",
		Lines: []quickbooks.JournalLine{
			{
				PostingType: quickbooks.PostingDebit,
				AccountID:   "84",
				Amount:      decimal.RequireFromString("1500.00"),
				Description: "Invoice INV-1001",
				EntityType:  quickbooks.JournalEntityCustomer,
				EntityID:    "58",
			},
			{
				PostingType: quickbooks.PostingCredit,
				AccountID:   "79",
				Amount:      decimal.RequireFromString("1500.00"),
				Description: "Freight revenue",
			},
		},
	}
}

func TestCreateJournalEntrySendsBalancedLinesWithTheirNames(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/journalentry", r.URL.Path)
		assert.Equal(t, testRequestID, r.URL.Query().Get("requestid"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_journal_entry.json"))
	})

	result, err := client.CreateJournalEntry(t.Context(), testRequestID, invoiceJournal())
	require.NoError(t, err)
	assert.Equal(t, quickbooks.TxnJournalEntry, result.Kind)
	assert.Equal(t, "310", result.ID)
	assert.Equal(t, "JE-1042", result.DocNumber)

	assert.Equal(t, "JE-1042", body["DocNumber"])
	assert.Equal(t, "2026-04-10", body["TxnDate"])
	assert.NotContains(t, body, "CurrencyRef")
	assert.Equal(t, []any{
		map[string]any{
			"DetailType":  "JournalEntryLineDetail",
			"Amount":      float64(1500),
			"Description": "Invoice INV-1001",
			"JournalEntryLineDetail": map[string]any{
				"PostingType": "Debit",
				"AccountRef":  map[string]any{"value": "84"},
				"Entity": map[string]any{
					"Type":      "Customer",
					"EntityRef": map[string]any{"value": "58"},
				},
			},
		},
		map[string]any{
			"DetailType":  "JournalEntryLineDetail",
			"Amount":      float64(1500),
			"Description": "Freight revenue",
			"JournalEntryLineDetail": map[string]any{
				"PostingType": "Credit",
				"AccountRef":  map[string]any{"value": "79"},
			},
		},
	}, body["Line"])
}

func TestJournalEntryCarriesItsCurrencyAndRate(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_journal_entry.json"))
	})
	txn := invoiceJournal()
	txn.CurrencyCode = "cad"
	txn.ExchangeRate = decimal.RequireFromString("0.7312")

	_, err := client.CreateJournalEntry(t.Context(), testRequestID, txn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"value": "CAD"}, body["CurrencyRef"])
	assert.InDelta(t, 0.7312, body["ExchangeRate"], 1e-9)
}

func TestUpdateJournalEntryReadsTheSyncTokenFirst(t *testing.T) {
	t.Parallel()

	var paths []string
	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodGet {
			_, _ = w.Write(fixture(t, "read_journal_entry.json"))
			return
		}
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_journal_entry.json"))
	})

	_, err := client.UpdateJournalEntry(t.Context(), testRequestID, "310", invoiceJournal())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"GET /v3/company/" + testRealm + "/journalentry/310",
		"POST /v3/company/" + testRealm + "/journalentry",
	}, paths)
	assert.Equal(t, "310", body["Id"])
	assert.Equal(t, "4", body["SyncToken"])
}

func TestJournalEntryRefusesWhatQuickBooksWould(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("an invalid journal entry must not reach QuickBooks")
	})

	unbalanced := invoiceJournal()
	unbalanced.Lines[1].Amount = decimal.RequireFromString("1499.99")
	oneLine := invoiceJournal()
	oneLine.Lines = oneLine.Lines[:1]
	noAccount := invoiceJournal()
	noAccount.Lines[1].AccountID = " "
	negative := invoiceJournal()
	negative.Lines[0].Amount = decimal.RequireFromString("-1500")
	negative.Lines[1].Amount = decimal.RequireFromString("-1500")
	badPosting := invoiceJournal()
	badPosting.Lines[0].PostingType = "Both"
	halfName := invoiceJournal()
	halfName.Lines[0].EntityID = ""
	wrongName := invoiceJournal()
	wrongName.Lines[0].EntityType = "Employee"

	cases := map[string]struct {
		txn  *quickbooks.JournalTxn
		want error
	}{
		"unbalanced":        {unbalanced, quickbooks.ErrJournalUnbalanced},
		"one line":          {oneLine, quickbooks.ErrJournalLinesTooFew},
		"no lines":          {&quickbooks.JournalTxn{}, quickbooks.ErrJournalLinesTooFew},
		"missing account":   {noAccount, quickbooks.ErrJournalAccount},
		"negative amount":   {negative, quickbooks.ErrNegativeAmount},
		"unknown posting":   {badPosting, quickbooks.ErrJournalPostingType},
		"name without id":   {halfName, quickbooks.ErrJournalEntityTarget},
		"unsupported names": {wrongName, quickbooks.ErrJournalEntityTarget},
	}
	for name, tc := range cases {
		_, err := client.CreateJournalEntry(t.Context(), testRequestID, tc.txn)
		require.ErrorIs(t, err, tc.want, name)
	}
	_, err := client.CreateJournalEntry(t.Context(), testRequestID, nil)
	require.ErrorIs(t, err, quickbooks.ErrJournalLinesTooFew)
}

func TestJournalEntryLinksToTheJournalPage(t *testing.T) {
	t.Parallel()

	assert.True(t, quickbooks.TxnJournalEntry.IsValid())
	assert.Equal(t, "/app/journal?txnId=", quickbooks.TxnJournalEntry.AppPath())
}

func TestTrialBalanceReadsEveryAccountRowAcrossSections(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/"+testRealm+"/reports/TrialBalance", r.URL.Path)
		assert.Equal(t, "Accrual", r.URL.Query().Get("accounting_method"))
		assert.Equal(t, "2026-04-01", r.URL.Query().Get("start_date"))
		assert.Equal(t, "2026-09-27", r.URL.Query().Get("end_date"))
		_, _ = w.Write(fixture(t, "report_trial_balance.json"))
	})

	rows, err := client.TrialBalance(t.Context(), &quickbooks.TrialBalanceRequest{
		StartDate: "2026-04-01",
		EndDate:   "2026-09-27",
	})
	require.NoError(t, err)
	require.Len(t, rows, 3, "the grand total row names no account")
	assert.Equal(t, "35", rows[0].AccountID)
	assert.True(t, decimal.RequireFromString("12480.50").Equal(rows[0].Debit), "thousands separators are read")
	assert.True(t, rows[0].Credit.IsZero())
	assert.Equal(t, "79", rows[2].AccountID, "rows nested in a section are read")
	assert.Equal(t, "Freight revenue", rows[2].AccountName)
	assert.True(t, decimal.RequireFromString("15680.50").Equal(rows[2].Credit))
}

func TestTrialBalanceNeedsAnEndDate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no request without an end date")
	})
	_, err := client.TrialBalance(t.Context(), &quickbooks.TrialBalanceRequest{StartDate: "2026-04-01"})
	require.ErrorIs(t, err, quickbooks.ErrReportDateRequired)
	_, err = client.TrialBalance(t.Context(), nil)
	require.ErrorIs(t, err, quickbooks.ErrReportDateRequired)
}

func TestJournalLineNamesAVendorOnPayables(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_journal_entry.json"))
	})
	txn := &quickbooks.JournalTxn{
		TxnDate: "2026-04-10",
		Lines: []quickbooks.JournalLine{
			{
				PostingType: quickbooks.PostingDebit,
				AccountID:   "50",
				Amount:      decimal.RequireFromString("1650"),
			},
			{
				PostingType: quickbooks.PostingCredit,
				AccountID:   "33",
				Amount:      decimal.RequireFromString("1650"),
				EntityType:  quickbooks.JournalEntityVendor,
				EntityID:    "91",
			},
		},
	}

	_, err := client.CreateJournalEntry(t.Context(), testRequestID, txn)
	require.NoError(t, err)
	lines := body["Line"].([]any)
	detail := lines[1].(map[string]any)["JournalEntryLineDetail"].(map[string]any)
	assert.Equal(t, map[string]any{
		"Type":      "Vendor",
		"EntityRef": map[string]any{"value": "91"},
	}, detail["Entity"])
	assert.NotContains(t, lines[0].(map[string]any)["JournalEntryLineDetail"], "Entity")
}

func TestDeleteJournalEntryReadsItThenDeletesIt(t *testing.T) {
	t.Parallel()

	var operation, path string
	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write(fixture(t, "read_journal_entry.json"))
			return
		}
		operation = r.URL.Query().Get("operation")
		path = r.URL.Path
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "delete_journal_entry.json"))
	})

	result, err := client.DeleteJournalEntry(t.Context(), testRequestID, "310")
	require.NoError(t, err)
	assert.Equal(t, "delete", operation)
	assert.Equal(t, "/v3/company/"+testRealm+"/journalentry", path)
	assert.Equal(t, "310", body["Id"])
	assert.NotEmpty(t, body["SyncToken"])
	assert.Equal(t, "Deleted", result.Status)
}
