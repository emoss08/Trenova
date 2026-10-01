package selectoptionsresolver

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveFiscalYearSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FiscalYearRepo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalYearSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FiscalYearRepo.SelectOptions(
		ctx,
		&repositories.FiscalYearSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		fiscalYearSelectOptionItem,
	)
}

func (r *Deps) resolveFiscalPeriodSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FiscalPeriodRepo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalPeriodSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FiscalPeriodRepo.SelectOptions(
		ctx,
		&repositories.FiscalPeriodSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			FiscalYearID:       selectOptionIDFilter(req.Filters, "fiscalYearId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		fiscalPeriodSelectOptionItem,
	)
}

func fiscalYearSelectOptionItem(entity *fiscalyear.FiscalYear) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		fiscalYearSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fiscalYearSelectOption(entity *fiscalyear.FiscalYear) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(strconv.Itoa(entity.Year)),
		Meta: map[string]any{
			"year":      entity.Year,
			"status":    string(entity.Status),
			"isCurrent": entity.IsCurrent,
			"startDate": entity.StartDate,
			"endDate":   entity.EndDate,
		},
	}
}

func fiscalPeriodSelectOptionItem(entity *fiscalperiod.FiscalPeriod) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		fiscalPeriodSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fiscalPeriodSelectOption(entity *fiscalperiod.FiscalPeriod) *gqlmodel.SelectOption {
	label := entity.Name
	if label == "" {
		label = "Period " + strconv.Itoa(entity.PeriodNumber)
	}

	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: label,
		Meta: map[string]any{
			"fiscalYearId": entity.FiscalYearID.String(),
			"periodNumber": entity.PeriodNumber,
			"periodType":   string(entity.PeriodType),
			"status":       string(entity.Status),
			"isAdjusting":  entity.IsAdjusting,
			"startDate":    entity.StartDate,
			"endDate":      entity.EndDate,
		},
	}
}
