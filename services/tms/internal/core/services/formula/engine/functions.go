package engine

import (
	"fmt"
	"math"

	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/expr-lang/expr"
	"github.com/shopspring/decimal"
)

func BuiltinFunctions() []expr.Option {
	options := make([]expr.Option, 0, len(functionSpecs))
	for _, spec := range functionSpecs {
		if spec.fn == nil {
			continue
		}
		options = append(options, expr.Function(spec.Name, spec.fn, spec.types...))
	}
	return options
}

const maxRoundDecimals = 12

// roundWith rounds through the decimal type rather than float arithmetic, so
// round(2.675, 2) is 2.68 the way a person expects and not the 2.67 that
// binary floating point produces. Every rounding function shares it; only
// the mode differs.
func roundWith(mode ratetypes.RoundingMode, args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	decimals := 0
	if len(args) > 1 {
		d, dErr := toInt(args[1])
		if dErr != nil {
			return nil, dErr
		}
		if d < -maxRoundDecimals || d > maxRoundDecimals {
			return nil, fmt.Errorf(
				"round decimals must be between %d and %d, got %d",
				-maxRoundDecimals, maxRoundDecimals, d,
			)
		}
		decimals = d
	}

	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("cannot round a non-finite value: %v", value)
	}

	return mode.Round(decimal.NewFromFloat(value), int32(decimals)).InexactFloat64(), nil
}

func roundFn(args ...any) (any, error) {
	return roundWith(ratetypes.RoundingModeHalfUp, args...)
}

func roundUpFn(args ...any) (any, error) {
	return roundWith(ratetypes.RoundingModeUp, args...)
}

func roundDownFn(args ...any) (any, error) {
	return roundWith(ratetypes.RoundingModeDown, args...)
}

func roundHalfEvenFn(args ...any) (any, error) {
	return roundWith(ratetypes.RoundingModeHalfEven, args...)
}

// roundToFn rounds to the nearest multiple of an increment: the nearest $5,
// the nearest quarter, the nearest hundredweight.
func roundToFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	increment, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	if increment <= 0 || math.IsNaN(increment) || math.IsInf(increment, 0) {
		return nil, fmt.Errorf("roundTo increment must be a positive number, got %v", increment)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("cannot round a non-finite value: %v", value)
	}

	step := decimal.NewFromFloat(increment)
	multiples := decimal.NewFromFloat(value).Div(step).Round(0)

	return multiples.Mul(step).InexactFloat64(), nil
}

func ceilFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Ceil(value), nil
}

func floorFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Floor(value), nil
}

func absFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Abs(value), nil
}

func minFn(args ...any) (any, error) {
	return extremum("min", args, func(candidate, best float64) bool { return candidate < best })
}

func maxFn(args ...any) (any, error) {
	return extremum("max", args, func(candidate, best float64) bool { return candidate > best })
}

func extremum(name string, args []any, better func(candidate, best float64) bool) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s requires at least one argument", name)
	}
	best, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	for _, arg := range args[1:] {
		candidate, convErr := toFloat64(arg)
		if convErr != nil {
			return nil, convErr
		}
		if better(candidate, best) {
			best = candidate
		}
	}
	return best, nil
}

func sumFn(args ...any) (any, error) {
	var total float64
	for _, arg := range args {
		v, err := toFloat64(arg)
		if err != nil {
			return nil, err
		}
		total += v
	}
	return total, nil
}

func avgFn(args ...any) (any, error) {
	if len(args) == 0 {
		return 0.0, nil
	}
	sum, err := sumFn(args...)
	if err != nil {
		return nil, err
	}
	return sum.(float64) / float64( //nolint:errcheck // ignore error because we know the type is correct
		len(args),
	), nil
}

func coalesceFn(args ...any) (any, error) {
	for _, v := range args {
		if v != nil {
			return v, nil
		}
	}
	return nil, nil //nolint:nilnil // coalesce of all-nil arguments is nil by contract
}

func clampFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	minVal, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	maxVal, err := toFloat64(args[2])
	if err != nil {
		return nil, err
	}
	if value < minVal {
		return minVal, nil
	}
	if value > maxVal {
		return maxVal, nil
	}
	return value, nil
}

func powFn(args ...any) (any, error) {
	base, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	exponent, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	return math.Pow(base, exponent), nil
}

func sqrtFn(args ...any) (any, error) {
	value, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	if value < 0 {
		return nil, fmt.Errorf("sqrt of negative number %v", value)
	}
	return math.Sqrt(value), nil
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case int32:
		return int(val), nil
	case float64:
		return int(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}
