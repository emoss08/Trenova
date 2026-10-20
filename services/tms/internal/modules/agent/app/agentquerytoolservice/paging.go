package agentquerytoolservice

import (
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const cursorScopeLength = 16

var errOffsetOnCursorList = errors.New(
	"this list pages with after, the nextCursor of the previous page, not offset. Call " +
		"it again without offset, passing that nextCursor as after for the next page",
)

var errStaleCursor = errors.New(
	"after is not a nextCursor this tool returned for these same arguments: the query, " +
		"filters, status or sort changed since that page, or the cursor was altered. Call " +
		"it again without after to start from the first page",
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
	return page{
		limit:  readLimit(params, fallback, most),
		offset: max(optionalInt(params, paramOffset, 0), 0),
	}
}

func readLimit(params map[string]any, fallback, most int) int {
	limit := optionalInt(params, paramLimit, fallback)
	if limit <= 0 {
		return fallback
	}

	return min(limit, most)
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

type cursorWindow struct {
	limit int
	after string
	scope string
	info  pagination.CursorInfo
}

func readCursorWindow(
	tool string,
	params map[string]any,
	fallback, most int,
) (cursorWindow, error) {
	if optionalInt(params, paramOffset, 0) != 0 {
		return cursorWindow{}, errOffsetOnCursorList
	}
	scope, err := cursorScope(tool, params)
	if err != nil {
		return cursorWindow{}, err
	}

	window := cursorWindow{
		limit: readLimit(params, fallback, most),
		after: optionalString(params, paramAfter),
		scope: scope,
	}
	key := ""
	if window.after != "" {
		if key, err = pagination.DecodeKeyCursor(scope, window.after); err != nil {
			return cursorWindow{}, errStaleCursor
		}
	}
	if window.info, err = pagination.NewCursorInfo(window.limit, key); err != nil {
		return cursorWindow{}, errStaleCursor
	}
	window.info.IncludeTotalCount = false

	return window, nil
}

func firstCursorWindow(limit int) cursorWindow {
	return cursorWindow{limit: limit, info: pagination.CursorInfo{Limit: limit}}
}

func cursorScope(tool string, params map[string]any) (string, error) {
	shape := make(map[string]any, len(params))
	for key, value := range params {
		if key == paramLimit || key == paramAfter || key == paramOffset {
			continue
		}
		shape[key] = value
	}

	encoded, err := jsonutils.CanonicalMarshal(shape)
	if err != nil {
		return "", fmt.Errorf("read the arguments a page cursor is bound to: %w", err)
	}

	return hashutils.SHA256Hex(tool + ":" + string(encoded))[:cursorScopeLength], nil
}

func (w *cursorWindow) first() bool {
	return w.after == ""
}

func (w *cursorWindow) next(cursor string) string {
	if cursor == "" {
		return ""
	}

	return pagination.EncodeKeyCursor(w.scope, cursor)
}

func refuseStaleCursor(err error) error {
	var invalid *errortypes.Error
	if errors.As(err, &invalid) && invalid.Field == paramAfter {
		return errStaleCursor
	}

	return err
}

func cursorPageSchema(fallback, most int) map[string]any {
	return map[string]any{
		paramLimit: limitSchema(fallback, most),
		paramAfter: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The nextCursor of the previous page, with every other " +
				"argument unchanged; omit for the first page.",
		},
	}
}
