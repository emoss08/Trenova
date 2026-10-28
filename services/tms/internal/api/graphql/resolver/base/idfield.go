package base

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func ParseIDField(field, raw string) (pulid.ID, error) {
	id, err := pulid.MustParse(raw)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Invalid identifier",
		)
	}

	return id, nil
}

func ParseOptionalIDField(field string, raw *string) (pulid.ID, error) {
	if raw == nil || *raw == "" {
		return pulid.Nil, nil
	}

	return ParseIDField(field, *raw)
}

func ParseIDListField(field string, raw []string) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		id, err := ParseIDField(field, value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}
