package reportjobs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeReader streams a fixed set of rows, standing in for the executor.
type fakeReader struct {
	schema []services.ReportResultColumn
	rows   []services.ReportRow
	totals services.ReportRow
	cursor int
	closed bool
}

func (f *fakeReader) Schema() []services.ReportResultColumn { return f.schema }

func (f *fakeReader) Next(_ context.Context) (services.ReportRow, error) {
	if f.cursor >= len(f.rows) {
		return nil, io.EOF
	}
	row := f.rows[f.cursor]
	f.cursor++
	return row, nil
}

func (f *fakeReader) Totals(_ context.Context) (services.ReportRow, error) { return f.totals, nil }
func (f *fakeReader) RowCount() int64                                     { return int64(f.cursor) }
func (f *fakeReader) Truncated() bool                                     { return false }
func (f *fakeReader) Close() error                                        { f.closed = true; return nil }

func moneyColumn(label string, rules ...reportfmt.Rule) services.ReportResultColumn {
	return services.ReportResultColumn{
		ID:    strings.ToLower(label),
		Label: label,
		Type:  reportcatalog.FieldDecimal,
		Display: reportfmt.Resolve(reportfmt.ResolveInput{
			Type: reportcatalog.FieldDecimal,
			Hint: reportcatalog.FormatMoney,
			Spec: &reportfmt.Spec{Rules: rules},
		}),
	}
}

func textColumn(label string) services.ReportResultColumn {
	return services.ReportResultColumn{
		ID:      strings.ToLower(label),
		Label:   label,
		Type:    reportcatalog.FieldString,
		Display: reportfmt.Resolve(reportfmt.ResolveInput{Type: reportcatalog.FieldString}),
	}
}

func drainDigest(t *testing.T, reader *digestReader) *services.ReportDigest {
	t.Helper()
	for {
		if _, err := reader.Next(t.Context()); err != nil {
			require.True(t, errors.Is(err, io.EOF))
			break
		}
	}
	return reader.digest(t.Context())
}

// The digest rides along with the render pass rather than costing a second
// query, so every row the renderer sees must still reach it.
func TestDigestCapturesRowsWhilePassingThemThrough(t *testing.T) {
	inner := &fakeReader{
		schema: []services.ReportResultColumn{textColumn("Customer"), moneyColumn("Revenue")},
		rows: []services.ReportRow{
			{"ACME", decimal.NewFromInt(1000)},
			{"Globex", decimal.NewFromInt(2000)},
		},
	}
	reader := newDigestReader(inner, 10)

	var seen int
	for {
		row, err := reader.Next(t.Context())
		if err != nil {
			break
		}
		seen++
		assert.Len(t, row, 2)
	}

	assert.Equal(t, 2, seen, "the renderer still receives every row")
	digest := reader.digest(t.Context())
	assert.Len(t, digest.Rows, 2)
	assert.False(t, digest.Truncated)
}

// An email is a summary: past a couple of dozen rows the table stops being
// readable, and the attachment is the right answer.
func TestDigestStopsAtTheLimitAndSaysSo(t *testing.T) {
	rows := make([]services.ReportRow, 0, 50)
	for i := range 50 {
		rows = append(rows, services.ReportRow{"row", decimal.NewFromInt(int64(i))})
	}
	inner := &fakeReader{
		schema: []services.ReportResultColumn{textColumn("Name"), moneyColumn("Revenue")},
		rows:   rows,
	}

	digest := drainDigest(t, newDigestReader(inner, 5))

	assert.Len(t, digest.Rows, 5)
	assert.True(t, digest.Truncated)
	assert.Equal(t, 50, inner.cursor, "every row still streams to the renderer")
}

func TestDigestLimitIsCappedGlobally(t *testing.T) {
	inner := &fakeReader{schema: []services.ReportResultColumn{textColumn("Name")}}
	assert.Equal(t, services.MaxDigestRows, newDigestReader(inner, 10_000).limit)
}

