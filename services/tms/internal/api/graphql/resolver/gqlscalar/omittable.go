package gqlscalar

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

func ApplyRequired[T any](field, label string, value graphql.Omittable[*T], dest *T) error {
	resolved, err := RequiredOmittable(field, label, value)
	if err != nil {
		return err
	}
	resolved.Apply(dest)

	return nil
}

func ApplyRequiredInt[D NarrowInteger](
	field, label string,
	value graphql.Omittable[*int],
	dest *D,
) error {
	resolved, err := RequiredOmittable(field, label, value)
	if err != nil {
		return err
	}
	if !resolved.Set {
		return nil
	}
	narrowed, err := NarrowInt[D](field, resolved.Value)
	if err != nil {
		return err
	}
	*dest = narrowed

	return nil
}

func ApplyClearable[T any](value graphql.Omittable[*T], dest *T) {
	if !value.IsSet() {
		return
	}
	if ptr := value.Value(); ptr != nil {
		*dest = *ptr
		return
	}
	var zero T
	*dest = zero
}

func ApplyNullable[T any](value graphql.Omittable[*T], dest **T) {
	NullableOmittable(value).Apply(dest)
}
