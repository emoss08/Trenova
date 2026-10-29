package querytool

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const cursorScopeLength = 16

var ErrOffsetOnCursorList = errors.New(
	"this list pages with after, the nextCursor of the previous page, not offset. Call " +
		"it again without offset, passing that nextCursor as after for the next page",
)

var ErrStaleCursor = errors.New(
	"after is not a nextCursor this tool returned for these same arguments: the query, " +
		"filters, status or sort changed since that page, or the cursor was altered. Call " +
		"it again without after to start from the first page",
)

// Page is one window of a list: how many rows, from where.
//
// A list used to take a limit and nothing else, and its answer carried no
// sign of whether more rows existed. A model handed a full page of 25 could
// not tell it from the whole set, and a long catalog was simply cut off by the
// result bound mid-row. Paging lets it ask for the next window instead.
type Page struct {
	Limit  int
	Offset int
}

func ReadPage(params map[string]any, fallback, most int) Page {
	return Page{
		Limit:  ReadLimit(params, fallback, most),
		Offset: max(OptionalInt(params, ParamOffset, 0), 0),
	}
}

func ReadLimit(params map[string]any, fallback, most int) int {
	limit := OptionalInt(params, ParamLimit, fallback)
	if limit <= 0 {
		return fallback
	}

	return min(limit, most)
}

// fetch is how many rows to ask for: one more than the page, so the answer
// knows whether another page exists without counting the whole set.
func (p Page) Fetch() int {
	return p.Limit + 1
}

// Trim cuts a fetched slice to the page and reports whether rows were left.
func Trim[T any](p Page, rows []T) ([]T, bool) {
	if len(rows) <= p.Limit {
		return rows, false
	}

	return rows[:p.Limit], true
}

// SlicePage slices an in-memory list to the page.
func SlicePage[T any](p Page, rows []T) ([]T, bool) {
	if p.Offset >= len(rows) {
		return nil, false
	}

	return Trim(p, rows[p.Offset:])
}

// PageSchema is the limit and offset parameters every paged tool takes.
func PageSchema(fallback, most int) map[string]any {
	return map[string]any{
		ParamLimit: LimitSchema(fallback, most),
		ParamOffset: map[string]any{
			toolschema.KeyType:    toolschema.TypeInteger,
			toolschema.KeyMinimum: 0,
			toolschema.KeyDescription: "How many rows to skip. When a result says hasMore, " +
				"call again with its nextOffset for the next page.",
		},
	}
}

// LimitSchema is a limit of one to most rows, read as the bound past it.
func LimitSchema(fallback, most int) map[string]any {
	return toolschema.Clamped(map[string]any{
		toolschema.KeyType:    toolschema.TypeInteger,
		toolschema.KeyMinimum: 1,
		toolschema.KeyMaximum: most,
		toolschema.KeyDescription: fmt.Sprintf(
			"How many rows to return: %d unless you ask, at most %d.", fallback, most),
	})
}

// WithPaging adds the paging parameters to a schema's properties.
func WithPaging(properties map[string]any, fallback, most int) map[string]any {
	for name, property := range PageSchema(fallback, most) {
		properties[name] = property
	}

	return properties
}

type CursorWindow struct {
	Limit int
	after string
	Scope string
	Info  pagination.CursorInfo
}

func ReadCursorWindow(
	tool string,
	params map[string]any,
	fallback, most int,
) (CursorWindow, error) {
	if OptionalInt(params, ParamOffset, 0) != 0 {
		return CursorWindow{}, ErrOffsetOnCursorList
	}
	scope, err := CursorScope(tool, params)
	if err != nil {
		return CursorWindow{}, err
	}

	window := CursorWindow{
		Limit: ReadLimit(params, fallback, most),
		after: OptionalString(params, ParamAfter),
		Scope: scope,
	}
	key := ""
	if window.after != "" {
		if key, err = pagination.DecodeKeyCursor(scope, window.after); err != nil {
			return CursorWindow{}, ErrStaleCursor
		}
	}
	if window.Info, err = pagination.NewCursorInfo(window.Limit, key); err != nil {
		return CursorWindow{}, ErrStaleCursor
	}
	window.Info.IncludeTotalCount = false

	return window, nil
}

// OffsetWindow is a window of a list that pages by offset under the cursor
// contract: the offset never reaches the model, only a nextCursor bound to
// the call's other arguments, so it learns one convention for every list.
type OffsetWindow struct {
	Limit  int
	Offset int
	Scope  string
}

func ReadOffsetWindow(
	tool string,
	params map[string]any,
	fallback, most int,
) (OffsetWindow, error) {
	if OptionalInt(params, ParamOffset, 0) != 0 {
		return OffsetWindow{}, ErrOffsetOnCursorList
	}
	scope, err := CursorScope(tool, params)
	if err != nil {
		return OffsetWindow{}, err
	}

	window := OffsetWindow{Limit: ReadLimit(params, fallback, most), Scope: scope}
	if after := OptionalString(params, ParamAfter); after != "" {
		key, decodeErr := pagination.DecodeKeyCursor(scope, after)
		if decodeErr != nil {
			return OffsetWindow{}, ErrStaleCursor
		}
		offset, parseErr := strconv.Atoi(key)
		if parseErr != nil || offset < 0 {
			return OffsetWindow{}, ErrStaleCursor
		}
		window.Offset = offset
	}

	return window, nil
}

// fetch is how many rows to ask for: one more than the page, so the answer
// knows whether another page exists, within the server's page bound.
func (w OffsetWindow) Fetch(most int) int {
	return min(w.Limit+1, most)
}

// more reports whether rows remain past the page, given what a fetch
// returned. A fetch the server's bound held to the page itself cannot tell,
// and says there may be more when it came back full.
func (w OffsetWindow) More(fetched, most int) bool {
	return fetched > w.Limit || (w.Fetch(most) == w.Limit && fetched == w.Limit)
}

func (w OffsetWindow) Next() string {
	return pagination.EncodeKeyCursor(w.Scope, strconv.Itoa(w.Offset+w.Limit))
}

func FirstCursorWindow(limit int) CursorWindow {
	return CursorWindow{Limit: limit, Info: pagination.CursorInfo{Limit: limit}}
}

func CursorScope(tool string, params map[string]any) (string, error) {
	shape := make(map[string]any, len(params))
	for key, value := range params {
		if key == ParamLimit || key == ParamAfter || key == ParamOffset ||
			key == agenttoolports.ListFieldsParam {
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

func (w *CursorWindow) First() bool {
	return w.after == ""
}

func (w *CursorWindow) Next(cursor string) string {
	if cursor == "" {
		return ""
	}

	return pagination.EncodeKeyCursor(w.Scope, cursor)
}

func RefuseStaleCursor(err error) error {
	var invalid *errortypes.Error
	if errors.As(err, &invalid) && invalid.Field == ParamAfter {
		return ErrStaleCursor
	}

	return err
}

func CursorPageSchema(fallback, most int) map[string]any {
	return map[string]any{
		ParamLimit: LimitSchema(fallback, most),
		ParamAfter: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The nextCursor of the previous page, with every other " +
				"argument unchanged; omit for the first page.",
		},
	}
}
