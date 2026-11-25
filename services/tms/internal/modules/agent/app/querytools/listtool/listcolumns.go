package listtool

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// ColumnSource is what a list's fields parameter may name: its columns and
// the id, which every row keeps whatever fields names. A call that names the
// id is not refused for it: records_query takes it, and a script asking for
// fields=["id"] wants just the ids.
func ColumnSource(name string, columns []string) agenttoolschema.EnumSource[string] {
	names := make([]string, 0, len(columns)+1)
	names = append(names, querytool.RecordIDField)
	for _, column := range columns {
		if !slices.Contains(names, column) {
			names = append(names, column)
		}
	}

	return agenttoolschema.Derived(name, names)
}

// OffersFields reports whether a list has columns worth choosing among: more
// than one beside the id.
func OffersFields(source agenttoolschema.EnumSource[string]) bool {
	return len(source.Values) > 2
}

// FieldsSchema is the fields parameter of a list whose columns are source.
func FieldsSchema(source agenttoolschema.EnumSource[string]) map[string]any {
	return map[string]any{
		toolschema.KeyType:  toolschema.TypeArray,
		toolschema.KeyItems: agenttoolschema.Enum("", source),
		toolschema.KeyDescription: "Only these columns of each row, besides its id, in what you " +
			"read; the id is always kept, so naming it changes nothing. A filter such as " +
			"status or a has... flag is its own parameter, never a column here. The person's " +
			"table keeps every column. Name just what the answer needs: a narrow row fits " +
			"many times as many rows on a page, so a question across many records reads in " +
			"one or two pages. Leave it out for every column.",
	}
}
