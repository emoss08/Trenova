package floatutils

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
