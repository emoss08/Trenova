package reportjobs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func moneyColumn(label string, rules ...format.Rule) table.Column {
	return table.Column{
		ID:    strings.ToLower(label),
		Label: label,
		Type:  value.Decimal,
		Display: format.Resolve(
			value.Decimal,
			format.Hint{Kind: format.Money},
			&format.Spec{Rules: rules},
		),
	}
}

func textColumn(label string) table.Column {
	return table.Column{
		ID:      strings.ToLower(label),
		Label:   label,
		Type:    value.String,
		Display: format.Resolve(value.String, format.Hint{}, nil),
	}
}

type digestRow struct {
	level  table.Level
	values []any
}

func detail(values ...any) digestRow { return digestRow{level: table.Detail, values: values} }

func total(values ...any) digestRow { return digestRow{level: table.Total, values: values} }

func resultTable(t *testing.T, columns []table.Column, rows ...digestRow) *table.Table {
	t.Helper()
	builder := table.NewBuilder(columns)
	for _, row := range rows {
		require.NoError(t, builder.AppendLevel(row.level, row.values...))
	}
	return builder.Table()
}

func captureDigest(
	t *testing.T,
	limit int,
	columns []table.Column,
	rows ...digestRow,
) *reportports.ReportDigest {
	t.Helper()
	capture := newDigestCapture(limit)
	result := resultTable(t, columns, rows...)
	reader := result.Reader()
	for {
		batch, err := reader.Next(t.Context())
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		capture.capture(columns, batch)
	}
	digest, err := capture.digest(t.Context(), columns)
	require.NoError(t, err)
	return digest
}

func digestDetailRows(t *testing.T, digest *reportports.ReportDigest) int {
	t.Helper()
	envelope, err := readDigest(digest)
	require.NoError(t, err)
	require.NotNil(t, envelope)
	return int(envelope.RowCount)
}

func TestRunReaderHandsEveryBatchToTheRendererAndTheSidecar(t *testing.T) {
	columns := []table.Column{textColumn("Customer"), moneyColumn("Revenue")}
	source := resultTable(t, columns, detail("ACME", "1000"), detail("Globex", "2000"))
	capture := newDigestCapture(10)
	beats := 0
	reader := newRunReader(source.Reader(), func() { beats++ }, 1, capture)

	sideRows := make(chan int, 1)
	go func() {
		side := reader.Sidecar()
		seen := 0
		for {
			batch, err := side.Next(context.Background())
			if err != nil {
				sideRows <- seen
				return
			}
			seen += batch.Len
		}
	}()

	seen := 0
	for {
		batch, err := reader.Next(t.Context())
		if err != nil {
			require.ErrorIs(t, err, io.EOF)
			break
		}
		seen += batch.Len
	}

	assert.Equal(t, 2, seen, "the renderer still receives every row")
	assert.Equal(t, 2, <-sideRows, "the stored rows receive every row")
	assert.Positive(t, beats, "a long render keeps heartbeating")
	digest, err := capture.digest(t.Context(), columns)
	require.NoError(t, err)
	assert.Equal(t, 2, digestDetailRows(t, digest))
	assert.False(t, digest.Truncated)
}

func TestDigestStopsAtTheLimitAndSaysSo(t *testing.T) {
	columns := []table.Column{textColumn("Name"), moneyColumn("Revenue")}
	rows := make([]digestRow, 0, 50)
	for range 50 {
		rows = append(rows, detail("row", "1"))
	}

	digest := captureDigest(t, 5, columns, rows...)

	assert.Equal(t, 5, digestDetailRows(t, digest))
	assert.True(t, digest.Truncated)
}

func TestDigestLimitIsCappedGlobally(t *testing.T) {
	assert.Equal(t, reportports.MaxDigestRows, newDigestCapture(10_000).limit)
}

