package expression

import (
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/pkg/formula/conversion"
	"github.com/rotisserie/eris"
)

// Function represents a callable function in expressions
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

// FunctionRegistry maps function names to implementations
type FunctionRegistry map[string]Function

// DefaultFunctionRegistry returns the standard function set
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

	// Advanced math functions
	registry["log"] = &logFunction{}
	registry["exp"] = &expFunction{}
	registry["sin"] = &sinFunction{}
	registry["cos"] = &cosFunction{}
	registry["tan"] = &tanFunction{}

	// Type conversion
	registry["number"] = &numberFunction{}
	registry["string"] = &stringFunction{}
	registry["bool"] = &boolFunction{}

	// Array functions
	registry["len"] = &lenFunction{}
	registry["sum"] = &sumFunction{}
	registry["avg"] = &avgFunction{}
	registry["slice"] = &sliceFunction{}
	registry["concat"] = &concatFunction{}
	registry["contains"] = &containsFunction{}
	registry["indexOf"] = &indexOfFunction{}

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
		return nil, eris.New("abs argument must be a number")
	}
	return math.Abs(val), nil
}

type minFunction struct{}

func (f *minFunction) Name() string { return "min" }
func (f *minFunction) MinArgs() int { return 1 }
func (f *minFunction) MaxArgs() int { return -1 }
func (f *minFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) == 0 {
		return nil, eris.New("min: requires at least one argument")
	}

	minimum, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("min: all arguments must be numbers")
	}

	for i := 1; i < len(args); i++ {
		val, valOk := conversion.ToFloat64(args[i])
		if !valOk {
			return nil, eris.New("min: all arguments must be numbers")
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
		return nil, eris.New("max: requires at least one argument")
	}

	maximum, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("max: all arguments must be numbers")
	}

	for i := 1; i < len(args); i++ {
		val, valOk := conversion.ToFloat64(args[i])
		if !valOk {
			return nil, eris.New("max: all arguments must be numbers")
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
		return nil, eris.New("round: first argument must be a number")
	}

	precision := 0
	if len(args) > 1 {
		p, valOk := conversion.ToFloat64(args[1])
		if !valOk {
			return nil, eris.New("round: precision must be a number")
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
		return nil, eris.New("floor: argument must be a number")
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
		return nil, eris.New("ceil: argument must be a number")
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
		return nil, eris.New("sqrt: argument must be a number")
	}
	if val < 0 {
		return nil, eris.New("sqrt: cannot take square root of negative number")
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
		return nil, eris.New("pow: base must be a number")
	}

	exp, ok := conversion.ToFloat64(args[1])
	if !ok {
		return nil, eris.New("pow: exponent must be a number")
	}

	result := math.Pow(base, exp)
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, eris.New("pow: result out of range")
	}

	return result, nil
}

// * Advanced math functions

type logFunction struct{}

func (f *logFunction) Name() string { return "log" }
func (f *logFunction) MinArgs() int { return 1 }
func (f *logFunction) MaxArgs() int { return 2 }
func (f *logFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("log: requires 1 or 2 arguments, got %d", len(args))
	}

	x, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("log: first argument must be a number")
	}
	if x <= 0 {
		return nil, eris.New("log: argument must be positive")
	}

	// Natural logarithm if no base specified
	if len(args) == 1 {
		return math.Log(x), nil
	}

	// Logarithm with specified base
	base, ok := conversion.ToFloat64(args[1])
	if !ok {
		return nil, eris.New("log: base must be a number")
	}
	if base <= 0 || base == 1 {
		return nil, eris.New("log: base must be positive and not equal to 1")
	}

	return math.Log(x) / math.Log(base), nil
}

type expFunction struct{}

func (f *expFunction) Name() string { return "exp" }
func (f *expFunction) MinArgs() int { return 1 }
func (f *expFunction) MaxArgs() int { return 1 }
func (f *expFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("exp: requires exactly 1 argument, got %d", len(args))
	}

	x, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("exp: argument must be a number")
	}

	result := math.Exp(x)
	if math.IsInf(result, 0) {
		return nil, eris.New("exp: result out of range")
	}

	return result, nil
}

