package qboconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const trialBalanceReport = `{
  "Columns": {"Column": [{"ColTitle": "", "ColType": "Account"}, {"ColTitle": "Debit"}, {"ColTitle": "Credit"}]},
  "Rows": {"Row": [
    {"ColData": [{"value": "Checking", "id": "35"}, {"value": "12,480.50"}, {"value": ""}]},
    {"type": "Section", "Rows": {"Row": [
      {"ColData": [{"value": "Freight revenue", "id": "79"}, {"value": ""}, {"value": "12480.50"}]}
    ]}}
  ]}
}`

func dailyJournal() *services.AccountingJournalDocument {
	return &services.AccountingJournalDocument{
		Auth:         auth(),
		RequestID:    docRequestID,
		Kind:         accountingsync.SyncObjectJournalSummary,
		DocNumber:    "TRN-2026-09-27",
		TxnDate:      "2026-09-27",
		CurrencyCode: "USD",
		PrivateNote:  "Trenova daily summary",
		Lines: []services.AccountingJournalLine{
			{
				Posting:           services.AccountingJournalDebit,
				AccountExternalID: "84",
				Amount:            decimal.NewFromInt(1500),
				Description:       "Accounts receivable",
				PartyKind:         services.AccountingJournalCustomer,
				PartyExternalID:   " 58 ",
			},
			{
				Posting:           services.AccountingJournalCredit,
				AccountExternalID: "79",
				Amount:            decimal.NewFromInt(1200),
				Description:       "Freight revenue",
			},
			{
				Posting:           services.AccountingJournalCredit,
				AccountExternalID: "33",
				Amount:            decimal.NewFromInt(300),
				Description:       "Accounts payable",
				PartyKind:         services.AccountingJournalVendor,
				PartyExternalID:   "91",
				},
		},
	}
}

func journalLines(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["Line"].([]any)
	require.True(t, ok)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		line, isMap := item.(map[string]any)
		require.True(t, isMap)
		out = append(out, line)
	}
	return out
}

