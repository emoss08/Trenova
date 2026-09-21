package toolschema

import "github.com/bytedance/sonic"

// Changed keeps only the modifications that differ from what was proposed.
// A form sends every field back; what the person actually changed is what
// the decision records and what the model is told about, and a value equal
// to the proposed one is not a change. Values compare as JSON, so 3 and 3.0
// agree and key order does not matter.
func Changed(proposed, modifications map[string]any) map[string]any {
	changed := make(map[string]any, len(modifications))
	for key, value := range modifications {
		current, had := proposed[key]
		if !had && value == nil {
			continue
		}
		if had && equivalent(current, value) {
			continue
		}
		changed[key] = value
	}

	return changed
}

func equivalent(a, b any) bool {
	left, err := sonic.ConfigStd.Marshal(a)
	if err != nil {
		return false
	}
	right, err := sonic.ConfigStd.Marshal(b)
	if err != nil {
		return false
	}

	return string(left) == string(right)
}
