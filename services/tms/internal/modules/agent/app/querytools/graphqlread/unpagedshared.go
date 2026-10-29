package graphqlread

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/textcase"
)

const (
	// AlongsideMost is how many records a read made alongside a list returns:
	// the server's most for one page, which the catalogs the repository tools
	// read beside a worker's records (checklist and review templates) never
	// approached.
	AlongsideMost       = 100
	OutcomeCount        = "count"
	OutcomeSearchedFor  = "searchedFor"
	OutcomeItems        = "items"
	OutcomeColumns      = "columns"
	OutcomeNote         = "note"
	OutcomeWithheld     = "withheldByAccess"
	OutcomeWithheldNote = "withheldNote"
)

// ArgumentParam is one argument's parameter. An id is a reference the spec
// types by its resource; anything else is derived as a table input's own
// field is.
func ArgumentParam(field *Field) []Param {
	if field.Type == TypeID && field.List {
		words := textcase.HumanizeSnakeCase(textcase.ConvertCamelToSnake(
			strings.TrimSuffix(field.Name, "Ids")))
		return []Param{{
			Name: field.Name, Field: field.Name, Kind: ParamReferences,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The " + words + " records whose records are read.",
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
				toolschema.KeyMinItems: 1,
			},
		}}
	}
	if field.Type == TypeID && !field.List {
		words := textcase.HumanizeSnakeCase(textcase.ConvertCamelToSnake(
			strings.TrimSuffix(field.Name, suffixID)))
		return []Param{{
			Name: field.Name, Field: field.Name, Kind: ParamReference,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The " + words + " whose records are read.",
			},
		}}
	}

	if field.Type == TypeID {
		description := field.Description
		if description == "" {
			description = "The " + textcase.HumanizeSnakeCase(textcase.ConvertCamelToSnake(
				field.Name)) + " whose records are read."
		}
		return []Param{{
			Name: field.Name, Field: field.Name, Kind: ParamReferences,
			Schema: agenttoolschema.IDList(description, querytool.UnpagedMost),
		}}
	}
	if field.Type == TypeInt && !field.List {
		return intArgumentParam(field)
	}
	if field.Type == TypeString && !field.List &&
		strings.Contains(field.Description, dayTextFormat) {
		return []Param{{
			Name: field.Name, Field: field.Name, Kind: ParamDayText,
			Schema: agenttoolschema.Date(strings.TrimSpace(
				strings.ReplaceAll(field.Description, ", as "+dayTextFormat, ""))),
		}}
	}

	return inputFieldParams(field)
}

// intArgumentParam is an Int argument's parameter: a limit takes the list
// tools' bounds, and any other number is the value the schema describes (a
// case year), sent as it is.
func intArgumentParam(field *Field) []Param {
	if field.Name == querytool.ParamLimit {
		return []Param{{
			Name: field.Name, Field: field.Name, Kind: ParamEquals,
			Schema: querytool.LimitSchema(querytool.GraphQLListLimit, querytool.UnpagedMost),
		}}
	}
	if field.Description == "" {
		return nil
	}

	return []Param{{
		Name: field.Name, Field: field.Name, Kind: ParamEquals,
		Schema: map[string]any{
			toolschema.KeyType:        toolschema.TypeInteger,
			toolschema.KeyDescription: field.Description,
		},
	}}
}

// alongsideVariables are what a read alongside sends: a keyed read the
// values the list's own call sent for its keys, and whether the call named
// them all (a read keyed by nothing is made on every call); a connection
// read its table input.
func alongsideVariables(
	bound *Alongside,
	spec *readspec.Alongside,
	sent map[string]any,
) (variables map[string]any, keyed bool) {
	if bound.Input == "" {
		variables = make(map[string]any, len(bound.Keys)+len(spec.Inputs))
		for name, value := range spec.Inputs {
			variables[name] = value
		}
		for _, key := range bound.Keys {
			value, present := sent[key]
			if !present || value == nil {
				return nil, false
			}
			variables[key] = value
		}

		return variables, true
	}
	input := make(map[string]any, len(spec.Inputs)+1)
	for name, value := range spec.Inputs {
		input[name] = value
	}
	input[InputFirst] = AlongsideMost

	return map[string]any{bound.Input: input}, true
}

// alongsideRows are the records a read alongside returned, the prefix its
// withheld paths carry, and the note a connection read cut short leaves.
func alongsideRows(
	bound *Alongside,
	spec *readspec.Alongside,
	data map[string]any,
) (rows []map[string]any, prefix, note string) {
	if bound.Input == "" {
		list, _ := data[bound.RootField].([]any)
		rows = make([]map[string]any, 0, len(list))
		for _, row := range list {
			if node, ok := row.(map[string]any); ok {
				rows = append(rows, Compact(node))
			}
		}

		return rows, bound.RootField + ".", ""
	}
	connection, _ := data[bound.RootField].(map[string]any)
	edges, _ := connection[ConnectionEdges].([]any)
	rows = make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		object, _ := edge.(map[string]any)
		if node, ok := object[ConnectionNode].(map[string]any); ok {
			rows = append(rows, Compact(node))
		}
	}
	if info, ok := connection[ConnectionPageInfo].(map[string]any); ok {
		if more, _ := info[PageHasNext].(bool); more {
			note = fmt.Sprintf("Only the first %d %s are shown in %s.",
				AlongsideMost, spec.What, spec.Key)
		}
	}

	return rows, bound.RootField + "." + ConnectionEdges + "." + ConnectionNode + ".", note
}

// RelativePaths names withheld fields from the record, as a model reads
// them: workerChecklists.items.note is items.note.
func RelativePaths(paths []string, prefix string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, strings.TrimPrefix(path, prefix))
	}

	return out
}
