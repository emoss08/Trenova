package typeutils

// EqualPtr reports whether two pointers hold the same value, or are both nil.
func EqualPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}
