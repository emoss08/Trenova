package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func postedManualJournal() *manualjournal.Request {
	account := &glaccount.GLAccount{ID: pulid.MustNew("gla_"), AccountCode: "6100", Name: "Fuel"}
	entity := &manualjournal.Request{
		ID:            pulid.MustNew("mjr_"),
		RequestNumber: "MJR-12",
		Status:        manualjournal.StatusApproved,
		Description:   "Accrue fuel",
		Reason:        "Card statement arrives late",
		CurrencyCode:  "USD",
		Lines: []*manualjournal.Line{
			{
				LineNumber:  1,
				GLAccountID: account.ID,
				GLAccount:   account,
				Description: "Fuel",
				DebitAmount: 2500,
			},
			{
				LineNumber:   2,
				GLAccountID:  pulid.MustNew("gla_"),
				Description:  "Accrual",
				CreditAmount: 2500,
			},
		},
	}
	entity.SyncTotals()

	return entity
}

func TestListManualJournals_NarrowsByStatusAndGatesAmounts(t *testing.T) {
	t.Parallel()

	entity := postedManualJournal()
	repo := mocks.NewMockManualJournalRepository(t)
	var captured *repositories.ListManualJournalRequest
	repo.EXPECT().List(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *repositories.ListManualJournalRequest,
		) (*pagination.ListResult[*manualjournal.Request], error) {
			captured = req
			return &pagination.ListResult[*manualjournal.Request]{
				Items: []*manualjournal.Request{entity},
				Total: 1,
			}, nil
		},
	)
	tool := newListManualJournalsTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Approved"},
	), permission.SensitivityInternal))
	require.NoError(t, err)

	require.NotNil(t, captured)
	require.Len(t, captured.Filter.FieldFilters, 1)
	assert.Equal(t, "status", captured.Filter.FieldFilters[0].Field)
	outcome := result.(*gatedOutcome)
	rows := outcome.Items.([]any)
	require.Len(t, rows, 1)
	row := rows[0].(manualJournalRow)
	assert.Equal(t, "MJR-12", row.RequestNumber)
	assert.True(t, row.Balanced)
	assert.Empty(t, row.TotalDebit)
	assert.Contains(t, outcome.Withheld, "totalDebit")
	assert.Equal(t, permission.ResourceManualJournal, tool.Policy().Resource)
}

func TestGetManualJournal_ShowsLinesWithTheirAccounts(t *testing.T) {
	t.Parallel()

	entity := postedManualJournal()
	tool := &getManualJournalTool{
		journals: fakeManualJournalReader{entity: entity},
		access:   newFieldAccess(&fakePermissions{}),
	}

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{paramManualJournalID: entity.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(manualJournalView)
	require.Len(t, view.Lines, 2)
	assert.Equal(t, "6100", view.Lines[0].AccountCode)
	assert.Equal(t, "25.00", view.Lines[0].Debit)
	assert.Equal(t, "25.00", view.Lines[1].Credit)
	assert.Equal(t, "25.00", view.TotalDebit)
	assert.Equal(t, "Card statement arrives late", view.Reason)
}

type fakeManualJournalReader struct {
	entity *manualjournal.Request
}

func (f fakeManualJournalReader) GetByID(
	context.Context,
	repositories.GetManualJournalByIDRequest,
) (*manualjournal.Request, error) {
	return f.entity, nil
}

func TestListJournalReversals_ListsWhatEachReverses(t *testing.T) {
	t.Parallel()

	reversal := &journalreversal.Reversal{
		ID:                     pulid.MustNew("jrev_"),
		OriginalJournalEntryID: pulid.MustNew("je_"),
		Status:                 journalreversal.StatusApproved,
		ReasonCode:             "Duplicate",
		ReasonText:             "Entered twice",
	}
	repo := mocks.NewMockJournalReversalRepository(t)
	repo.EXPECT().List(mock.Anything, mock.Anything).Return(
		&pagination.ListResult[*journalreversal.Reversal]{
			Items: []*journalreversal.Reversal{reversal},
			Total: 1,
		}, nil,
	)
	tool := newListJournalReversalsTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{},
		permission.SensitivityRestricted))
	require.NoError(t, err)

	rows := result.(*gatedOutcome).Items.([]any)
	require.Len(t, rows, 1)
	row := rows[0].(journalReversalRow)
	assert.Equal(t, reversal.OriginalJournalEntryID.String(), row.OriginalJournalEntryID)
	assert.Equal(t, "Entered twice", row.ReasonText)
}