type sinFunction struct{}

func (f *sinFunction) Name() string { return "sin" }
func (f *sinFunction) MinArgs() int { return 1 }
func (f *sinFunction) MaxArgs() int { return 1 }
func (f *sinFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sin: requires exactly 1 argument, got %d", len(args))
	}

	x, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("sin: argument must be a number")
	}

	return math.Sin(x), nil
}

type cosFunction struct{}

func (f *cosFunction) Name() string { return "cos" }
func (f *cosFunction) MinArgs() int { return 1 }
func (f *cosFunction) MaxArgs() int { return 1 }
func (f *cosFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cos: requires exactly 1 argument, got %d", len(args))
	}

	x, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("cos: argument must be a number")
	}

	return math.Cos(x), nil
}

type tanFunction struct{}

func (f *tanFunction) Name() string { return "tan" }
func (f *tanFunction) MinArgs() int { return 1 }
func (f *tanFunction) MaxArgs() int { return 1 }
func (f *tanFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("tan: requires exactly 1 argument, got %d", len(args))
	}

	x, ok := conversion.ToFloat64(args[0])
	if !ok {
		return nil, eris.New("tan: argument must be a number")
	}

	return math.Tan(x), nil
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
		return nil, eris.New("len: argument must be string or array")
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
				val, valOk := conversion.ToFloat64(elem)
				if !valOk {
					return nil, eris.New("sum: all elements must be numbers")
				}
				sum += val
			}
		} else {
			// Otherwise treat as a number
			val, valOk := conversion.ToFloat64(arg)
			if !valOk {
				return nil, eris.New("sum: all arguments must be numbers or arrays")
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
				val, valOk := conversion.ToFloat64(elem)
				if !valOk {
					return nil, eris.New("avg: all elements must be numbers")
				}
				sum += val
				count++
			}
		} else {
			// Otherwise treat as a number
			val, valOk := conversion.ToFloat64(arg)
			if !valOk {
				return nil, eris.New("avg: all arguments must be numbers or arrays")
			}
			sum += val
			count++
		}
	}

	if count == 0 {
		return nil, eris.New("avg: cannot compute average of empty array")
	}

	return sum / float64(count), nil
}

type sliceFunction struct{}

func (f *sliceFunction) Name() string { return "slice" }
func (f *sliceFunction) MinArgs() int { return 2 }
func (f *sliceFunction) MaxArgs() int { return 3 }
func (f *sliceFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("slice: requires 2 or 3 arguments, got %d", len(args))
	}

	// Get the array or string
	var result []any
	switch val := args[0].(type) {
	case []any:
		result = val
	case string:
		// Convert string to array of characters
		runes := []rune(val)
		result = make([]any, len(runes))
		for i, r := range runes {
			result[i] = string(r)
		}
	default:
		return nil, eris.New("slice: first argument must be array or string")
	}

	// Get start index
	startFloat, ok := conversion.ToFloat64(args[1])
	if !ok {
		return nil, eris.New("slice: start index must be a number")
	}
	start := int(startFloat)

	// Get end index (default to length)
	end := len(result)
	if len(args) > 2 {
		endFloat, valOk := conversion.ToFloat64(args[2])
		if !valOk {
			return nil, eris.New("slice: end index must be a number")
		}
		end = int(endFloat)
	}

	// Handle negative indices
	if start < 0 {
		start = len(result) + start
	}
	if end < 0 {
		end = len(result) + end
	}

	// Bounds checking
	if start < 0 {
		start = 0
	}
	if end > len(result) {
		end = len(result)
	}
	if start > end {
		return []any{}, nil
	}

	// For strings, return a string
	if _, isString := args[0].(string); isString {
		chars := make([]string, end-start)
		for i := start; i < end; i++ {
			chars[i-start], _ = result[i].(string)
		}
		return joinStrings(chars, ""), nil
	}

	// For arrays, return a slice
	return result[start:end], nil
}

