package tableconfigurationresolver

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

func requiredPatchValue[T any](field, label string, value *T) (T, error) {
	if value == nil {
		var zero T
		return zero, errortypes.NewValidationError(
			field,
			errortypes.ErrRequired,
			"{0} cannot be cleared", label,
		)
	}
	return *value, nil
}
