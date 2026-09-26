package toolschema

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

// RecordSubset marks an array-of-ids parameter as a set of records of one
// resource that an approver may narrow but never widen. The resource is the
// permission resource the ids belong to, which is what lets a client show the
// records as rows a person can untick.
func RecordSubset(resource string, property map[string]any) map[string]any {
	property[KeySubsetOf] = resource

	return property
}

// MaxSubsetChoices is the most records one subset parameter lists for a
// person to untick, whatever its schema declares.
const MaxSubsetChoices = 5000

// SubsetChoices lists, per record-subset parameter, the records a call
// proposed: in the order proposed, each once, at most the parameter's
// maxItems. Each is named by its id until its label is read.
func SubsetChoices(schema, params map[string]any) map[string][]Choice {
	properties, _ := schema[KeyProperties].(map[string]any)
	choices := make(map[string][]Choice)

	for name, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok || SubsetResource(property) == "" {
			continue
		}
		choices[name] = choicesOf(idsOf(params[name]), subsetCap(property))
	}

	return choices
}

func subsetCap(property map[string]any) int {
	if declared := intOf(property[KeyMaxItems]); declared > 0 && declared < MaxSubsetChoices {
		return declared
	}

	return MaxSubsetChoices
}

func choicesOf(ids []string, limit int) []Choice {
	out := make([]Choice, 0, min(len(ids), limit))
	seen := make(map[string]struct{}, min(len(ids), limit))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, Choice{ID: id, Label: id})
		if len(out) == limit {
			break
		}
	}

	return out
}

// SubsetResource reads the resource a property declares its ids a subset of.
func SubsetResource(property map[string]any) string {
	resource, _ := property[KeySubsetOf].(string)

	return strings.TrimSpace(resource)
}

// CheckSubsets refuses a change to a record-subset parameter that is not a
// narrowing of what was proposed. The agent proposed a write over a set of
// records and a person approves a part of it: removing records keeps the write
// inside what was proposed, adding one would approve a change nobody proposed,
// and removing every one leaves nothing to approve.
func CheckSubsets(schema, proposed, changed map[string]any) error {
	properties, _ := schema[KeyProperties].(map[string]any)
	multiErr := errortypes.NewMultiError()

	for name, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok || SubsetResource(property) == "" {
			continue
		}
		value, modified := changed[name]
		if !modified {
			continue
		}
		checkSubset(multiErr, name, idsOf(proposed[name]), value)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func checkSubset(multiErr *errortypes.MultiError, name string, proposed []string, value any) {
	items, ok := value.([]any)
	if !ok {
		if typed, isStrings := value.([]string); isStrings {
			items = make([]any, 0, len(typed))
			for _, id := range typed {
				items = append(items, id)
			}
			ok = true
		}
	}
	if !ok {
		multiErr.Add(name, errortypes.ErrInvalid, "Must be a list of the proposed records")

		return
	}
	if len(items) == 0 {
		multiErr.Add(
			name,
			errortypes.ErrRequired,
			"Keep at least one of the proposed records, or reject the proposal",
		)

		return
	}

	allowed := make(map[string]struct{}, len(proposed))
	for _, id := range proposed {
		allowed[id] = struct{}{}
	}
	for _, item := range items {
		id, isText := item.(string)
		if _, kept := allowed[strings.TrimSpace(id)]; isText && kept {
			continue
		}
		multiErr.Add(
			name,
			errortypes.ErrForbidden,
			"{0} was not in the proposal; an approver may remove records, never add them",
			fmt.Sprint(item),
		)
	}
}

func idsOf(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		ids := make([]string, 0, len(typed))
		for _, item := range typed {
			if id, ok := item.(string); ok {
				ids = append(ids, strings.TrimSpace(id))
			}
		}

		return ids
	default:
		return nil
	}
}

// ForModel is the schema as a model is shown it: a copy without the
// extension keywords ("x-…") this package reads for people. A provider that
// validates tool schemas strictly refuses a keyword it does not know, and
// none of them means anything to a model.
func ForModel(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	return stripSchema(schema)
}

// nameMaps are the keywords whose value maps names to schemas: the names are
// the tool's, never keywords, and are kept whatever they look like.
var nameMaps = map[string]struct{}{
	KeyProperties:       {},
	"patternProperties": {},
	"$defs":             {},
	"definitions":       {},
	"dependentSchemas":  {},
}

func stripSchema(schema map[string]any) map[string]any {
	out := make(map[string]any, len(schema))
	for key, value := range schema {
		if strings.HasPrefix(key, extensionPrefix) {
			continue
		}
		if _, named := nameMaps[key]; named {
			if children, ok := value.(map[string]any); ok {
				out[key] = stripNamed(children)

				continue
			}
		}
		out[key] = stripValue(value)
	}

	return out
}

func stripNamed(children map[string]any) map[string]any {
	out := make(map[string]any, len(children))
	for name, child := range children {
		out[name] = stripValue(child)
	}

	return out
}

func stripValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return stripSchema(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, stripValue(item))
		}

		return out
	default:
		return value
	}
}
