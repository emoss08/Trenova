package intelkit

import (
	"math"
	"strings"

	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	fractionCeiling  = 1
	percentScale     = 100
	roundingDecimals = 10_000
)

func Text(v *jsonflex.String) string {
	return v.Value()
}

func Int(v *jsonflex.Int) *int {
	if v == nil {
		return nil
	}
	value := int(v.Value())
	return &value
}

func Int64(v *jsonflex.Int) *int64 {
	return v.Ptr()
}

func IntFromFloat(v *jsonflex.Float) *int {
	if v == nil {
		return nil
	}
	value := int(math.Round(v.Value()))
	return &value
}

func Float(v *jsonflex.Float) *float64 {
	if v == nil {
		return nil
	}
	value := round(v.Value())
	return &value
}

func NonNegativeFloat(v *jsonflex.Float) *float64 {
	if v == nil || v.Value() < 0 {
		return nil
	}
	return Float(v)
}

func Percent(v *jsonflex.Float) *float64 {
	if v == nil || v.Value() < 0 {
		return nil
	}
	value := v.Value()
	if value <= fractionCeiling {
		value *= percentScale
	}
	value = round(value)
	return &value
}

func FractionPercent(v *jsonflex.Float) *float64 {
	if v == nil || v.Value() < 0 {
		return nil
	}
	value := round(v.Value() * percentScale)
	return &value
}

func Bool(v *jsonflex.Bool) *bool {
	return v.Ptr()
}

func Unix(v *jsonflex.Time) *int64 {
	return v.Unix()
}

func Decimal(v *jsonflex.Float) *decimal.Decimal {
	if v == nil {
		return nil
	}
	value := decimal.NewFromFloat(v.Value())
	return &value
}

func Truthy(v *jsonflex.String) bool {
	text := strings.TrimSpace(v.Value())
	if text == "" {
		return false
	}
	switch strings.ToUpper(text) {
	case "Y", "T", "YES", "TRUE":
		return true
	case "N", "F", "NO", "NONE", "FALSE", "NULL", "N/A", "NA":
		return false
	}
	if parsed, ok := stringutils.ParseBool(text); ok {
		return parsed
	}
	return true
}

func SumInts(values ...*jsonflex.Int) *int {
	var total *int
	for _, value := range values {
		if value == nil {
			continue
		}
		if total == nil {
			total = new(int)
		}
		*total += int(value.Value())
	}
	return total
}

func round(value float64) float64 {
	return math.Round(value*roundingDecimals) / roundingDecimals
}
