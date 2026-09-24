package domainvalidation

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	errBudgetNotDecimal = validation.NewError(
		"validation_budget_not_decimal",
		"A budget must be an amount",
	)
	errBudgetNegative = validation.NewError(
		"validation_budget_negative",
		"A budget cannot be negative",
	)
	errBudgetFractionalCents = validation.NewError(
		"validation_budget_fractional_cents",
		"A budget is in whole cents",
	)
)

func BudgetUSD(maximum decimal.Decimal, maximumLabel string) validation.Rule {
	tooLarge := validation.NewError(
		"validation_budget_too_large",
		"A budget is at most "+maximumLabel,
	)

	return validation.By(func(value any) error {
		var amount decimal.Decimal
		switch typed := value.(type) {
		case decimal.Decimal:
			amount = typed
		case *decimal.Decimal:
			if typed == nil {
				return nil
			}
			amount = *typed
		default:
			return errBudgetNotDecimal
		}

		switch {
		case amount.IsNegative():
			return errBudgetNegative
		case amount.GreaterThan(maximum):
			return tooLarge
		case !amount.Equal(amount.Round(2)):
			return errBudgetFractionalCents
		default:
			return nil
		}
	})
}
