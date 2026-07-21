package costingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type glActualRates struct {
	rates  map[costingcontrol.CategoryType]decimal.Decimal
	window *GLWindowInfo
}

func (s *Service) resolveGLActualRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *costingcontrol.CostingControl,
	asOf time.Time,
) (*glActualRates, error) {
	months := int(control.GLRollingMonths)
	from := asOf.AddDate(0, -months, 0)

	window := &GLWindowInfo{
		FromDate: from.Unix(),
		ToDate:   asOf.Unix(),
	}

	accountCategories, accountIDs := glActualAccountMappings(control)

	result := &glActualRates{
		rates:  make(map[costingcontrol.CategoryType]decimal.Decimal),
		window: window,
	}

	if len(accountIDs) == 0 {
		return result, nil
	}

	miles, err := s.actualsRepo.FleetMiles(ctx, &repositories.FleetMilesRequest{
		TenantInfo: tenantInfo,
		FromDate:   window.FromDate,
		ToDate:     window.ToDate,
	})
	if err != nil {
		s.l.Error("failed to load fleet miles for GL actuals", zap.Error(err))
		return nil, err
	}
	window.FleetMiles = miles.TotalMiles

	sums, err := s.actualsRepo.SumExpenseByAccounts(ctx, &repositories.SumExpenseByAccountsRequest{
		TenantInfo:   tenantInfo,
		GLAccountIDs: accountIDs,
		FromDate:     window.FromDate,
		ToDate:       window.ToDate,
	})
	if err != nil {
		s.l.Error("failed to sum GL expenses for actuals", zap.Error(err))
		return nil, err
	}

	expenseByCategory := sumExpensesByCategory(accountCategories, sums)

	for _, category := range control.Categories {
		if category == nil || !category.IsActive ||
			category.RateSource != costingcontrol.RateSourceGLActual {
			continue
		}

		expense, ok := expenseByCategory[category.Category]
		if !ok || expense.LessThanOrEqual(decimal.Zero) {
			continue
		}

		divisor := s.glDivisorMiles(control, category, miles, months)
		if divisor.LessThanOrEqual(decimal.Zero) {
			continue
		}

		result.rates[category.Category] = expense.Div(divisor)
		window.HasPostings = true
	}

	return result, nil
}

func glActualAccountMappings(
	control *costingcontrol.CostingControl,
) (map[pulid.ID][]costingcontrol.CategoryType, []pulid.ID) {
	accountCategories := make(map[pulid.ID][]costingcontrol.CategoryType)
	accountIDs := make([]pulid.ID, 0, len(control.Categories))
	for _, category := range control.Categories {
		if category == nil || !category.IsActive ||
			category.RateSource != costingcontrol.RateSourceGLActual {
			continue
		}
		for _, link := range category.GLAccounts {
			if link == nil {
				continue
			}
			if _, seen := accountCategories[link.GLAccountID]; !seen {
				accountIDs = append(accountIDs, link.GLAccountID)
			}
			accountCategories[link.GLAccountID] = append(
				accountCategories[link.GLAccountID],
				category.Category,
			)
		}
	}
	return accountCategories, accountIDs
}

func sumExpensesByCategory(
	accountCategories map[pulid.ID][]costingcontrol.CategoryType,
	sums map[pulid.ID]decimal.Decimal,
) map[costingcontrol.CategoryType]decimal.Decimal {
	expenseByCategory := make(map[costingcontrol.CategoryType]decimal.Decimal)
	for accountID, categories := range accountCategories {
		expense, ok := sums[accountID]
		if !ok || expense.IsZero() {
			continue
		}
		for _, category := range categories {
			expenseByCategory[category] = expenseByCategory[category].Add(expense)
		}
	}
	return expenseByCategory
}

func (s *Service) glDivisorMiles(
	control *costingcontrol.CostingControl,
	category *costingcontrol.CostCategory,
	miles *repositories.FleetMilesResult,
	months int,
) decimal.Decimal {
	fleetMiles := decimal.NewFromFloat(miles.TotalMiles)

	if category.CostBehavior != costingcontrol.CostBehaviorFixed ||
		control.PlannedMonthlyMiles == nil {
		return fleetMiles
	}

	plannedMiles := decimal.NewFromInt(*control.PlannedMonthlyMiles * int64(months))
	if plannedMiles.GreaterThan(fleetMiles) {
		return plannedMiles
	}

	return fleetMiles
}
