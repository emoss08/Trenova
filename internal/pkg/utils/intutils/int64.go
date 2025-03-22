package intutils

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
