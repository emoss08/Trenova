package report

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

// DecodeDefinition reads a definition out of a decoded JSON value, such as a
// tool argument, and fills the version a caller left out. What it hands back
// is a shape, not a valid report: the compiler decides that, in its own words.
func DecodeDefinition(value any) (*Definition, error) {
	if value == nil {
		return nil, errors.New("a report definition is required")
	}

	encoded, err := sonic.Marshal(normalizeDefinitionShape(value))
	if err != nil {
		return nil, fmt.Errorf("encode report definition: %w", err)
	}

	definition := new(Definition)
	if err = sonic.Unmarshal(encoded, definition); err != nil {
		return nil, fmt.Errorf("the report definition is not the shape expected: %w", err)
	}
	if definition.IRVersion == 0 {
		definition.IRVersion = CurrentIRVersion
	}
	if definition.Entity == "" {
		return nil, errors.New("the report definition names no dataset (entity)")
	}
	if len(definition.Columns) == 0 {
		return nil, errors.New("the report definition has no columns")
	}
	for idx, column := range definition.Columns {
		if column.Kind == ColumnKindComputed || column.Ref.Field != "" {
			continue
		}

		return nil, fmt.Errorf(
			"column %q (columns[%d]) names no field: give it {\"ref\": {\"field\": \"<key>\"}}, "+
				"or {\"ref\": {\"path\": [\"<edge>\"], \"field\": \"<key>\"}} for a field on a "+
				"related dataset, using the keys describe_report_dataset lists",
			column.ID, idx,
		)
	}

	return definition, nil
}

// normalizeDefinitionShape accepts the ways a field reference is written
// besides the canonical one, so a definition copied from describe_report's
// column summary — where a field reads "assignment.primaryWorker.firstName" —
// decodes to the same thing the builder saves.
//
// A column with a dotted "field" and no "ref" gains a ref split on the dots;
// a ref whose field is dotted and whose path is empty is split the same
// way. Anything else is left exactly as written.
func normalizeDefinitionShape(value any) any {
	root, ok := value.(map[string]any)
	if !ok {
		return value
	}

	for _, key := range [...]string{"filters", "having"} {
		group, present := root[key]
		if !present {
			continue
		}
		if normalized := normalizeFilterGroup(group); normalized != nil {
			root[key] = normalized
		} else {
			delete(root, key)
		}
	}

	columns, ok := root["columns"].([]any)
	if !ok {
		return normalizeRefs(root)
	}
	for _, raw := range columns {
		column, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		if _, hasRef := column["ref"]; !hasRef {
			if field, isString := column["field"].(string); isString && field != "" {
				column["ref"] = map[string]any{"field": field}
				delete(column, "field")
			}
		}
	}

	return normalizeRefs(root)
}

// normalizeFilterGroup reads a filter group in the forms a caller writes one.
//
// The builder saves {"op": "and", "filters": [...], "groups": [...]}. A model
// asked for "workers whose driver type is OTR" writes the conditions as a
// bare list, which has one meaning — all of them must hold — and was refused
// as the wrong shape at the proposal, with an error about a type mismatch at
// a byte offset that the model could not act on. A list is read as an "and"
// group, a group with no op as "and", and null as no filter at all. Nested
// groups are read the same way. Anything else is left for the decoder to
// report.
func normalizeFilterGroup(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []any:
		if len(typed) == 0 {
			return nil
		}
		group := map[string]any{"op": "and"}
		filters := make([]any, 0, len(typed))
		groups := make([]any, 0)
		for _, item := range typed {
			if isFilterGroupShape(item) {
				if nested := normalizeFilterGroup(item); nested != nil {
					groups = append(groups, nested)
				}
				continue
			}
			filters = append(filters, item)
		}
		if len(filters) > 0 {
			group["filters"] = filters
		}
		if len(groups) > 0 {
			group["groups"] = groups
		}

		return group
	case map[string]any:
		if op, _ := typed["op"].(string); strings.TrimSpace(op) == "" {
			typed["op"] = "and"
		} else {
			typed["op"] = strings.ToLower(strings.TrimSpace(op))
		}
		if filters, present := typed["filters"]; present && filters == nil {
			delete(typed, "filters")
		}
		if nested, present := typed["groups"]; present {
			switch groups := nested.(type) {
			case nil:
				delete(typed, "groups")
			case []any:
				kept := make([]any, 0, len(groups))
				for _, group := range groups {
					if normalized := normalizeFilterGroup(group); normalized != nil {
						kept = append(kept, normalized)
					}
				}
				typed["groups"] = kept
			}
		}

		return typed
	default:
		return value
	}
}

// isFilterGroupShape tells a nested group from a condition inside a list:
// a condition names a field to compare, a group holds conditions.
func isFilterGroupShape(value any) bool {
	item, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if _, hasRef := item["ref"]; hasRef {
		return false
	}
	_, hasFilters := item["filters"]
	_, hasGroups := item["groups"]
	_, hasOp := item["op"]

	return hasFilters || hasGroups || hasOp
}

// normalizeRefs walks the definition and splits every dotted ref field
// with no path into its edges and field.
func normalizeRefs(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "ref" {
				if ref, isMap := child.(map[string]any); isMap {
					splitDottedRef(ref)
				}

				continue
			}
			typed[key] = normalizeRefs(child)
		}

		return typed
	case []any:
		for idx, child := range typed {
			typed[idx] = normalizeRefs(child)
		}

		return typed
	default:
		return value
	}
}

