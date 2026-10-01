package base

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func DecimalString(value decimal.Decimal) string {
	return value.StringFixed(2)
}

func ParseDecimalField(field, raw string, required bool) (decimal.Decimal, error) {
	if raw == "" {
		if required {
			return decimal.Zero, errortypes.NewValidationError(
				field,
				errortypes.ErrRequired,
				"Value is required",
			)
		}
		return decimal.Zero, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Value must be a number",
		)
	}
	return value, nil
}

func ParseNullDecimalField(field string, raw *string) (decimal.NullDecimal, error) {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}, nil
	}
	value, err := ParseDecimalField(field, *raw, true)
	if err != nil {
		return decimal.NullDecimal{}, err
	}
	return decimal.NewNullDecimal(value), nil
}
