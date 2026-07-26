package floatutils

import "math"

// IsFinite reports whether a float carries a usable number. Author-supplied
// values arrive as JSON floats, where NaN and the infinities are reachable and
// would otherwise poison every arithmetic result they touch.
func IsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func Clamp(value, minVal, maxVal float64) float64 {
	switch {
	case value < minVal:
		return minVal
	case value > maxVal:
		return maxVal
	default:
		return value
	}
}

func FloatValue(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func FloatPtrValue(v any) *float64 {
	if v == nil {
		return nil
	}
	converted := FloatValue(v)
	return &converted
}
