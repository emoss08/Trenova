package expression

import (
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/pkg/formula/conversion"
)

// * Function represents a callable function in expressions
type Function interface {
	// Name returns the function name
	Name() string

	// MinArgs returns the minimum number of arguments
	MinArgs() int

	// MaxArgs returns the maximum number of arguments (-1 for unlimited)
	MaxArgs() int

	// Call executes the function with given arguments
	Call(ctx *EvaluationContext, args ...any) (any, error)
}

// * FunctionRegistry maps function names to implementations
type FunctionRegistry map[string]Function

// * DefaultFunctionRegistry returns the standard function set
func DefaultFunctionRegistry() FunctionRegistry {
	registry := make(FunctionRegistry)

	// Math functions
	registry["abs"] = &absFunction{}
	registry["min"] = &minFunction{}
	registry["max"] = &maxFunction{}
	registry["round"] = &roundFunction{}
	registry["floor"] = &floorFunction{}
	registry["ceil"] = &ceilFunction{}
	registry["sqrt"] = &sqrtFunction{}
	registry["pow"] = &powFunction{}

	// Type conversion
	registry["number"] = &numberFunction{}
	registry["string"] = &stringFunction{}
	registry["bool"] = &boolFunction{}

	// Array functions
	registry["len"] = &lenFunction{}
	registry["sum"] = &sumFunction{}
	registry["avg"] = &avgFunction{}

	// Conditional
	registry["if"] = &ifFunction{}
	registry["coalesce"] = &coalesceFunction{}

	return registry
}

// * Built-in function implementations

type absFunction struct{}

func (f *absFunction) Name() string { return "abs" }
func (f *absFunction) MinArgs() int { return 1 }
func (f *absFunction) MaxArgs() int { return 1 }
func (f *absFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("abs: requires exactly 1 argument, got %d", len(args))
	}
	val, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("abs: argument must be a number")
	}
	return math.Abs(val), nil
}

type minFunction struct{}

func (f *minFunction) Name() string { return "min" }
func (f *minFunction) MinArgs() int { return 1 }
func (f *minFunction) MaxArgs() int { return -1 }
func (f *minFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("min: requires at least one argument")
	}

	minimum, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("min: all arguments must be numbers")
	}

	for i := 1; i < len(args); i++ {
		val, ok := conversion.ToFloat64(args[i])
		if !ok {
			return nil, fmt.Errorf("min: all arguments must be numbers")
		}
		if val < minimum {
			minimum = val
		}
	}

	return minimum, nil
}

type maxFunction struct{}

func (f *maxFunction) Name() string { return "max" }
func (f *maxFunction) MinArgs() int { return 1 }
func (f *maxFunction) MaxArgs() int { return -1 }
func (f *maxFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("max: requires at least one argument")
	}

	maximum, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("max: all arguments must be numbers")
	}

	for i := 1; i < len(args); i++ {
		val, ok := conversion.ToFloat64(args[i])
		if !ok {
			return nil, fmt.Errorf("max: all arguments must be numbers")
		}
		if val > maximum {
			maximum = val
		}
	}

	return maximum, nil
}

type roundFunction struct{}

func (f *roundFunction) Name() string { return "round" }
func (f *roundFunction) MinArgs() int { return 1 }
func (f *roundFunction) MaxArgs() int { return 2 }
func (f *roundFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("round: requires 1 or 2 arguments, got %d", len(args))
	}
	val, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("round: first argument must be a number")
	}

	precision := 0
	if len(args) > 1 {
		p, ok := conversion.ToFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("round: precision must be a number")
		}
		precision = int(p)
	}

	multiplier := math.Pow(10, float64(precision))
	return math.Round(val*multiplier) / multiplier, nil
}

type floorFunction struct{}

func (f *floorFunction) Name() string { return "floor" }
func (f *floorFunction) MinArgs() int { return 1 }
func (f *floorFunction) MaxArgs() int { return 1 }
func (f *floorFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor: requires exactly 1 argument, got %d", len(args))
	}
	val, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("floor: argument must be a number")
	}
	return math.Floor(val), nil
}

type ceilFunction struct{}

func (f *ceilFunction) Name() string { return "ceil" }
func (f *ceilFunction) MinArgs() int { return 1 }
func (f *ceilFunction) MaxArgs() int { return 1 }
func (f *ceilFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceil: requires exactly 1 argument, got %d", len(args))
	}
	val, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("ceil: argument must be a number")
	}
	return math.Ceil(val), nil
}

type sqrtFunction struct{}

func (f *sqrtFunction) Name() string { return "sqrt" }
func (f *sqrtFunction) MinArgs() int { return 1 }
func (f *sqrtFunction) MaxArgs() int { return 1 }
func (f *sqrtFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt: requires exactly 1 argument, got %d", len(args))
	}
	val, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("sqrt: argument must be a number")
	}
	if val < 0 {
		return nil, fmt.Errorf("sqrt: cannot take square root of negative number")
	}
	return math.Sqrt(val), nil
}

type powFunction struct{}

