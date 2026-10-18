package base

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

var maxMinorDecimal = decimal.New(1, 15)

func MinorString(minor int64) string {
	return money.DecimalFromMinor(minor).StringFixed(2)
}

func ParseMinorPtr(field string, raw *string) (*int64, error) {
	if raw == nil {
		return nil, nil //nolint:nilnil // an absent value leaves the setting as it is
	}

	value, err := ParseDecimalField(field, *raw, true)
	if err != nil {
		return nil, err
	}
	if value.Abs().GreaterThan(maxMinorDecimal) {
		return nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Value is too large",
		)
	}
	if !value.Equal(value.Round(2)) {
		return nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Value cannot be more precise than a cent",
		)
	}

	minor := money.MinorUnits(value)

	return &minor, nil
}
