package base

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/optional"
)

func RequiredOmittable[T any](
	field, label string,
	value graphql.Omittable[*T],
) (optional.Value[T], error) {
	if !value.IsSet() {
		return optional.Value[T]{}, nil
	}
	ptr := value.Value()
	if ptr == nil {
		return optional.Value[T]{}, errortypes.NewValidationError(
			field, errortypes.ErrRequired, "{0} cannot be cleared", label,
		)
	}

	return optional.Some(*ptr), nil
}

func NullableOmittable[T any](value graphql.Omittable[*T]) optional.Value[*T] {
	if !value.IsSet() {
		return optional.Value[*T]{}
	}

	return optional.Some(value.Value())
}
