// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package intutils

import "fmt"

const (
	maxInt16  = 32767
	minInt16  = -32768
	maxInt32  = 2147483647
	minInt32  = -2147483648
	maxUInt16 = 65535
	minUInt16 = 0
)

// SafeInt16Ptr converts an int to a *int16.
// If the input is 0 and returnNilOnZero is true, it returns nil.
// Otherwise, it returns a pointer to the converted int16 value.
func SafeInt16Ptr(i int, returnNilOnZero bool) *int16 {
	if i == 0 && returnNilOnZero {
		return nil
	}

	// Check for int16 bounds to prevent overflow
	if i > 32767 || i < -32768 {
		// Return max or min int16 value as appropriate
		var val int16
		if i > 32767 {
			val = 32767 // max int16
		} else {
			val = -32768 // min int16
		}
		return &val
	}

	i16 := int16(i)
	return &i16
}

func SafeUint16(value uint16) (uint16, error) {
	if value > maxUInt16 {
		return 0, fmt.Errorf("value %d is outside uint16 range", value)
	}

	return value, nil
}

// SafeInt16 converts an int to int16 safely, returning an error if the value would overflow
func SafeInt16(value int) (int16, error) {
	if value > maxInt16 || value < minInt16 {
		return 0, fmt.Errorf("value %d is outside int16 range", value)
	}
	return int16(value), nil
}

// SafeInt32 converts an int to int32 safely, clamping to int32 bounds
func SafeInt32(value int) int32 {
	if value > maxInt32 {
		return maxInt32
	}
	if value < minInt32 {
		return minInt32
	}
	return int32(value)
}

func SafeUint64Ptr(i uint64, returnNilOnZero bool) *uint64 {
	if i == 0 && returnNilOnZero {
		return nil
	}

	return &i
}

// SafeInt64Ptr converts an int to a *int64.
// If the input is 0 and returnNilOnZero is true, it returns nil.
// Otherwise, it returns a pointer to the converted int64 value.
func SafeInt64Ptr(i int, returnNilOnZero bool) *int64 {
	if i == 0 && returnNilOnZero {
		return nil
	}

	i64 := int64(i)
	return &i64
}

// SafeInt64PtrOrNil converts an int to a *int64.
// Returns nil when the input is 0, otherwise returns a pointer to the converted int64 value.
// This maintains backward compatibility with the original function.
func SafeInt64PtrOrNil(i int) *int64 {
	return SafeInt64Ptr(i, true)
}

// SafeInt64PtrNonNil converts an int to a *int64.
// Always returns a pointer to the converted int64 value, even when the input is 0.
func SafeInt64PtrNonNil(i int) *int64 {
	return SafeInt64Ptr(i, false)
}

func ToInt16(i *int16) int16 {
	if i == nil {
		return 0
	}
	return *i
}
