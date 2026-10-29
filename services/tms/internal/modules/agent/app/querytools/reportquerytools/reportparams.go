package reportquerytools

import (
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
)

// NormalizeParameters coerces a model's parameters toward what the report
// declares, without inventing any value it did not supply.
//
// Only the container shape is corrected. A value that is wrong — a status the
// report does not accept, a window that is not a number — is still rejected by
// the compiler, in the compiler's own words, because a tool that quietly
// reinterpreted those would run a report nobody asked for.
func NormalizeParameters(
	definition *ir.Definition,
	values map[string]any,
) map[string]any {
	if definition == nil || len(values) == 0 {
		return values
	}

	normalized := make(map[string]any, len(values))
	for key, value := range values {
		normalized[key] = value
	}

	for i := range definition.Parameters {
		parameter := &definition.Parameters[i]
		value, ok := normalized[parameter.Name]
		if !ok || value == nil {
			continue
		}

		if parameter.Multi {
			normalized[parameter.Name] = querytool.AsList(value)
			continue
		}

		// The mirror image: a single-valued parameter handed a one-item list.
		if list, listOk := value.([]any); listOk && len(list) == 1 {
			normalized[parameter.Name] = list[0]
		}
	}

	return normalized
}

// describeParameterShape says what one parameter will accept, in the words a
// model needs to produce it: the type, and whether it is one value or a list.
func describeParameterShape(parameter *ir.Parameter) string {
	kind := string(parameter.Type)
	if kind == "" {
		kind = "value"
	}

	if parameter.Multi {
		return "a JSON array of " + kind
	}

	return "a single " + kind
}
