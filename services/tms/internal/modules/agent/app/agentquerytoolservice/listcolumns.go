package agentquerytoolservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// paramFields names the columns of each row a model reads of a list. A worker
// row with its profile is about 900 characters and a shipment row with its
// route about 600, so a page a model had room to read held thirteen workers:
// asked for every inactive driver, it read a quarter of them and guessed at
// the rest. Names and statuses alone fit ten times as many. The list returns
// every column whatever the call names, so the person's table keeps them all;
// the runtime narrows the model's copy (agentruntime.narrowRows).
const paramFields = agentports.ListFieldsParam

// fieldsSchema is the fields parameter of a list whose columns are source.
func fieldsSchema(source agenttoolschema.EnumSource[string]) map[string]any {
	return map[string]any{
		toolschema.KeyType:  toolschema.TypeArray,
		toolschema.KeyItems: agenttoolschema.Enum("", source),
		toolschema.KeyDescription: "Only these columns of each row, besides its id, in what you " +
			"read; the person's table keeps every column. Name just what the answer needs: a " +
			"narrow row fits many times as many rows on a page, so a question across many " +
			"records reads in one or two pages. Leave it out for every column.",
	}
}

// listColumns are the columns a model may name in fields: a read's selected
// top-level fields but the id, each under the name a row shows it by.
func listColumns(fields []graphQLField, amounts bool) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if !field.Selected || field.Name == recordIDField {
			continue
		}
		name := field.Name
		if amounts {
			name = presentedName(name)
		}
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}

	return out
}
