package typeutils

// EqualPtr reports whether two pointers hold the same value, or are both nil.
func EqualPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func Deref[T any](value *T) (T, bool) {
	if value == nil {
		var zero T
		return zero, false
	}

	return *value, true
}
