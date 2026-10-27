package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
)

func (r *Deps) resolveJournalEntrySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.JournalEntryService.GetEntriesByIDs(
			ctx,
			journalentryports.GetJournalEntriesByIDsRequest{
				TenantInfo: req.TenantInfo,
				IDs:        req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := make([]selectOptionConnectionItem, 0, len(entities))
		for _, entity := range entities {
			items = append(items, journalEntrySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.JournalEntryService.SelectOptions(
		ctx,
		&journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			ReversibleOnly:     selectOptionBoolFilter(req.Filters, "reversible"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		journalEntrySelectOptionItem,
	)
}

func journalEntrySelectOptionItem(entity *journalentry.JournalEntry) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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
			"entryType":      string(entity.EntryType),
			"status":         string(entity.Status),
			"accountingDate": entity.AccountingDate,
		},
	}
}
