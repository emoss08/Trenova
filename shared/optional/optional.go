package optional

type Value[T any] struct {
	Set   bool
	Value T
}

func Some[T any](v T) Value[T] {
	return Value[T]{Set: true, Value: v}
}

func (v Value[T]) Apply(field *T) bool {
	if !v.Set {
		return false
	}
	*field = v.Value

	return true
}
