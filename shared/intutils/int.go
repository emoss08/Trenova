package intutils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func SafeUint32ToUint8(value uint32) uint8 {
	if value > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(value) //nolint:gosec // safe: checked above
}

func SafeIntToUint32(value int) uint32 {
	if value > math.MaxUint32 || value < 0 {
		return 0
	}

	return uint32(value)
}

func SafeUint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("value %d is outside int64 range", value)
	}

	return int64(value), nil
}

func WithDefault(val, def int) int {
	if val == 0 {
		return def
	}

	return val
}

func Clamp[T ~int | ~int64](value, minVal, maxVal T) T {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

func AbsDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

func SafeShiftAmount(n, maxShift int) uint {
	if n < 0 {
		return 0
	}
	if n > maxShift {
		return uint(maxShift)
	}
	return uint(n)
}

func ClonePointer[T Numeric](value *T) *T {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func FormatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	digits := strings.TrimPrefix(s, "-")
	if len(digits) <= 3 {
		return s
	}

	result := make([]byte, 0, len(s)+(len(digits)-1)/3)
	if len(digits) != len(s) {
		result = append(result, '-')
	}
	for i := range len(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, digits[i])
	}

	return string(result)
}

func IntValue(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	default:
		return 0
	}
}

func IntPtrValue(v any) *int {
	switch value := v.(type) {
	case int:
		return &value
	case int32:
		converted := int(value)
		return &converted
	case int64:
		converted := int(value)
		return &converted
	case float64:
		converted := int(value)
		return &converted
	case float32:
		converted := int(value)
		return &converted
	default:
		return nil
	}
}

func SafeToInt32[T ~int | ~int64](value T) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}

	return int32(value)
}

func SafeToInt16[T ~int | ~int64](value T) int16 {
	if value > math.MaxInt16 {
		return math.MaxInt16
	}
	if value < math.MinInt16 {
		return math.MinInt16
	}

	return int16(value)
}