func TestCreateJournalEntryPostsEachLineWithItsParty(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"JournalEntry":{"Id":"212","DocNumber":"TRN-2026-09-27","SyncToken":"0"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.CreateJournalEntry(t.Context(), dailyJournal())
	require.NoError(t, err)
	assert.Equal(t, "212", result.ExternalID)
	assert.Equal(t, "TRN-2026-09-27", result.DocNumber)
	assert.Equal(t, "212", result.Refs[accountingsync.ExternalRefDocument])
	assert.Equal(t, "JournalEntry", result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Contains(t, result.Refs[accountingsync.ExternalRefURL], "/app/journal?txnId=212")

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/journalentry", writes[0].Path)
	assert.Equal(t, docRequestID, writes[0].RequestID)
	assert.Equal(t, "TRN-2026-09-27", writes[0].Body["DocNumber"])
	assert.NotContains(t, writes[0].Body, "Id")

	lines := journalLines(t, writes[0].Body)
	require.Len(t, lines, 3)
	details := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		detail, ok := line["JournalEntryLineDetail"].(map[string]any)
		require.True(t, ok)
		details = append(details, detail)
	}
	assert.Equal(t, "Debit", details[0]["PostingType"])
	assert.Equal(t, map[string]any{"value": "84"}, details[0]["AccountRef"])
	assert.Equal(t, map[string]any{"Type": "Customer", "EntityRef": map[string]any{"value": "58"}}, details[0]["Entity"])
	assert.Equal(t, "Credit", details[1]["PostingType"])
	assert.NotContains(t, details[1], "Entity")
	assert.Equal(t, map[string]any{"Type": "Vendor", "EntityRef": map[string]any{"value": "91"}}, details[2]["Entity"])
}

func TestUpdateJournalEntryRewritesTheSameEntry(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{"JournalEntry":{"Id":"212","SyncToken":"3"}}`
		}
		return http.StatusOK, `{"JournalEntry":{"Id":"212","DocNumber":"TRN-2026-09-27","SyncToken":"4"}}`
	}}
	conn := documentConnector(t, fake)

	doc := dailyJournal()
	doc.ExternalID = " 212 "
	result, err := conn.UpdateJournalEntry(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, "212", result.ExternalID)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/journalentry", writes[0].Path)
	assert.Equal(t, "212", writes[0].Body["Id"])
	assert.Equal(t, "3", writes[0].Body["SyncToken"])
}

func TestJournalEntryRefusals(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(recordedCall) (int, string) {
		return http.StatusOK, `{"JournalEntry":{"Id":"1"}}`
	}}
	conn := documentConnector(t, fake)

	wrongKind := dailyJournal()
	wrongKind.Kind = accountingsync.SyncObjectInvoice
	_, err := conn.CreateJournalEntry(t.Context(), wrongKind)
	require.ErrorIs(t, err, errDocumentKind)
	_, err = conn.UpdateJournalEntry(t.Context(), wrongKind)
	require.ErrorIs(t, err, errDocumentKind)

	_, err = conn.UpdateJournalEntry(t.Context(), dailyJournal())
	require.ErrorIs(t, err, errExternalIDRequired)

	badPosting := dailyJournal()
	badPosting.Lines[1].Posting = services.AccountingJournalPosting("Both")
	_, err = conn.CreateJournalEntry(t.Context(), badPosting)
	require.ErrorIs(t, err, errJournalPosting)

	badParty := dailyJournal()
	badParty.Lines[0].PartyKind = services.AccountingJournalPartyKind("Employee")
	_, err = conn.CreateJournalEntry(t.Context(), badParty)
	require.ErrorIs(t, err, errJournalParty)

	assert.Empty(t, fake.writes())
}

func TestReadTrialBalanceReadsEveryAccountRow(t *testing.T) {
	t.Parallel()

	var query map[string]string
	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		query = map[string]string{"path": call.Path}
		return http.StatusOK, trialBalanceReport
	}}
	conn := documentConnector(t, fake)

	rows, err := conn.ReadTrialBalance(t.Context(), &services.ReadTrialBalanceRequest{
		Auth:      auth(),
		StartDate: "2026-04-01",
		EndDate:   "2026-09-27",
	})
	require.NoError(t, err)
	assert.Equal(t, "/reports/TrialBalance", query["path"])
	require.Len(t, rows, 2)
	assert.Equal(t, "35", rows[0].AccountExternalID)
	assert.Equal(t, "Checking", rows[0].AccountName)
	assert.True(t, decimal.RequireFromString("12480.50").Equal(rows[0].Net()))
	assert.Equal(t, "79", rows[1].AccountExternalID)
	assert.True(t, decimal.RequireFromString("-12480.50").Equal(rows[1].Net()))
}

func TestDeleteJournalEntryRetiresTheProviderEntry(t *testing.T) {
	t.Parallel()

	fake := &fakeQBO{respond: func(call recordedCall) (int, string) {
		if call.Method == http.MethodGet {
			return http.StatusOK, `{"JournalEntry":{"Id":"212","SyncToken":"5"}}`
		}
		return http.StatusOK, `{"JournalEntry":{"Id":"212","status":"Deleted"}}`
	}}
	conn := documentConnector(t, fake)

	result, err := conn.DeleteJournalEntry(t.Context(), &services.AccountingDocumentRef{
		Auth:       auth(),
		RequestID:  docRequestID,
		Kind:       accountingsync.SyncObjectJournalSummary,
		ExternalID: " 212 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "212", result.ExternalID)

	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/journalentry", writes[0].Path)
	assert.Equal(t, "delete", writes[0].Operation)
	assert.Equal(t, "5", writes[0].Body["SyncToken"])

	_, err = conn.DeleteJournalEntry(t.Context(), &services.AccountingDocumentRef{
		Auth: auth(), RequestID: docRequestID, Kind: accountingsync.SyncObjectInvoice, ExternalID: "1",
	})
	require.ErrorIs(t, err, errDocumentKind)
	_, err = conn.DeleteJournalEntry(t.Context(), &services.AccountingDocumentRef{
		Auth: auth(), RequestID: docRequestID, Kind: accountingsync.SyncObjectJournalEntry,
	})
	require.ErrorIs(t, err, errExternalIDRequired)
	assert.Len(t, fake.writes(), 1)
}
