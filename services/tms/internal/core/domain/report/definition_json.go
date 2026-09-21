package report

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
)

// DecodeDefinition reads a definition out of a decoded JSON value, such as a
// tool argument, and fills the version a caller left out. What it hands back
// is a shape, not a valid report: the compiler decides that, in its own words.
func DecodeDefinition(value any) (*Definition, error) {
	if value == nil {
		return nil, errors.New("a report definition is required")
	}

	encoded, err := sonic.Marshal(value)
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

	return definition, nil
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
					"required":             []string{"id", "kind"},
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
