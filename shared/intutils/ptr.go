package intutils

func NilIfZero[T Numeric](value T) *T {
	if value == 0 {
		return nil
	}

	return &value
}

func MaxPointer[T Numeric](a, b *T) *T {
	switch {
	case a == nil:
		return ClonePointer(b)
	case b == nil || *a >= *b:
		return ClonePointer(a)
	default:
		return ClonePointer(b)
	}
}
