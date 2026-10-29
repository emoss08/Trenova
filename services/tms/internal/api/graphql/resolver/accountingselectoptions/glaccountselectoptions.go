package accountingselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accounttypeservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accounttypeports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	FiscalYearRepo      fiscalyearports.FiscalYearRepository
	FiscalPeriodRepo    fiscalperiodports.FiscalPeriodRepository
	GlAccountRepo       glaccountports.GLAccountRepository
	JournalEntryService *journalentryservice.Service
	AccountTypeService  *accounttypeservice.Service
}

func (r *Sources) GLAccount(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.GlAccountRepo.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
			TenantInfo:   req.TenantInfo,
			GLAccountIDs: req.IDs,
		})
		if err != nil {
			return nil, err
		}

		items := make([]selectoption.Item, 0, len(entities))
		for _, entity := range entities {
			items = append(items, glAccountSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.GlAccountRepo.SelectOptions(
		ctx,
		&glaccountports.GLAccountSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		glAccountSelectOptionItem,
	)
}

func glAccountSelectOptionItem(entity *glaccount.GLAccount) selectoption.Item {
	return selectoption.ItemFor(
		glAccountSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func glAccountSelectOption(entity *glaccount.GLAccount) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.AccountCode,
		Description: &description,
	}
}

func (r *Sources) AccountType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.AccountTypeService.Get(
				ctx,
				accounttypeports.GetAccountTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accountTypeSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.AccountTypeService.SelectOptions(
		ctx,
		&accounttypeports.AccountTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		accountTypeSelectOptionItem,
	)
}

func accountTypeSelectOptionItem(entity *accounttype.AccountType) selectoption.Item {
	return selectoption.ItemFor(
		accountTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func accountTypeSelectOption(entity *accounttype.AccountType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Name),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
			"color":                  entity.Color,
			"name":                   entity.Name,
		},
	}
}

func (r *Sources) FiscalYear(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FiscalYearRepo.GetByID(ctx, fiscalyearports.GetFiscalYearByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalYearSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FiscalYearRepo.SelectOptions(
		ctx,
		&fiscalyearports.FiscalYearSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		fiscalYearSelectOptionItem,
	)
}

func (r *Sources) FiscalPeriod(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FiscalPeriodRepo.GetByID(
				ctx,
				fiscalperiodports.GetFiscalPeriodByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalPeriodSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FiscalPeriodRepo.SelectOptions(
		ctx,
		&fiscalperiodports.FiscalPeriodSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			FiscalYearID:       selectoption.IDFilter(req.Filters, "fiscalYearId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		fiscalPeriodSelectOptionItem,
	)
}

func (r *Sources) JournalEntry(
	ctx context.Context,
	req *selectoption.Request,
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

		items := make([]selectoption.Item, 0, len(entities))
		for _, entity := range entities {
			items = append(items, journalEntrySelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.JournalEntryService.SelectOptions(
		ctx,
		&journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			ReversibleOnly:     selectoption.BoolFilter(req.Filters, "reversible"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		journalEntrySelectOptionItem,
	)
}
