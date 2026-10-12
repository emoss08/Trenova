package common

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func OptionalID(params map[string]any, key string) (pulid.ID, error) {
	id, err := OptionalPulidParam(params, key)
	if err != nil || id == nil {
		return pulid.Nil, err
	}

	return *id, nil
}

func OptionalIntInRange(
	params map[string]any,
	key string,
	minimum, maximum int,
) (*int, error) {
	if raw, given := params[key]; !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent number is no change and no error
	}
	value, err := RequireIntInRange(params, key, minimum, maximum)
	if err != nil {
		return nil, err
	}

	return &value, nil
}
