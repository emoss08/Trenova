package nsconnector

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDocumentsReadsTotalsBalancesAndVoids(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("t.id IN (") {
			return reply{status: http.StatusOK, body: rowsJSON(t, false,
				map[string]any{
					"id": testInvoice, "tranid": "INV-1001", "foreigntotal": "1250.00",
					"foreignamountremaining": "250.00", "voided": "F", "currency": "usd",
					"lastmodified": "2026-10-13 14:04:05",
				},
				map[string]any{
					"id": "1205", "tranid": "INV-1005", "foreigntotal": "400.00",
					"foreignamountremaining": "0", "voided": "T", "currency": "USD",
				},
			)}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	assert.Equal(t, 500, conn.DocumentReadLimits().MaxPerRead)
	states, err := conn.ReadDocuments(
		t.Context(),
		&accountingsyncports.ReadAccountingDocumentsRequest{
			Auth: testAuth(),
			Kind: accountingsync.SyncObjectInvoice,
			Targets: []accountingsyncports.AccountingDocumentTarget{
				{ExternalID: testInvoice},
				{ExternalID: "1205"},
				{ExternalID: "1299"},
				{ExternalID: "eid:junk"},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, states, 4)

	first := states[0]
	assert.True(t, first.Found)
	assert.False(t, first.Voided)
	assert.Equal(t, "INV-1001", first.DocNumber)
	assert.True(t, first.Total.Equal(decimal.NewFromInt(1250)))
	require.NotNil(t, first.Balance)
	assert.True(t, first.Balance.Equal(decimal.NewFromInt(250)))
	assert.Equal(t, "USD", first.CurrencyCode)
	assert.Positive(t, first.ModifiedAt)
	assert.True(t, states[1].Voided)
	assert.False(t, states[2].Found)
	assert.False(t, states[3].Found)

	call, _ := fake.query("t.id IN (")
	assert.Equal(t, []any{float64(1201), float64(1205), float64(1299)}, call.Params)
}

func TestReadDocumentsBatchesFiveHundredIDsAQuery(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	targets := make([]accountingsyncports.AccountingDocumentTarget, 0, 650)
	for idx := range 650 {
		targets = append(targets, accountingsyncports.AccountingDocumentTarget{
			ExternalID: strconv.Itoa(1000 + idx),
		})
	}
	states, err := conn.ReadDocuments(
		t.Context(),
		&accountingsyncports.ReadAccountingDocumentsRequest{
			Auth:    testAuth(),
			Kind:    accountingsync.SyncObjectCarrierBill,
			Targets: targets,
		},
	)
	require.NoError(t, err)
	assert.Len(t, states, 650)
	calls := fake.all()
	require.Len(t, calls, 2)
	assert.Len(t, calls[0].Params, 500)
	assert.Len(t, calls[1].Params, 150)
	assert.Equal(t, 500, strings.Count(calls[0].SQL, "?"))
}

func TestReadDocumentsSkipsCreditApplications(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	states, err := conn.ReadDocuments(
		t.Context(),
		&accountingsyncports.ReadAccountingDocumentsRequest{
			Auth:    testAuth(),
			Kind:    accountingsync.SyncObjectCreditApplication,
			Targets: []accountingsyncports.AccountingDocumentTarget{{ExternalID: testCreditMemo}},
		},
	)
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.False(t, states[0].Found)
	assert.Empty(t, fake.all())

	_, err = conn.ReadDocuments(t.Context(), &accountingsyncports.ReadAccountingDocumentsRequest{
		Auth: testAuth(),
		Kind: accountingsync.SyncObjectJournalEntry,
	})
	require.ErrorIs(t, err, errDocumentKind)
}
