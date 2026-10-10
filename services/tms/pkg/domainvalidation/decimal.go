package domainvalidation

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var errNotDecimal = errors.New("must be an amount")

func DecimalAtLeast(minimum decimal.Decimal, message string) validation.Rule {
	return decimalBound(func(amount decimal.Decimal) bool { return amount.GreaterThanOrEqual(minimum) }, message)
}

func DecimalAtMost(maximum decimal.Decimal, message string) validation.Rule {
	return decimalBound(func(amount decimal.Decimal) bool { return amount.LessThanOrEqual(maximum) }, message)
}

func decimalBound(within func(decimal.Decimal) bool, message string) validation.Rule {
	return validation.By(func(value any) error {
		amount, present, err := decimalOf(value)
		if err != nil || !present {
			return err
		}
		if !within(amount) {
			return errors.New(message)
		}

		return nil
	})
}

func decimalOf(value any) (decimal.Decimal, bool, error) {
	switch typed := value.(type) {
	case decimal.Decimal:
		return typed, true, nil
	case *decimal.Decimal:
		if typed == nil {
			return decimal.Zero, false, nil
		}
		return *typed, true, nil
	case decimal.NullDecimal:
		return typed.Decimal, typed.Valid, nil
	case *decimal.NullDecimal:
		if typed == nil {
			return decimal.Zero, false, nil
		}
		return typed.Decimal, typed.Valid, nil
	default:
		return decimal.Zero, false, errNotDecimal
	}
}
