package listtool

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// FieldsSchema is the fields parameter of a list whose columns are source.
func FieldsSchema(source agenttoolschema.EnumSource[string]) map[string]any {
	return map[string]any{
		toolschema.KeyType:  toolschema.TypeArray,
		toolschema.KeyItems: agenttoolschema.Enum("", source),
		toolschema.KeyDescription: "Only these columns of each row, besides its id, in what you " +
			"read; the person's table keeps every column. Name just what the answer needs: a " +
			"narrow row fits many times as many rows on a page, so a question across many " +
			"records reads in one or two pages. Leave it out for every column.",
	}
}
