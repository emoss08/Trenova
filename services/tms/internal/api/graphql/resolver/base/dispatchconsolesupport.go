package base

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func OptionalPulid(raw *string) (*pulid.ID, error) {
	if raw == nil || *raw == "" {
		return nil, nil //nolint:nilnil // an absent optional ID is not an error
	}
	id, err := pulid.Parse(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
