package dbhelper

import "github.com/uptrace/bun/dialect/pgdialect"

// TextArray binds a Go slice to a Postgres TEXT[] column.
//
// Bun renders a bare slice passed to UpdateQuery.Set as JSON, because Set takes
// an untyped value and never sees the field's `array` tag — the tag only steers
// model-based inserts. Postgres then rejects the JSON against a text[] column:
//
//	malformed array literal: "[\"DocumentClassification\",\"AssistantChat\"]"
//
// which is what saving an AI provider or an agent definition did, and what the
// same mistake did earlier in a WHERE clause on the same column. A Go slice and
// a pgdialect.Array are indistinguishable at the call site, so the only defence
// is to route every array binding through one function and to test the rendered
// SQL rather than the compile.
//
// Empty returns nil rather than an empty array, so clearing a list writes NULL
// and matches the nullzero the columns are declared with.
func TextArray[T ~string](values []T) any {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return pgdialect.Array(out)
}
