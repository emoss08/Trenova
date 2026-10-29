package querytool

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

func MinorText(minor int64) string {
	return money.DecimalFromMinor(minor).StringFixed(2)
}

func PulidString(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

func OptionalPulidSlice(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil
	}

	var values []string
	if err := DecodeParam(params, key, &values); err != nil {
		return nil, err
	}
	if len(values) > limit {
		return nil, fmt.Errorf("parameter %q lists more than %d ids", key, limit)
	}

	ids := make([]pulid.ID, 0, len(values))
	for idx, value := range values {
		id, err := pulid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%d] is not an id", key, idx)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func FieldFilterMaps(filters []domaintypes.FieldFilter) []map[string]any {
	out := make([]map[string]any, 0, len(filters))
	for _, filter := range filters {
		entry := map[string]any{
			ParamField:     filter.Field,
			FilterOperator: string(filter.Operator),
		}
		if filter.Value != nil {
			entry[FilterValue] = filter.Value
		}
		out = append(out, entry)
	}

	return out
}

func OptionalPulid(params map[string]any, key string) (pulid.ID, bool, error) {
	value := OptionalString(params, key)
	if value == "" {
		return pulid.Nil, false, nil
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, false, fmt.Errorf("parameter %q is not an id", key)
	}

	return id, true, nil
}

func InputTexts(inputs tarifftypes.Inputs) map[string]string {
	if len(inputs) == 0 {
		return nil
	}
	out := make(map[string]string, len(inputs))
	for name, raw := range inputs {
		out[name] = tarifftypes.DisplayValue(raw)
	}
	return out
}

// ViewRun is what a composed view holds when it is run: up to its counting
// limit of rows in the list's own shape, and whether there were more.
type ViewRun struct {
	Outcome  SearchOutcome
	Withheld []string
	More     bool
}

func PointerSeconds(seconds *int64) int64 {
	if seconds == nil {
		return 0
	}

	return *seconds
}

func StringOf(value any) string {
	text, _ := value.(string)

	return text
}

func RequireIDList(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	ids, err := OptionalPulidSlice(params, key, limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("parameter %q needs at least one id", key)
	}

	return ids, nil
}

// AsList turns what a model sent into the list a multi-valued parameter needs.
func AsList(value any) any {
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

func TenantOf(params *agenttoolports.QueryToolParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

// PulidList reads an array of ids, refusing a malformed one by position.
func PulidList(raw any, key string) ([]pulid.ID, error) {
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an array of ids", key)
	}

	ids := make([]pulid.ID, 0, len(values))
	for index, value := range values {
		text, isText := value.(string)
		if !isText || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("parameter %q[%d] must be a non-empty id", key, index)
		}
		id, err := pulid.Parse(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("parameter %q[%d] is not a valid id: %w", key, index, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// WorkerName is how a person names a driver. Every tool that returns a worker
// row needs it, and a driver whose record carries only one of the two names
// should still read as a name rather than a stray space.
func WorkerName(w *worker.Worker) string {
	if w == nil {
		return ""
	}

	return strings.TrimSpace(w.FirstName + " " + w.LastName)
}
