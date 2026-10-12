package common

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const FieldDescription = "description"

// decimalArg reads an amount written as a decimal string. It is never
// negative; a missing optional amount is zero.
func DecimalArg(params map[string]any, key string, required bool) (decimal.Decimal, error) {
	raw := strings.TrimSpace(OptionalString(params, key))
	if raw == "" {
		if required {
			return decimal.Zero, fmt.Errorf("missing required parameter %q", key)
		}

		return decimal.Zero, nil
	}

	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf(
			"parameter %q must be a decimal amount such as 1850.00, not %q", key, raw,
		)
	}
	if value.IsNegative() {
		return decimal.Zero, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return value, nil
}
