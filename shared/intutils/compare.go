package intutils

import "math"

// Max returns the larger of two values.
func Max[T Numeric](a, b T) T {
	if a > b {
		return a
	}

	return b
}

// Min returns the smaller of two values.
func Min[T Numeric](a, b T) T {
	if a < b {
		return a
	}

	return b
}

// Abs returns the distance from zero. Negating math.MinInt64 overflows and
// stays negative, so the minimum is returned unchanged.
func Abs(value int64) int64 {
	if value < 0 {
		if value == math.MinInt64 {
			return value
		}

		return -value
	}

	return value
}
