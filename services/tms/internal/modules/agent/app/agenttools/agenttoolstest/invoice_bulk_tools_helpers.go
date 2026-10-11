package agenttoolstest

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func IdList(ids []pulid.ID) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}
