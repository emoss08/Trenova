package agentquerytoolservice

import (
	"reflect"
	"strings"

	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

// searchOutcome is what a search tool returns instead of a bare slice.
//
// A model handed `[]` cannot tell "nothing matched what you asked for" from
// "this system holds no such records", and it will guess. It guessed wrong in
// production: a driver search that matched nobody became "there are no driver
// records currently in the Trenova system", which is a confident claim about a
// customer's database made from a single empty result.
//
// Naming the terms back turns the same answer into something the model can act
// on — it can widen the search, or say plainly that nothing matched these terms.
type searchOutcome struct {
	Count          int      `json:"count"`
	Total          *int     `json:"total,omitempty"`
	TotalEstimated bool     `json:"totalEstimated,omitempty"`
	SearchedFor    []string `json:"searchedFor"`
	Items          any      `json:"items"`
	// Columns is the row projection's fields in the order it declares them.
	//
	// The Desk draws a list result as a table beside the conversation, and a
	// table needs its columns in an order somebody chose. JSON objects have
	// none: by the time the result reaches the pane it is a map, and a map
	// scatters "pro number, customer, status" into whatever order it hashes
	// to. So the order is taken here, where the rows are still a type.
	//
	// It is also the cheapest possible answer to "what can I ask about these",
	// which the model otherwise infers from whichever fields happened to be
	// populated in the first row.
	Columns []string `json:"columns,omitempty"`
	// Note is set only when nothing matched, or when the rows matched only as a
	// close spelling of the query, because a full result speaks for itself and
	// an extra sentence in front of it is noise in a context window.
	Note string `json:"note,omitempty"`
	// Offset, HasMore and NextOffset say where this page sits in the whole,
	// so a full page is never mistaken for the whole set.
	Offset     int    `json:"offset,omitempty"`
	HasMore    bool   `json:"hasMore"`
	NextOffset *int   `json:"nextOffset,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
	// resume is where a page cut short goes on, set only by a list whose rows
	// are in the list's own order. A type that embeds the outcome with it set
	// declares its own CutPage, or the cut would drop the type's own fields.
	resume *pageResume
}

// pageResume is where a page cut short goes on: each row's cursor under the
// call's cursor scope, or, for a list paged by offset, the offset after the
// rows kept.
type pageResume struct {
	scope    string
	cursors  []string
	byOffset bool
}

func (o searchOutcome) PageRows() int {
	list := reflect.ValueOf(o.Items)
	if list.Kind() != reflect.Slice {
		return 0
	}

	return list.Len()
}

func (o searchOutcome) CutPage(rows int) (any, bool) {
	cut, ok := o.cut(rows)
	if !ok {
		return nil, false
	}

	return cut, true
}

// cut is the outcome ending after its first rows rows, resuming at the row
// after them.
func (o *searchOutcome) cut(rows int) (searchOutcome, bool) {
	list := reflect.ValueOf(o.Items)
	if o.resume == nil || list.Kind() != reflect.Slice || rows <= 0 || rows >= list.Len() {
		return searchOutcome{}, false
	}
	cut := *o
	cut.Items = list.Slice(0, rows).Interface()
	cut.Count = rows
	cut.HasMore = true
	switch {
	case o.resume.byOffset:
		next := o.Offset + rows
		cut.NextOffset = &next
	case len(o.resume.cursors) == list.Len() && o.resume.cursors[rows-1] != "":
		cut.NextCursor = pagination.EncodeKeyCursor(o.resume.scope, o.resume.cursors[rows-1])
	default:
		return searchOutcome{}, false
	}

	return cut, true
}

// paged records where a page sits in the whole list.
func (o searchOutcome) paged(p page, hasMore bool) searchOutcome {
	o.Offset = p.offset
	o.HasMore = hasMore
	if hasMore {
		next := p.offset + p.limit
		o.NextOffset = &next
	}

	return o
}

func (o *searchOutcome) cursorPaged(window *cursorWindow, hasMore bool, next string) {
	o.HasMore = hasMore
	o.NextCursor = window.next(next)
}

// searchResult wraps what a repository returned. count is taken from the
// caller rather than derived by reflection so a tool stays in charge of what
// "how many" means for its own shape.
func searchResult(criteria *filtercatalog.Criteria, items any, count int) searchOutcome {
	outcome := searchOutcome{
		Count:       count,
		SearchedFor: criteria.Describe(),
		Items:       items,
		Columns:     columnsOf(items),
	}
	if count == 0 {
		outcome.Note = criteria.EmptyNote()
	}

	return outcome
}

// columnsOf reads a row projection's JSON field names in declaration order.
//
// It works off the first row because every row in one result is the same
// projection. A result with no rows has no columns to report, which is
// correct: there is no table to draw.
func columnsOf(items any) []string {
	list := reflect.ValueOf(items)
	if list.Kind() != reflect.Slice || list.Len() == 0 {
		return nil
	}

	rowType := reflect.Indirect(reflect.ValueOf(list.Index(0).Interface())).Type()
	if rowType.Kind() != reflect.Struct {
		return nil
	}

	names := make([]string, 0, rowType.NumField())

	return appendColumns(names, rowType)
}

// appendColumns walks one struct, descending into embedded structs so a row
// built from a shared base reports its inherited fields where they appear
// rather than as a nested object the grid cannot draw.
func appendColumns(names []string, rowType reflect.Type) []string {
	for i := range rowType.NumField() {
		field := rowType.Field(i)

		tag, ok := field.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}

		// An embedded struct's fields are promoted onto the row, and that
		// holds whether the embedded type is exported or not — which is why
		// this runs before the exported check rather than after it.
		if field.Anonymous && name == "" {
			embedded := field.Type
			if embedded.Kind() == reflect.Ptr {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				names = appendColumns(names, embedded)

				continue
			}
		}

		if !field.IsExported() {
			continue
		}
		if !ok || name == "" {
			name = field.Name
		}
		names = append(names, name)
	}

	return names
}