// The renderer owns the slice it is handed, so a digest that kept the reference
// would show whatever the last row happened to leave behind.
func TestDigestCopiesRowsItKeeps(t *testing.T) {
	shared := services.ReportRow{"ACME", decimal.NewFromInt(1)}
	inner := &fakeReader{
		schema: []services.ReportResultColumn{textColumn("Customer"), moneyColumn("Revenue")},
		rows:   []services.ReportRow{shared},
	}
	reader := newDigestReader(inner, 5)

	row, err := reader.Next(t.Context())
	require.NoError(t, err)
	row[0] = "MUTATED"

	digest := reader.digest(t.Context())
	assert.Equal(t, "ACME", digest.Rows[0][0])
}

// Every cell renders through its column's display, so the email agrees with the
// grid and the exports — including the conditional-format tones.
func TestDigestHTMLFormatsThroughTheDisplayContract(t *testing.T) {
	digest := &services.ReportDigest{
		Columns: []services.ReportResultColumn{
			textColumn("Customer"),
			moneyColumn("Revenue", reportfmt.Rule{
				Op: reportfmt.RuleGreaterThan, Value: 1500, Tone: reportfmt.ToneNegative,
			}),
		},
		Rows: []services.ReportRow{
			{"ACME", decimal.NewFromInt(2000)},
			{"Globex", decimal.NewFromInt(500)},
		},
		Totals: services.ReportRow{nil, decimal.NewFromInt(2500)},
	}

	out := renderDigestHTML(digest, nil)

	assert.Contains(t, out, "$2,000.00")
	assert.Contains(t, out, "$500.00")
	assert.Contains(t, out, "Customer")
	// The toned cell is coloured; the one under the threshold is not.
	assert.Contains(t, out, digestToneColors[reportfmt.ToneNegative])
	assert.Equal(t, 1, strings.Count(out, digestToneColors[reportfmt.ToneNegative]))
	// Numbers right-align, text left-aligns.
	assert.Contains(t, out, "text-align:right;")
	assert.Contains(t, out, "text-align:left;")
	// Totals row, with the leading dimension labelled rather than blank.
	assert.Contains(t, out, "$2,500.00")
	assert.Contains(t, out, "Total")
}

// An email body is untrusted output: a customer named with markup must not
// become markup.
func TestDigestHTMLEscapesValues(t *testing.T) {
	digest := &services.ReportDigest{
		Columns: []services.ReportResultColumn{textColumn("Customer")},
		Rows:    []services.ReportRow{{"<script>alert(1)</script>"}},
	}

	out := renderDigestHTML(digest, nil)

	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestDigestRendersNothingWhenEmpty(t *testing.T) {
	assert.Empty(t, renderDigestHTML(nil, nil))
	assert.Empty(t, renderDigestText(nil, nil))
	assert.Empty(t, renderDigestHTML(&services.ReportDigest{}, nil))

	// Columns but no rows is a real case — an alert that fired on an empty
	// result should not draw a headers-only table.
	headersOnly := &services.ReportDigest{
		Columns: []services.ReportResultColumn{textColumn("Customer")},
	}
	assert.Empty(t, renderDigestHTML(headersOnly, nil))
}

// Nobody should reconcile against a sample believing it is the whole result.
func TestDigestNoteOnlyAppearsWhenRowsWereDropped(t *testing.T) {
	full := &services.ReportDigest{
		Columns: []services.ReportResultColumn{textColumn("Customer")},
		Rows:    []services.ReportRow{{"ACME"}},
	}
	assert.Empty(t, digestNote(full, 1))

	sampled := &services.ReportDigest{
		Columns:   full.Columns,
		Rows:      full.Rows,
		Truncated: true,
	}
	assert.Contains(t, digestNote(sampled, 4000), "first 1 of 4000 rows")
}

func TestDigestTextIsLabelledRatherThanColumnar(t *testing.T) {
	digest := &services.ReportDigest{
		Columns: []services.ReportResultColumn{textColumn("Customer"), moneyColumn("Revenue")},
		Rows:    []services.ReportRow{{"ACME", decimal.NewFromInt(1000)}},
	}

	out := renderDigestText(digest, nil)

	assert.Contains(t, out, "Customer: ACME")
	assert.Contains(t, out, "Revenue: $1,000.00")
}