type concatFunction struct{}

func (f *concatFunction) Name() string { return "concat" }
func (f *concatFunction) MinArgs() int { return 2 }
func (f *concatFunction) MaxArgs() int { return -1 }
func (f *concatFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("concat: requires at least 2 arguments, got %d", len(args))
	}

	// Check if all arguments are strings
	allStrings := true
	for _, arg := range args {
		if _, ok := arg.(string); !ok {
			allStrings = false
			break
		}
	}

	if allStrings {
		// Concatenate strings
		result := make([]string, len(args))
		for i, arg := range args {
			result[i], _ = arg.(string)
		}
		return joinStrings(result, ""), nil
	}

	// Concatenate arrays
	result := []any{}
	for _, arg := range args {
		switch val := arg.(type) {
		case []any:
			result = append(result, val...)
		default:
			// Single values are treated as single-element arrays
			result = append(result, val)
		}
	}

	return result, nil
}

type containsFunction struct{}

func (f *containsFunction) Name() string { return "contains" }
func (f *containsFunction) MinArgs() int { return 2 }
func (f *containsFunction) MaxArgs() int { return 2 }
func (f *containsFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains: requires exactly 2 arguments, got %d", len(args))
	}

	// Handle string contains
	if str, ok := args[0].(string); ok {
		search, valOk := args[1].(string)
		if !valOk {
			return false, nil
		}
		return contains(str, search), nil
	}

	// Handle array contains
	arr, ok := args[0].([]any)
	if !ok {
		return nil, eris.New("contains: first argument must be string or array")
	}

	// Check if array contains the value
	for _, elem := range arr {
		if equal(elem, args[1]) {
			return true, nil
		}
	}

	return false, nil
}

type indexOfFunction struct{}

func (f *indexOfFunction) Name() string { return "indexOf" }
func (f *indexOfFunction) MinArgs() int { return 2 }
func (f *indexOfFunction) MaxArgs() int { return 2 }
func (f *indexOfFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("indexOf: requires exactly 2 arguments, got %d", len(args))
	}

	// Handle string indexOf
	if str, ok := args[0].(string); ok {
		search, valOk := args[1].(string)
		if !valOk {
			return -1.0, nil
		}
		return float64(indexOf(str, search)), nil
	}

	// Handle array indexOf
	arr, ok := args[0].([]any)
	if !ok {
		return nil, eris.New("indexOf: first argument must be string or array")
	}

	// Find index of value in array
	for i, elem := range arr {
		if equal(elem, args[1]) {
			return float64(i), nil
		}
	}

	return -1.0, nil
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
	return nil, eris.New("coalesce: all arguments must be non-nil")
}

// Helper functions for array operations
func equal(a, b any) bool {
	// Handle nil cases
	if a == nil || b == nil {
		return a == b
	}

	// Handle numeric comparison with type conversion
	aFloat, aOk := conversion.ToFloat64(a)
	bFloat, bOk := conversion.ToFloat64(b)
	if aOk && bOk {
		return aFloat == bFloat
	}

	// Handle string comparison
	aStr, aStrOk := a.(string)
	bStr, bStrOk := b.(string)
	if aStrOk && bStrOk {
		return aStr == bStr
	}

	// Handle bool comparison
	aBool, aBoolOk := a.(bool)
	bBool, bBoolOk := b.(bool)
	if aBoolOk && bBoolOk {
		return aBool == bBool
	}

	// Arrays require deep comparison
	aArr, aArrOk := a.([]any)
	bArr, bArrOk := b.([]any)
	if aArrOk && bArrOk {
		if len(aArr) != len(bArr) {
			return false
		}
		for i := range aArr {
			if !equal(aArr[i], bArr[i]) {
				return false
			}
		}
		return true
	}

	// Different types
	return false
}

func contains(str, substr string) bool {
	return strings.Contains(str, substr)
}

func indexOf(str, substr string) int {
	return strings.Index(str, substr)
}
