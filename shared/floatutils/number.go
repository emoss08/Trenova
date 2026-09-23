package floatutils

import (
	"strconv"
	"strings"
)

var numberNoise = strings.NewReplacer(",", "", "$", "", "%", "", " ", "")

func ParseNumber(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, IsFinite(value)
	case float32:
		return float64(value), IsFinite(float64(value))
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	case string:
		cleaned := numberNoise.Replace(strings.TrimSpace(value))
		if cleaned == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return 0, false
		}

		return parsed, IsFinite(parsed)
	case interface{ String() string }:
		return ParseNumber(value.String())
	default:
		return 0, false
	}
}
