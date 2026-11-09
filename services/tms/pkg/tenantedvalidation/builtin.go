package tenantedvalidation

import (
	"reflect"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func (v *Validator[T]) validateBasics(
	entity T,
	valCtx *validationframework.Context,
	multiErr *errortypes.MultiError,
) {
	entity.Validate(multiErr)

	if valCtx.IsCreate() && entity.GetID().IsNotNil() {
		multiErr.Add("id", errortypes.ErrInvalid, "ID must not be set on create")
	}

	if original, ok := validationframework.OriginalOf[T](valCtx); ok && !isBlank(original) {
		v.validateImmutable(entity, original, multiErr)
	}

	v.validateDates(entity, multiErr)
	v.validateRanges(entity, multiErr)
}

func (v *Validator[T]) validateImmutable(entity, original T, multiErr *errortypes.MultiError) {
	for _, field := range v.immutableFields {
		if sameValue(field.GetValue(entity), field.GetValue(original)) {
			continue
		}
		if field.Message != "" {
			multiErr.Add(field.FieldName, errortypes.ErrInvalidOperation, field.Message)
			continue
		}
		multiErr.Add(
			field.FieldName,
			errortypes.ErrInvalidOperation,
			"This field cannot be changed after creation",
		)
	}
}

func (v *Validator[T]) validateDates(entity T, multiErr *errortypes.MultiError) {
	for _, config := range v.dateComparisons {
		start := config.GetStartDate(entity)
		end := config.GetEndDate(entity)
		if start == nil || end == nil {
			continue
		}
		if *end > *start || (config.AllowEqual && *end == *start) {
			continue
		}

		switch {
		case config.Message != "":
			multiErr.Add(config.FieldName, errortypes.ErrInvalid, config.Message)
		case config.AllowEqual:
			multiErr.Add(
				config.FieldName,
				errortypes.ErrInvalid,
				"Must be on or after the start date",
			)
		default:
			multiErr.Add(config.FieldName, errortypes.ErrInvalid, "Must be after the start date")
		}
	}
}

func (v *Validator[T]) validateRanges(entity T, multiErr *errortypes.MultiError) {
	for _, config := range v.numericRanges {
		value := config.GetValue(entity)
		if value == nil {
			continue
		}

		switch {
		case config.Min != nil && *value < *config.Min:
			if config.Message != "" {
				multiErr.Add(config.FieldName, errortypes.ErrInvalid, config.Message)
				continue
			}
			multiErr.Add(
				config.FieldName,
				errortypes.ErrInvalid,
				"Must be at least {0}",
				*config.Min,
			)
		case config.Max != nil && *value > *config.Max:
			if config.Message != "" {
				multiErr.Add(config.FieldName, errortypes.ErrInvalid, config.Message)
				continue
			}
			multiErr.Add(
				config.FieldName,
				errortypes.ErrInvalid,
				"Must be at most {0}",
				*config.Max,
			)
		}
	}
}

func sameValue(a, b any) bool {
	return reflect.DeepEqual(dereference(a), dereference(b))
}

func dereference(value any) any {
	current, ok := indirect(value)
	if !ok {
		return nil
	}

	return current.Interface()
}

func isBlank(value any) bool {
	current, ok := indirect(value)

	return !ok || (current.Kind() == reflect.String && current.Len() == 0)
}

func indirect(value any) (reflect.Value, bool) {
	current := reflect.ValueOf(value)
	for current.IsValid() && isReference(current.Kind()) {
		if current.IsNil() {
			return reflect.Value{}, false
		}
		current = current.Elem()
	}

	return current, current.IsValid()
}

func isReference(kind reflect.Kind) bool {
	return kind == reflect.Pointer || kind == reflect.Interface
}
