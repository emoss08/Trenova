// Package reportrows is the wire shape of a report run's rows.
//
// It exists so the writer and the reader of that shape are the same
// definition. The rows are written by the reporting renderer as a run streams,
// and read back much later — by a different process, against a run made by a
// version of the code that has moved on — to compare two runs. A second copy
// of the field names would be a second thing to keep in step, and the day they
// drifted is the day a comparison reported a change nobody made.
//
// Values are raw, not formatted: decimals are exact strings and instants are
// unix seconds. A reader comparing "$2,840.00" with "$2,840.0" cannot tell a
// currency setting from a price.
package reportrows

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
)

type Meta struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	GeneratedAt int64          `json:"generatedAt"`
	Timezone    string         `json:"timezone,omitempty"`
	RequestedBy string         `json:"requestedBy,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
}

type Column struct {
	ID      string                   `json:"id"`
	Label   string                   `json:"label"`
	Type    reportcatalog.FieldType  `json:"type"`
	Format  reportcatalog.FormatHint `json:"format,omitempty"`
	Display reportfmt.Resolved       `json:"display"`
}

// IsMeasure reports whether the column holds a quantity that can be summed and
// subtracted. It is what decides, absent an explicit choice, which columns a
// comparison treats as numbers and which identify the row.
func (c Column) IsMeasure() bool {
	return c.Type == reportcatalog.FieldInt || c.Type == reportcatalog.FieldDecimal
}

// IsDimension reports whether the column is a stable identifier — the kind of
// value that says which row this is rather than how much of something it holds.
//
// JSON is neither: an unordered document has no reliable equality, so it can
// neither key a row nor be subtracted.
func (c Column) IsDimension() bool {
	switch c.Type {
	case reportcatalog.FieldString,
		reportcatalog.FieldEnum,
		reportcatalog.FieldRef,
		reportcatalog.FieldBool,
		reportcatalog.FieldEpoch:
		return true
	case reportcatalog.FieldInt, reportcatalog.FieldDecimal, reportcatalog.FieldJSON:
		return false
	default:
		return false
	}
}

type Summary struct {
	RowCount  int64 `json:"rowCount"`
	Truncated bool  `json:"truncated"`
}

// Envelope is one run's rows, positional against Schema.
//
// Totals is absent rather than null when the report has no grand-total row: a
// report that does not total is a different thing from one whose totals are
// zero.
type Envelope struct {
	Meta    Meta     `json:"meta"`
	Schema  []Column `json:"schema"`
	Rows    [][]any  `json:"rows"`
	Totals  []any    `json:"totals,omitempty"`
	Summary Summary  `json:"summary"`
}

var (
	ErrNoSchema  = errors.New("the stored rows carry no schema")
	ErrRowWidth  = errors.New("a stored row does not match the schema width")
	ErrMalformed = errors.New("the stored rows could not be read")
)

// Decode reads an envelope and refuses one that cannot be trusted positionally.
//
// The checks are not defensive habit: every consumer indexes a row by its
// column's position, so a row of the wrong width would silently compare one
// column against another and report differences that do not exist.
func Decode(data []byte) (*Envelope, error) {
	envelope := new(Envelope)
	if err := sonic.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformed, err)
	}

	if len(envelope.Schema) == 0 {
		return nil, ErrNoSchema
	}

	width := len(envelope.Schema)
	for i := range envelope.Rows {
		if len(envelope.Rows[i]) != width {
			return nil, fmt.Errorf(
				"%w: row %d has %d values against %d columns",
				ErrRowWidth, i, len(envelope.Rows[i]), width,
			)
		}
	}
	if envelope.Totals != nil && len(envelope.Totals) != width {
		return nil, fmt.Errorf(
			"%w: the totals row has %d values against %d columns",
			ErrRowWidth, len(envelope.Totals), width,
		)
	}

	return envelope, nil
}

// ColumnIndex maps column ids to their position, for looking up a caller's
// chosen keys and measures without scanning the schema per lookup.
func (e *Envelope) ColumnIndex() map[string]int {
	index := make(map[string]int, len(e.Schema))
	for i := range e.Schema {
		index[e.Schema[i].ID] = i
	}

	return index
}

// ColumnIDs returns the schema's column ids in order.
func (e *Envelope) ColumnIDs() []string {
	ids := make([]string, len(e.Schema))
	for i := range e.Schema {
		ids[i] = e.Schema[i].ID
	}

	return ids
}
