package render

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The sidecar's whole promise is that a CSV, XLSX or PDF run becomes as
// diffable as a JSON one. That only holds if what it writes is what the JSON
// renderer would have written for the same dataset — byte for byte, since a
// diff reads these two objects as interchangeable.
func TestRowsSidecarMatchesJSONRender(t *testing.T) {
	var direct bytes.Buffer
	_, err := NewJSON().Render(t.Context(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows(), totals: testTotals()},
		Sink:    &direct,
		Meta:    testMeta(),
	})
	require.NoError(t, err)

	var observed bytes.Buffer
	sidecar := NewRowsSidecar(
		&fakeReader{schema: testSchema(), rows: testRows(), totals: testTotals()},
		&observed,
		testMeta(),
	)

	// A format that is not JSON, driving the same reader.
	stats, err := NewCSV(CSVParams{Config: testConfig()}).Render(
		t.Context(),
		&services.ReportRenderRequest{Dataset: sidecar, Sink: &bytes.Buffer{}, Meta: testMeta()},
	)
	require.NoError(t, err)
	require.NoError(t, sidecar.Finish(t.Context()))

	assert.Equal(t, int64(3), stats.Rows)
	assert.Equal(t, direct.String(), observed.String())
}

// PDF never asks for totals, so the sidecar has to fetch them itself or a PDF
// run would diff against a CSV run with a column of totals missing.
func TestRowsSidecarWritesTotalsRendererNeverAsked(t *testing.T) {
	var buf bytes.Buffer
	reader := &fakeReader{schema: testSchema(), rows: testRows(), totals: testTotals()}
	sidecar := NewRowsSidecar(reader, &buf, testMeta())

	for {
		if _, err := sidecar.Next(t.Context()); err != nil {
			break
		}
	}
	require.NoError(t, sidecar.Finish(t.Context()))

	doc := decodeSidecar(t, &buf)
	require.Len(t, doc.Totals, 5)
	assert.Equal(t, "1135.4", doc.Totals[1])
	assert.Nil(t, doc.Totals[0])
}

// The totals row is an aggregate query, not a buffered value, so asking twice
// is a second round trip to the database for an answer already in hand.
func TestRowsSidecarAsksForTotalsOnce(t *testing.T) {
	reader := &countingTotalsReader{
		fakeReader: fakeReader{schema: testSchema(), rows: testRows(), totals: testTotals()},
	}
	sidecar := NewRowsSidecar(reader, &bytes.Buffer{}, testMeta())

	_, err := sidecar.Totals(t.Context())
	require.NoError(t, err)
	_, err = sidecar.Totals(t.Context())
	require.NoError(t, err)
	require.NoError(t, sidecar.Finish(t.Context()))

	assert.Equal(t, 1, reader.calls)
}

// A truncated run must produce a truncated sidecar carrying the flag: a diff
// that mistakes a row cap for deleted rows reports a change nobody made.
func TestRowsSidecarCarriesTruncation(t *testing.T) {
	var buf bytes.Buffer
	sidecar := NewRowsSidecar(
		&fakeReader{schema: testSchema(), rows: testRows(), truncated: true},
		&buf,
		testMeta(),
	)

	_, err := NewCSV(CSVParams{Config: testConfig()}).Render(
		t.Context(),
		&services.ReportRenderRequest{Dataset: sidecar, Sink: &bytes.Buffer{}, Meta: testMeta()},
	)
	require.NoError(t, err)
	require.NoError(t, sidecar.Finish(t.Context()))

	doc := decodeSidecar(t, &buf)
	assert.True(t, doc.Summary.Truncated)
	assert.Equal(t, int64(3), doc.Summary.RowCount)
	assert.Len(t, doc.Rows, 3)
}

// A report with no total row is not the same as one whose totals are zero, so
// the key is absent rather than null.
func TestRowsSidecarOmitsAbsentTotals(t *testing.T) {
	var buf bytes.Buffer
	sidecar := NewRowsSidecar(
		&fakeReader{schema: testSchema(), rows: testRows()},
		&buf,
		testMeta(),
	)

	for {
		if _, err := sidecar.Next(t.Context()); err != nil {
			break
		}
	}
	require.NoError(t, sidecar.Finish(t.Context()))

	assert.NotContains(t, buf.String(), `"totals"`)
	doc := decodeSidecar(t, &buf)
	assert.Len(t, doc.Rows, 3)
}

// An empty result still has to be a readable envelope: the diff tool
// distinguishes a run with no rows from a run with no sidecar, and it cannot
// do that if the empty one is zero bytes.
func TestRowsSidecarEmptyResultIsStillAnEnvelope(t *testing.T) {
	var buf bytes.Buffer
	sidecar := NewRowsSidecar(&fakeReader{schema: testSchema()}, &buf, testMeta())

	_, err := sidecar.Next(t.Context())
	require.Error(t, err)
	require.NoError(t, sidecar.Finish(t.Context()))

	doc := decodeSidecar(t, &buf)
	assert.Empty(t, doc.Rows)
	assert.Equal(t, int64(0), doc.Summary.RowCount)
	assert.Len(t, doc.Schema, 5)
}

