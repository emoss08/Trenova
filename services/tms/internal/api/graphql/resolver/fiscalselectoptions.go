package resolver

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveFiscalYearSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.fiscalYearRepo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalYearSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.fiscalYearRepo.SelectOptions(
		ctx,
		&repositories.FiscalYearSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		fiscalYearSelectOptionItem,
	)
}

func (r *Resolver) resolveFiscalPeriodSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.fiscalPeriodRepo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fiscalPeriodSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.fiscalPeriodRepo.SelectOptions(
		ctx,
		&repositories.FiscalPeriodSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			FiscalYearID:       selectOptionIDFilter(req.filters, "fiscalYearId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
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
		Description: stringPtr(strconv.Itoa(entity.Year)),
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
