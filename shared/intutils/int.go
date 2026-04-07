package intutils

import (
	"fmt"
	"math"
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
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	result := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
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
	default:
		return 0
	}
}
