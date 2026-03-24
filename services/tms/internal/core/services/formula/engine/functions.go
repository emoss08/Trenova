package engine

import (
	"errors"
	"fmt"
	"math"

	"github.com/expr-lang/expr"
)

func BuiltinFunctions() []expr.Option {
	return []expr.Option{
		expr.Function("round", roundFn,
			new(func(float64) float64),
			new(func(float64, int) float64),
		),
		expr.Function("ceil", ceilFn, new(func(float64) float64)),
		expr.Function("floor", floorFn, new(func(float64) float64)),
		expr.Function("abs", absFn, new(func(float64) float64)),
		expr.Function("min", minFn, new(func(float64, float64) float64)),
		expr.Function("max", maxFn, new(func(float64, float64) float64)),
		expr.Function("sum", sumFn, new(func(...float64) float64)),
		expr.Function("avg", avgFn, new(func(...float64) float64)),
		expr.Function("coalesce", coalesceFn, new(func(...any) any)),
		expr.Function("clamp", clampFn, new(func(float64, float64, float64) float64)),
		expr.Function("pow", powFn, new(func(float64, float64) float64)),
		expr.Function("sqrt", sqrtFn, new(func(float64) float64)),
	}
}

func roundFn(args ...any) (any, error) {
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
		decimals = d
	}

	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier, nil
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
	a, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	if a < b {
		return a, nil
	}
	return b, nil
}

func maxFn(args ...any) (any, error) {
	a, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	if a > b {
		return a, nil
	}
	return b, nil
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
	return nil, errors.New("no value found")
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
