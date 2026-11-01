package gqlid

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func ApplyRequiredID(field, label string, value graphql.Omittable[*string], dest *pulid.ID) error {
	raw, ok := value.ValueOK()
	if !ok {
		return nil
	}
	if raw == nil {
		return errortypes.NewValidationError(
			field, errortypes.ErrRequired, "{0} cannot be cleared", label,
		)
	}

	id, err := RequiredID(field, *raw)
	if err != nil {
		return err
	}
	*dest = id

	return nil
}

func ApplyOptionalID(field string, value graphql.Omittable[*string], dest *pulid.ID) error {
	raw, ok := value.ValueOK()
	if !ok {
		return nil
	}

	id, err := OptionalScopedID(field, raw)
	if err != nil {
		return err
	}
	*dest = id

	return nil
}
