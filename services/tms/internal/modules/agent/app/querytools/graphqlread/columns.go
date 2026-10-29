package graphqlread

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
)

// ListColumns are the columns a model may name in fields: a read's selected
// top-level fields but the id, each under the name a row shows it by.
func ListColumns(fields []Field, amounts bool) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if !field.Selected || field.Name == querytool.RecordIDField {
			continue
		}
		name := field.Name
		if amounts {
			name = PresentedName(name)
		}
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}

	return out
}
