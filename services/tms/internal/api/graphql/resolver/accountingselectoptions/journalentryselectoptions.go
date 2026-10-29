package accountingselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
)

func journalEntrySelectOptionItem(entity *journalentry.JournalEntry) selectoption.Item {
	return selectoption.ItemFor(
		journalEntrySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func journalEntrySelectOption(entity *journalentry.JournalEntry) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: entity.EntryNumber,
		Meta: map[string]any{
			"entryType":                string(entity.EntryType),
			selectoption.MetaKeyStatus: string(entity.Status),
			"accountingDate":           entity.AccountingDate,
		},
	}
}
