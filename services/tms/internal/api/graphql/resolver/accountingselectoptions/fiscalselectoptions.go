package accountingselectoptions

import (
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/shared/stringutils"
)

func fiscalYearSelectOptionItem(entity *fiscalyear.FiscalYear) selectoption.Item {
	return selectoption.ItemFor(
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
			"year":                     entity.Year,
			selectoption.MetaKeyStatus: string(entity.Status),
			"isCurrent":                entity.IsCurrent,
			"startDate":                entity.StartDate,
			"endDate":                  entity.EndDate,
		},
	}
}

func fiscalPeriodSelectOptionItem(entity *fiscalperiod.FiscalPeriod) selectoption.Item {
	return selectoption.ItemFor(
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
			"fiscalYearId":             entity.FiscalYearID.String(),
			"periodNumber":             entity.PeriodNumber,
			"periodType":               string(entity.PeriodType),
			selectoption.MetaKeyStatus: string(entity.Status),
			"isAdjusting":              entity.IsAdjusting,
			"startDate":                entity.StartDate,
			"endDate":                  entity.EndDate,
		},
	}
}
