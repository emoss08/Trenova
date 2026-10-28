package gqldecimal

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func NullDecimalStringPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Decimal.String()
	return &formatted
}

func NullDecimalToStringPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	s := value.Decimal.String()
	return &s
}

func DecimalPtrToStringPtr(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	s := value.String()
	return &s
}

func NullDecimalFromStringPtr(value *string, field string) (decimal.NullDecimal, error) {
	if value == nil || *value == "" {
		return decimal.NullDecimal{}, nil
	}

	parsed, err := decimal.NewFromString(*value)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return decimal.NewNullDecimal(parsed), nil
}

func DecimalFromString(value, field string) (decimal.Decimal, error) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return parsed, nil
}

func NullDecimalToString(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	s := d.Decimal.String()
	return &s
}

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

func DecimalPtrString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	rendered := value.String()
	return &rendered
}

// nullDecimalValue unwraps an optional decimal into the pointer the domain
// models use for a column that may be absent. Its sibling nullDecimalPtr
// renders one for the wire, which is the opposite direction.
func NullDecimalValue(value decimal.NullDecimal) *decimal.Decimal {
	if !value.Valid {
		return nil
	}
	amount := value.Decimal
	return &amount
}

func NullDecimalPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	return stringutils.Ptr(value.Decimal.StringFixed(2))
}
