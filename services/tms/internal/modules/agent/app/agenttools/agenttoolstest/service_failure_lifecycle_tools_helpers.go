package agenttoolstest

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func ManyIDs(n int) []any {
	ids := make([]any, 0, n)
	for range n {
		ids = append(ids, pulid.MustNew("shp_").String())
	}

	return ids
}
