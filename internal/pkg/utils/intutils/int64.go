package intutils

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
