package common

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func OptionalPulid(params map[string]any, key string) (pulid.ID, bool, error) {
	value := OptionalString(params, key)
	if value == "" {
		return pulid.Nil, false, nil
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, false, err
	}

	return id, true, nil
}
