package agentquerytoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/toolschema"
)

// page is one window of a list: how many rows, from where.
//
// A list used to take a limit and nothing else, and its answer carried no
// sign of whether more rows existed. A model handed a full page of 25 could
// not tell it from the whole set, and a long catalog was simply cut off by the
// result bound mid-row. Paging lets it ask for the next window instead.
type page struct {
	limit  int
	offset int
}

func readPage(params map[string]any, fallback, most int) page {
	limit := optionalInt(params, "limit", fallback)
	if limit <= 0 {
		limit = fallback
	}
	if limit > most {
		limit = most
	}

	return page{limit: limit, offset: max(optionalInt(params, "offset", 0), 0)}
}

// fetch is how many rows to ask for: one more than the page, so the answer
// knows whether another page exists without counting the whole set.
func (p page) fetch() int {
	return p.limit + 1
}

// trim cuts a fetched slice to the page and reports whether rows were left.
func trim[T any](p page, rows []T) ([]T, bool) {
	if len(rows) <= p.limit {
		return rows, false
	}

	return rows[:p.limit], true
}

// slicePage slices an in-memory list to the page.
func slicePage[T any](p page, rows []T) ([]T, bool) {
	if p.offset >= len(rows) {
		return nil, false
	}

	return trim(p, rows[p.offset:])
}

// pageSchema is the limit and offset parameters every paged tool takes.
func pageSchema(fallback, most int) map[string]any {
	return map[string]any{
		paramLimit: limitSchema(fallback, most),
		paramOffset: map[string]any{
			toolschema.KeyType:    toolschema.TypeInteger,
			toolschema.KeyMinimum: 0,
			toolschema.KeyDescription: "How many rows to skip. When a result says hasMore, " +
				"call again with its nextOffset for the next page.",
		},
	}
}

// limitSchema is a limit of one to most rows, read as the bound past it.
func limitSchema(fallback, most int) map[string]any {
	return toolschema.Clamped(map[string]any{
		toolschema.KeyType:    toolschema.TypeInteger,
		toolschema.KeyMinimum: 1,
		toolschema.KeyMaximum: most,
		toolschema.KeyDescription: fmt.Sprintf(
			"How many rows to return: %d unless you ask, at most %d.", fallback, most),
	})
}

// withPaging adds the paging parameters to a schema's properties.
func withPaging(properties map[string]any, fallback, most int) map[string]any {
	for name, property := range pageSchema(fallback, most) {
		properties[name] = property
	}

	return properties
}