// The rows flowing to the renderer must be the reader's own values, untouched:
// the sidecar observes, it does not transform.
func TestRowsSidecarPassesRowsThroughUnchanged(t *testing.T) {
	reader := &fakeReader{schema: testSchema(), rows: testRows()}
	sidecar := NewRowsSidecar(reader, &bytes.Buffer{}, testMeta())

	row, err := sidecar.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, testRows()[0], row)
}

// A sidecar that cannot be written must not take the run down with it. The
// artifact the person asked for is the deliverable; the sidecar is how a later
// comparison becomes possible, and a run without one is a state the reader
// already understands.
func TestRowsSidecarFailureIsReportedNotRaised(t *testing.T) {
	// The envelope buffers, so the refusal lands on the flush inside Finish —
	// which is exactly where a real size cap bites.
	sink := &refusingWriter{}
	sidecar := NewRowsSidecar(&fakeReader{schema: testSchema(), rows: testRows()}, sink, testMeta())

	stats, err := NewCSV(CSVParams{Config: testConfig()}).Render(
		t.Context(),
		&services.ReportRenderRequest{Dataset: sidecar, Sink: &bytes.Buffer{}, Meta: testMeta()},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Rows)

	require.ErrorIs(t, sidecar.Finish(t.Context()), errSinkFull)
}

func testTotals() services.ReportRow {
	return services.ReportRow{
		nil,
		decimal.RequireFromString("1135.4000"),
		int64(12),
		nil,
		nil,
	}
}

type sidecarDoc struct {
	Meta struct {
		Title string `json:"title"`
	} `json:"meta"`
	Schema []struct {
		ID string `json:"id"`
	} `json:"schema"`
	Rows    [][]any `json:"rows"`
	Totals  []any   `json:"totals"`
	Summary struct {
		RowCount  int64 `json:"rowCount"`
		Truncated bool  `json:"truncated"`
	} `json:"summary"`
}

func decodeSidecar(t *testing.T, buf *bytes.Buffer) sidecarDoc {
	t.Helper()

	var doc sidecarDoc
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &doc))

	return doc
}

type countingTotalsReader struct {
	fakeReader
	calls int
}

func (r *countingTotalsReader) Totals(ctx context.Context) (services.ReportRow, error) {
	r.calls++

	return r.fakeReader.Totals(ctx)
}

var errSinkFull = errors.New("sink full")

// refusingWriter stands in for the artifact size cap, which is the way a
// sidecar actually fails in production.
type refusingWriter struct{}

func (w *refusingWriter) Write(_ []byte) (int, error) { return 0, errSinkFull }

// The sidecar is written here and read back by pkg/reportrows somewhere else
// entirely, against a run made by a version of the code that has moved on. If
// the two definitions ever part company, a comparison starts reporting changes
// nobody made — so the writer is held to the reader's own decoder.
func TestRowsSidecarDecodesThroughTheSharedShape(t *testing.T) {
	var buf bytes.Buffer
	sidecar := NewRowsSidecar(
		&fakeReader{schema: testSchema(), rows: testRows(), totals: testTotals()},
		&buf,
		testMeta(),
	)

	_, err := NewCSV(CSVParams{Config: testConfig()}).Render(
		t.Context(),
		&services.ReportRenderRequest{Dataset: sidecar, Sink: &bytes.Buffer{}, Meta: testMeta()},
	)
	require.NoError(t, err)
	require.NoError(t, sidecar.Finish(t.Context()))

	envelope, err := reportrows.Decode(buf.Bytes())
	require.NoError(t, err)

	assert.Equal(t, "Revenue by Customer", envelope.Meta.Title)
	assert.Equal(t, "America/Chicago", envelope.Meta.Timezone)
	require.Len(t, envelope.Schema, 5)
	assert.Equal(t, reportcatalog.FieldDecimal, envelope.Schema[1].Type)
	assert.Equal(t, reportcatalog.FormatMoney, envelope.Schema[1].Format)
	assert.True(t, envelope.Schema[1].IsMeasure())
	assert.True(t, envelope.Schema[0].IsDimension())

	require.Len(t, envelope.Rows, 3)
	assert.Equal(t, "1234.5", envelope.Rows[0][1])
	assert.Nil(t, envelope.Rows[2][1])
	require.Len(t, envelope.Totals, 5)
	assert.Equal(t, "1135.4", envelope.Totals[1])
	assert.Equal(t, int64(3), envelope.Summary.RowCount)
}
