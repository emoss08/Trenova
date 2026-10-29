package toolparams

import (
	"fmt"
)

func ReadObjects(params map[string]any, key string, limit int) ([]map[string]any, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of objects", key)
	}
	if len(entries) > limit {
		return nil, fmt.Errorf("%s lists %d entries; at most %d", key, len(entries), limit)
	}

	objects := make([]map[string]any, 0, len(entries))
	for idx, entry := range entries {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", key, idx)
		}
		objects = append(objects, fields)
	}

	return objects, nil
}