func splitDottedRef(ref map[string]any) {
	field, ok := ref["field"].(string)
	if !ok || !strings.Contains(field, ".") {
		return
	}
	if existing, hasPath := ref["path"].([]any); hasPath && len(existing) > 0 {
		return
	}

	parts := strings.Split(field, ".")
	path := make([]any, 0, len(parts)-1)
	for _, part := range parts[:len(parts)-1] {
		path = append(path, part)
	}
	ref["path"] = path
	ref["field"] = parts[len(parts)-1]
}

// DefinitionJSONSchema describes a definition to a caller that writes one
// by hand, such as a model. It names the shape the builder saves, in the
// builder's own words, so a report written here and one written in the
// builder are the same thing.
func DefinitionJSONSchema() map[string]any {
	ref := map[string]any{
		"type": "object",
		"description": "A field on the dataset, or on a related dataset reached along " +
			"edges: {\"field\": \"name\"} for the dataset's own field, " +
			"{\"path\": [\"customer\"], \"field\": \"name\"} for a field one edge away. " +
			"Edge names and field keys come from describe_report_dataset.",
		"properties": map[string]any{
			"path":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"field": map[string]any{"type": "string"},
		},
		"required":             []string{"field"},
		"additionalProperties": false,
	}
	filter := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ref": ref,
			"operator": map[string]any{
				"type": "string",
				"enum": []string{
					"eq", "ne", "gt", "gte", "lt", "lte", "contains", "startswith",
					"endswith", "in", "notin", "isnull", "isnotnull", "daterange",
					"lastndays", "nextndays", "today", "yesterday", "tomorrow", "thisweek",
					"lastweek", "thismonth", "lastmonth", "thisquarter", "lastquarter",
					"thisyear", "lastyear",
				},
			},
			"value": map[string]any{
				"description": "The value to compare with: a scalar, a list for in and " +
					"notin, a number of days for lastndays and nextndays, " +
					"[from, to] epoch seconds for daterange. Omit when param is set.",
			},
			"param": map[string]any{
				"type":        "string",
				"description": "Bind the value to a parameter by name instead of fixing it.",
			},
			"agg": map[string]any{
				"type":        "string",
				"description": "Only inside having: the aggregation to filter on.",
			},
		},
		"required":             []string{"ref", "operator"},
		"additionalProperties": false,
	}
	filterGroup := map[string]any{
		"type": "object",
		"description": "An object, not a list: {\"op\": \"and\", \"filters\": [conditions]}. " +
			"For workers whose driver type is OTR: {\"op\": \"and\", \"filters\": " +
			"[{\"ref\": {\"field\": \"driverType\"}, \"operator\": \"eq\", \"value\": \"OTR\"}]}. " +
			"Leave it out for no filter.",
		"properties": map[string]any{
			"op":      map[string]any{"type": "string", "enum": []string{"and", "or"}},
			"filters": map[string]any{"type": "array", "items": filter},
			"groups": map[string]any{
				"type":        "array",
				"description": "Nested groups, each the same shape as this one.",
				"items":       map[string]any{"type": "object"},
			},
		},
		"required":             []string{"op"},
		"additionalProperties": false,
	}

	return map[string]any{
		"type": "object",
		"description": "A report as the report builder saves it. Grouping is implied: a " +
			"dimension column groups, a measure column aggregates. A report with " +
			"only dimensions lists rows.",
		"properties": map[string]any{
			"entity": map[string]any{
				"type":        "string",
				"description": "The dataset key from list_report_datasets, such as shipment.",
			},
			"columns": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Your own short unique id for the column, such as c1.",
						},
						"ref": ref,
						"kind": map[string]any{
							"type": "string",
							"enum": []string{"dimension", "measure", "computed"},
						},
						"agg": map[string]any{
							"type":        "string",
							"enum":        []string{"count", "count_distinct", "sum", "avg", "min", "max"},
							"description": "Required for a measure; the field must support it.",
						},
						"bucket": map[string]any{
							"type":        "string",
							"enum":        []string{"day", "week", "month", "quarter", "year"},
							"description": "Groups a date dimension by period.",
						},
						"label": map[string]any{"type": "string"},
						"computed": map[string]any{
							"type": "object",
							"description": "For kind computed: {op: add|subtract|multiply|divide, " +
								"leftId, rightId} over two other column ids, or leftValue/" +
								"rightValue for a constant.",
						},
					},
					"required":             []string{"id", "kind", "ref"},
					"additionalProperties": true,
				},
			},
			"filters": filterGroup,
			"having": map[string]any{
				"type":        "object",
				"description": "Filters over aggregated measures, the same shape as filters.",
			},
			"sort": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"columnId":  map[string]any{"type": "string"},
						"direction": map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
					},
					"required":             []string{"columnId", "direction"},
					"additionalProperties": false,
				},
			},
			"limit": map[string]any{"type": "integer"},
			"parameters": map[string]any{
				"type":        "array",
				"description": "Values asked for at run time and bound to filters by param.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":  map[string]any{"type": "string"},
						"label": map[string]any{"type": "string"},
						"type": map[string]any{
							"type": "string",
							"enum": []string{"string", "int", "decimal", "bool", "enum", "epoch", "ref"},
						},
						"required":      map[string]any{"type": "boolean"},
						"default":       map[string]any{},
						"multi":         map[string]any{"type": "boolean"},
						"allowedValues": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"refEntity":     map[string]any{"type": "string"},
					},
					"required":             []string{"name", "type"},
					"additionalProperties": false,
				},
			},
			"totals": map[string]any{"type": "boolean"},
			"charts": map[string]any{
				"type": "array",
				"description": "Optional: {id, type: bar|hbar|line|area|pie|donut|scatter|kpi, " +
					"title, xColumnId, seriesIds}.",
				"items": map[string]any{"type": "object"},
			},
		},
		"required":             []string{"entity", "columns"},
		"additionalProperties": true,
	}
}
