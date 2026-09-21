package agentquerytoolservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
)

// listWrapperKeys are the shapes a model reaches for when it has to produce a
// list and nothing told it the shape.
//
// The tool's parameters are a free-form object — they have to be, because every
// report declares its own — so a model filling one in is guessing. Several guess
// {"item": [...]}, which is what an XML-shaped function call looks like once it
// has been through a JSON conversion, and one of them did it five times in a row
// against a report it could otherwise run, narrating a different fix each time
// and sending the identical payload.
//
// The compiler is right to reject it: it serves the reports UI too, where the
// client sends exactly what the definition declares, and loosening it there
// would let a malformed saved report through. This is the agent's boundary, so
// this is where a guess gets straightened out.
var listWrapperKeys = []string{"item", "items", "value", "values", "list"}

// normalizeReportParameters coerces a model's parameters toward what the report
// declares, without inventing any value it did not supply.
//
// Only the container shape is corrected. A value that is wrong — a status the
// report does not accept, a window that is not a number — is still rejected by
// the compiler, in the compiler's own words, because a tool that quietly
// reinterpreted those would run a report nobody asked for.
func normalizeReportParameters(
	definition *report.Definition,
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
			normalized[parameter.Name] = asList(value)
			continue
		}

		// The mirror image: a single-valued parameter handed a one-item list.
		if list, listOk := value.([]any); listOk && len(list) == 1 {
			normalized[parameter.Name] = list[0]
		}
	}

	return normalized
}

// asList turns what a model sent into the list a multi-valued parameter needs.
func asList(value any) any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		list := make([]any, 0, len(typed))
		for _, entry := range typed {
			list = append(list, entry)
		}

		return list
	case map[string]any:
		if unwrapped, ok := unwrapList(typed); ok {
			return unwrapped
		}

		return value
	case string:
		return splitList(typed)
	default:
		// A lone scalar is a list of one. A model that names a single status
		// means that status, not a type error.
		return []any{value}
	}
}

// unwrapList takes the list back out of a single-key wrapper.
//
// Only a wrapper carrying exactly one key is unwrapped: an object with several
// keys is a value the model meant to be an object, and reaching into it would
// be guessing about the report rather than about the shape.
func unwrapList(wrapper map[string]any) (any, bool) {
	if len(wrapper) != 1 {
		return nil, false
	}

	for _, key := range listWrapperKeys {
		inner, ok := wrapper[key]
		if !ok {
			continue
		}
		if list, listOk := inner.([]any); listOk {
			return list, true
		}

		return []any{inner}, true
	}

	return nil, false
}

// splitList reads a comma-separated string as the list it is standing in for.
//
// A single value with no comma stays a single-item list rather than being cut
// up, so a status that legitimately contains no separator is untouched. A value
// that should not have been split is caught by the allowed-value check, which
// names what it accepts.
func splitList(value string) []any {
	parts := strings.Split(value, ",")
	list := make([]any, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}

	if len(list) == 0 {
		return []any{value}
	}

	return list
}

// describeParameterShape says what one parameter will accept, in the words a
// model needs to produce it: the type, and whether it is one value or a list.
func describeParameterShape(parameter report.ParameterDef) string {
	kind := string(parameter.Type)
	if kind == "" {
		kind = "value"
	}

	if parameter.Multi {
		return "a JSON array of " + kind
	}

	return "a single " + kind
}