func (f *powFunction) Name() string { return "pow" }
func (f *powFunction) MinArgs() int { return 2 }
func (f *powFunction) MaxArgs() int { return 2 }
func (f *powFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow: requires exactly 2 arguments, got %d", len(args))
	}
	base, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("pow: base must be a number")
	}

	exp, ok := conversion.ToFloat64(args[1])
	if !ok {
		return nil, fmt.Errorf("pow: exponent must be a number")
	}

	result := math.Pow(base, exp)
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, fmt.Errorf("pow: result out of range")
	}

	return result, nil
}

// * Type conversion functions

type numberFunction struct{}

func (f *numberFunction) Name() string { return "number" }
func (f *numberFunction) MinArgs() int { return 1 }
func (f *numberFunction) MaxArgs() int { return 1 }
func (f *numberFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("number: requires exactly 1 argument, got %d", len(args))
	}
	
	// First try the conversion helper
	val, ok := conversion.ToFloat64(args[0])
	if ok {
		return val, nil
	}
	
	// Handle special cases
	switch v := args[0].(type) {
	case string:
		// Try to parse string as number
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		if err == nil {
			return f, nil
		}
		return nil, fmt.Errorf("number: cannot convert string %q to number", v)
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("number: cannot convert %T to number", args[0])
	}
}

type stringFunction struct{}

func (f *stringFunction) Name() string { return "string" }
func (f *stringFunction) MinArgs() int { return 1 }
func (f *stringFunction) MaxArgs() int { return 1 }
func (f *stringFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string: requires exactly 1 argument, got %d", len(args))
	}
	return fmt.Sprint(args[0]), nil
}

type boolFunction struct{}

func (f *boolFunction) Name() string { return "bool" }
func (f *boolFunction) MinArgs() int { return 1 }
func (f *boolFunction) MaxArgs() int { return 1 }
func (f *boolFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bool: requires exactly 1 argument, got %d", len(args))
	}
	return toBool(args[0]), nil
}

// * Array functions

type lenFunction struct{}

func (f *lenFunction) Name() string { return "len" }
func (f *lenFunction) MinArgs() int { return 1 }
func (f *lenFunction) MaxArgs() int { return 1 }
func (f *lenFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len: requires exactly 1 argument, got %d", len(args))
	}
	switch val := args[0].(type) {
	case string:
		return float64(len(val)), nil
	case []any:
		return float64(len(val)), nil
	default:
		return nil, fmt.Errorf("len: argument must be string or array")
	}
}

type sumFunction struct{}

func (f *sumFunction) Name() string { return "sum" }
func (f *sumFunction) MinArgs() int { return 1 }
func (f *sumFunction) MaxArgs() int { return -1 }
func (f *sumFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	sum := 0.0

	for _, arg := range args {
		if arr, ok := arg.([]any); ok {
			// If argument is an array, sum its elements
			for _, elem := range arr {
				val, ok := conversion.ToFloat64(elem)
				if !ok {
					return nil, fmt.Errorf("sum: all elements must be numbers")
				}
				sum += val
			}
		} else {
			// Otherwise treat as a number
			val, ok := conversion.ToFloat64(arg)
			if !ok {
				return nil, fmt.Errorf("sum: all arguments must be numbers or arrays")
			}
			sum += val
		}
	}

	return sum, nil
}

type avgFunction struct{}

func (f *avgFunction) Name() string { return "avg" }
func (f *avgFunction) MinArgs() int { return 1 }
func (f *avgFunction) MaxArgs() int { return -1 }
func (f *avgFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	sum := 0.0
	count := 0

	for _, arg := range args {
		if arr, ok := arg.([]any); ok {
			// If argument is an array, average its elements
			for _, elem := range arr {
				val, ok := conversion.ToFloat64(elem)
				if !ok {
					return nil, fmt.Errorf("avg: all elements must be numbers")
				}
				sum += val
				count++
			}
		} else {
			// Otherwise treat as a number
			val, ok := conversion.ToFloat64(arg)
			if !ok {
				return nil, fmt.Errorf("avg: all arguments must be numbers or arrays")
			}
			sum += val
			count++
		}
	}

	if count == 0 {
		return nil, fmt.Errorf("avg: cannot compute average of empty array")
	}

	return sum / float64(count), nil
}

// * Conditional functions

type ifFunction struct{}

func (f *ifFunction) Name() string { return "if" }
func (f *ifFunction) MinArgs() int { return 3 }
func (f *ifFunction) MaxArgs() int { return 3 }
func (f *ifFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("if: requires exactly 3 arguments, got %d", len(args))
	}
	condition := toBool(args[0])
	if condition {
		return args[1], nil
	}
	return args[2], nil
}

type coalesceFunction struct{}

func (f *coalesceFunction) Name() string { return "coalesce" }
func (f *coalesceFunction) MinArgs() int { return 1 }
func (f *coalesceFunction) MaxArgs() int { return -1 }
func (f *coalesceFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	for _, arg := range args {
		if arg != nil {
			// Also check for empty strings and zero values
			switch v := arg.(type) {
			case string:
				if v != "" {
					return v, nil
				}
			case float64:
				if v != 0 {
					return v, nil
				}
			case bool:
				return v, nil
			default:
				return arg, nil
			}
		}
	}
	return nil, nil
}
