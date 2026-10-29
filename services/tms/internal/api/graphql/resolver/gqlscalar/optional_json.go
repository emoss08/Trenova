package gqlscalar

import (
	"github.com/emoss08/trenova/shared/jsonutils"
)

func OptionalJSON(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}
	return jsonutils.ToJSON(value)
}