func TestDigestKeepsTheGrandTotal(t *testing.T) {
	columns := []table.Column{textColumn("Customer"), moneyColumn("Revenue")}

	digest := captureDigest(t, 5, columns,
		detail("ACME", "80000"), total(nil, "80000"))

	got := digestTotal(digest, "revenue")
	require.NotNil(t, got)
	assert.InDelta(t, 80000.0, *got, 0.001)
	assert.Nil(t, digestTotal(digest, "missing"), "a measure that vanished is not zero")
}

func TestDigestContextFormatsThroughTheDisplayContract(t *testing.T) {
	columns := []table.Column{
		textColumn("Customer"),
		moneyColumn("Revenue", format.Rule{
			Op: format.RuleGreater, Value: 1500, Tone: format.ToneNegative,
		}),
	}
	digest := captureDigest(t, 5, columns,
		detail("ACME", "2000"), detail("Globex", "500"), total(nil, "2500"))

	out, err := digestContext(t.Context(), digest, nil, "en", 2)
	require.NoError(t, err)

	assert.Equal(t, []string{"Customer", "Revenue"}, out.Headers)
	require.Len(t, out.Rows, 2)
	assert.Equal(t, "$2,000.00", out.Rows[0].Cells[1].Text)
	assert.Equal(t, "$500.00", out.Rows[1].Cells[1].Text)
	assert.Equal(t, string(format.ToneNegative), out.Rows[0].Cells[1].Tone)
	assert.Empty(t, out.Rows[1].Cells[1].Tone, "the value under the threshold is not toned")
	require.Len(t, out.Totals, 2)
	assert.Equal(t, "Total", out.Totals[0], "the leading dimension is labelled rather than blank")
	assert.Equal(t, "$2,500.00", out.Totals[1])
}

func TestDigestContextIsEmptyWhenThereIsNothingToShow(t *testing.T) {
	none, err := digestContext(t.Context(), nil, nil, "en", 0)
	require.NoError(t, err)
	assert.Empty(t, none.Rows)
	empty, err := digestContext(t.Context(), &reportports.ReportDigest{}, nil, "en", 0)
	require.NoError(t, err)
	assert.Empty(t, empty.Rows)

	headersOnly := captureDigest(t, 5, []table.Column{textColumn("Customer")})
	out, err := digestContext(t.Context(), headersOnly, nil, "en", 0)
	require.NoError(t, err)
	assert.Empty(t, out.Rows)
	assert.Empty(t, out.Headers, "no rows means no table, so no headers either")
}

func TestDigestNoteOnlyAppearsWhenRowsWereDropped(t *testing.T) {
	columns := []table.Column{textColumn("Customer")}
	full := captureDigest(t, 5, columns, detail("ACME"))
	assert.Empty(t, digestNote(t.Context(), full, 1, 1))

	sampled := captureDigest(t, 1, columns, detail("ACME"), detail("Globex"))
	assert.Contains(t, digestNote(t.Context(), sampled, 1, 4000), "first 1 of 4000 rows")
}

func TestRunSampleKeepsTheCapturedEnvelope(t *testing.T) {
	t.Parallel()

	columns := []table.Column{textColumn("Customer"), moneyColumn("Revenue")}
	digest := captureDigest(t, 2, columns,
		detail("ACME", "1180.50"),
		detail("Globex", "980"),
		detail("Initech", "640"),
		total(nil, "2800.50"),
	)

	sample := runSampleOf(digest)

	require.NotNil(t, sample)
	assert.Equal(t, digest.Envelope, sample)
}

func TestRunSampleIsNilWithoutACaptureOrPastItsCap(t *testing.T) {
	t.Parallel()

	assert.Nil(t, runSampleOf(nil))
	assert.Nil(t, runSampleOf(&reportports.ReportDigest{}))
	assert.Nil(t, runSampleOf(&reportports.ReportDigest{
		Envelope: make([]byte, report.MaxRunSampleBytes+1),
	}))
}
