package accountingselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestJournalEntrySelectOptionItem_NamesTheEntryByNumberAndCarriesOnlyItsReferenceView(
	t *testing.T,
) {
	t.Parallel()

	entity := &journalentry.JournalEntry{
		ID:             pulid.MustNew("je_"),
		EntryNumber:    "JE-2026-000418",
		EntryType:      journalentry.EntryType("Standard"),
		Status:         journalentry.StatusPosted,
		AccountingDate: 1792065600,
		Description:    "Fuel accrual for October",
		TotalDebit:     125000,
		TotalCredit:    125000,
		CreatedAt:      1792000000,
	}

	item := journalEntrySelectOptionItem(entity)

	assert.Equal(t, entity.ID.String(), item.Option.ID)
	assert.Equal(t, "JE-2026-000418", item.Option.Label)
	assert.Nil(t, item.Option.Description)
	assert.Equal(t, map[string]any{
		"entryType":      "Standard",
		"status":         "Posted",
		"accountingDate": int64(1792065600),
	}, item.Option.Meta)
	assert.Equal(t, int64(1792000000), item.Cursor.At)
}
