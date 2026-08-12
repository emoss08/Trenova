package render

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

type fakeReader struct {
	schema    []services.ReportResultColumn
	rows      []services.ReportRow
	generate  func(i int64) services.ReportRow
	totals    services.ReportRow
	total     int64
	pos       int64
	truncated bool
}

func (f *fakeReader) Schema() []services.ReportResultColumn { return f.schema }

func (f *fakeReader) Next(_ context.Context) (services.ReportRow, error) {
	if f.generate != nil {
		if f.pos >= f.total {
			return nil, io.EOF
		}
		row := f.generate(f.pos)
		f.pos++
		return row, nil
	}
	if f.pos >= int64(len(f.rows)) {
		return nil, io.EOF
	}
	row := f.rows[f.pos]
	f.pos++
	return row, nil
}

func (f *fakeReader) Totals(_ context.Context) (services.ReportRow, error) { return f.totals, nil }

func (f *fakeReader) RowCount() int64 { return f.pos }

func (f *fakeReader) Truncated() bool { return f.truncated }

func (f *fakeReader) Close() error { return nil }

// testColumn mirrors what the compiler emits: every column carries a resolved
// display contract derived from its type and format hint.
func testColumn(
	id, label string,
	fieldType reportcatalog.FieldType,
	hint reportcatalog.FormatHint,
) services.ReportResultColumn {
	return services.ReportResultColumn{
		ID:     id,
		Label:  label,
		Type:   fieldType,
		Format: hint,
		Display: reportfmt.Resolve(reportfmt.ResolveInput{
			Type: fieldType,
			Hint: hint,
		}),
	}
}

func testSchema() []services.ReportResultColumn {
	return []services.ReportResultColumn{
		testColumn("c0", "Customer", reportcatalog.FieldString, reportcatalog.FormatNone),
		testColumn("c1", "Total Charge", reportcatalog.FieldDecimal, reportcatalog.FormatMoney),
		testColumn("c2", "Shipments", reportcatalog.FieldInt, reportcatalog.FormatCount),
		testColumn("c3", "Created", reportcatalog.FieldEpoch, reportcatalog.FormatNone),
		testColumn("c4", "Active", reportcatalog.FieldBool, reportcatalog.FormatNone),
	}
}

func testRows() []services.ReportRow {
	return []services.ReportRow{
		{"ACME, Inc.", decimal.RequireFromString("1234.5000"), int64(12), int64(1784131200), true},
		{
			`Quote "Freight"`,
			decimal.RequireFromString("-99.1000"),
			int64(0),
			int64(1784131200),
			false,
		},
		{"Ünïcode Cargo 🚚", nil, nil, nil, nil},
	}
}

func testMeta() services.ReportRunMeta {
	return services.ReportRunMeta{
		Title:           "Revenue by Customer",
		Description:     "Total charges grouped by customer",
		GeneratedAtUnix: 1784131200,
		Timezone:        "America/Chicago",
		RequestedBy:     "Test User",
		Params:          map[string]any{"status": "InTransit"},
	}
}

func testConfig() *config.Config {
	return &config.Config{}
}

func TestCSVRendererGolden(t *testing.T) {
	renderer := NewCSV(CSVParams{Config: testConfig()})
	var buf bytes.Buffer

	stats, err := renderer.Render(context.Background(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows(), truncated: true},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Rows)
	assert.True(t, stats.Truncated)

	want := "Customer,Total Charge,Shipments,Created,Active\n" +
		"\"ACME, Inc.\",\"$1,234.50\",12,2026-07-15 11:00,Yes\n" +
		"\"Quote \"\"Freight\"\"\",($99.10),0,2026-07-15 11:00,No\n" +
		"Ünïcode Cargo 🚚,,,,\n" +
		truncationNotice + ",,,,\n"
	assert.Equal(t, want, buf.String())
}

func TestJSONRendererGolden(t *testing.T) {
	renderer := NewJSON()
	var buf bytes.Buffer

	stats, err := renderer.Render(context.Background(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows()},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Rows)
	assert.False(t, stats.Truncated)

	var doc struct {
		Meta struct {
			Title       string `json:"title"`
			GeneratedAt int64  `json:"generatedAt"`
		} `json:"meta"`
		Schema []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Type  string `json:"type"`
		} `json:"schema"`
		Rows    [][]any `json:"rows"`
		Summary struct {
			RowCount  int64 `json:"rowCount"`
			Truncated bool  `json:"truncated"`
		} `json:"summary"`
	}
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &doc))

	assert.Equal(t, "Revenue by Customer", doc.Meta.Title)
	require.Len(t, doc.Schema, 5)
	assert.Equal(t, "decimal", doc.Schema[1].Type)
	require.Len(t, doc.Rows, 3)
	assert.Equal(t, "1234.5", doc.Rows[0][1])
	assert.Equal(t, float64(12), doc.Rows[0][2])
	assert.Nil(t, doc.Rows[2][1])
	assert.Equal(t, int64(3), doc.Summary.RowCount)
	assert.False(t, doc.Summary.Truncated)
}

func TestXLSXRendererGolden(t *testing.T) {
	renderer := NewXLSX()
	var buf bytes.Buffer

	stats, err := renderer.Render(context.Background(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows()},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Rows)

	file, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	defer file.Close()

	sheets := file.GetSheetList()
	assert.Contains(t, sheets, "Report")
	assert.Contains(t, sheets, "Data")

	header, err := file.GetCellValue("Data", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Customer", header)

	customer, err := file.GetCellValue("Data", "A2")
	require.NoError(t, err)
	assert.Equal(t, "ACME, Inc.", customer)

	chargeType, err := file.GetCellType("Data", "B2")
	require.NoError(t, err)
	assert.NotEqual(t, excelize.CellTypeSharedString, chargeType,
		"decimal cells must be numeric, not strings")
	assert.NotEqual(t, excelize.CellTypeInlineString, chargeType,
		"decimal cells must be numeric, not strings")
	// The cell holds the raw number; the money format lives in the cell style,
	// so Excel can still sum the column.
	rawCharge, err := file.GetCellValue("Data", "B2", excelize.Options{RawCellValue: true})
	require.NoError(t, err)
	assert.Equal(t, "1234.5", rawCharge)

	charge, err := file.GetCellValue("Data", "B2")
	require.NoError(t, err)
	assert.Equal(t, "$1,234.50", charge)

	shipments, err := file.GetCellValue("Data", "C2")
	require.NoError(t, err)
	assert.Equal(t, "12", shipments)

	createdType, err := file.GetCellType("Data", "D2")
	require.NoError(t, err)
	assert.NotEqual(t, excelize.CellTypeSharedString, createdType,
		"epoch columns must be date-formatted numbers, not strings")
	created, err := file.GetCellValue("Data", "D2")
	require.NoError(t, err)
	assert.Equal(t, "2026-07-15 11:00", created)

	title, err := file.GetCellValue("Report", "B1")
	require.NoError(t, err)
	assert.Equal(t, "Revenue by Customer", title)
}

type countingWriter struct{ n int64 }

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

// TestStreamingMemoryBound guards the "never materialize the dataset"
// invariant: streaming 300k rows through the CSV and JSON renderers must not
// grow the heap in proportion to the dataset (~60MB+ if buffered).
func TestStreamingMemoryBound(t *testing.T) {
	const totalRows = 300_000

	newDataset := func() *fakeReader {
		return &fakeReader{
			schema: testSchema(),
			total:  totalRows,
			generate: func(i int64) services.ReportRow {
				return services.ReportRow{
					fmt.Sprintf("Customer %d with a reasonably long name", i),
					decimal.NewFromInt(i),
					i,
					int64(1784131200),
					i%2 == 0,
				}
			},
		}
	}

	renderers := []services.ReportRenderer{
		NewCSV(CSVParams{Config: testConfig()}),
		NewJSON(),
	}

	for _, renderer := range renderers {
		t.Run(string(renderer.Format()), func(t *testing.T) {
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)

			sink := &countingWriter{}
			stats, err := renderer.Render(context.Background(), &services.ReportRenderRequest{
				Dataset: newDataset(),
				Sink:    sink,
				Meta:    testMeta(),
			})
			require.NoError(t, err)
			require.Equal(t, int64(totalRows), stats.Rows)
			require.Greater(t, sink.n, int64(10_000_000),
				"expected tens of MB written through the sink")

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)

			growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
			assert.Less(t, growth, int64(20_000_000),
				"renderer retained %d bytes of heap after streaming %d rows", growth, totalRows)
		})
	}
}

// capturingPDFRenderer records the render request instead of printing it, so the
// report layout can be asserted without a browser in the loop.
type capturingPDFRenderer struct {
	got *services.PDFRenderRequest
	pdf []byte
	err error
}

func (c *capturingPDFRenderer) Render(
	_ context.Context,
	req *services.PDFRenderRequest,
) ([]byte, error) {
	c.got = req
	if c.err != nil {
		return nil, c.err
	}
	if c.pdf == nil {
		return []byte("%PDF-1.7 stub"), nil
	}
	return c.pdf, nil
}

func (c *capturingPDFRenderer) Healthy(_ context.Context) error { return nil }

func (c *capturingPDFRenderer) Enabled() bool { return true }

func TestPDFRendererRowCap(t *testing.T) {
	cfg := &config.Config{}
	cfg.Reporting.PDFMaxRows = 2
	renderer := NewPDF(PDFParams{
		Config:   cfg,
		Renderer: &capturingPDFRenderer{},
		Logger:   zapNop(),
	})

	var buf bytes.Buffer
	_, err := renderer.Render(t.Context(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows()},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CSV or XLSX")
}

func TestPDFRendererSubmitsReportPageSetup(t *testing.T) {
	backend := &capturingPDFRenderer{pdf: []byte("%PDF-1.7 body")}
	renderer := NewPDF(PDFParams{
		Config:   testConfig(),
		Renderer: backend,
		Logger:   zapNop(),
	})

	var buf bytes.Buffer
	stats, err := renderer.Render(t.Context(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows()},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Rows)
	assert.Equal(t, "%PDF-1.7 body", buf.String())

	require.NotNil(t, backend.got)
	// Reports are wide, so landscape Letter with the historical 0.4in box.
	assert.Equal(t, documenttemplate.PageSizeLetter, backend.got.PageSize)
	assert.Equal(t, documenttemplate.OrientationLandscape, backend.got.Orientation)
	assert.Equal(t, reportPDFMargins, backend.got.Margins)
	assert.Equal(t, testMeta().Title, backend.got.Title)

	assert.Contains(t, backend.got.HTML, "<!DOCTYPE html>")
	assert.Contains(t, backend.got.HTML, testMeta().Title)
	for _, column := range testSchema() {
		assert.Contains(t, backend.got.HTML, column.Label)
	}
}

func TestPDFRendererPropagatesBackendFailure(t *testing.T) {
	backend := &capturingPDFRenderer{err: services.ErrPDFRendererUnavailable}
	renderer := NewPDF(PDFParams{
		Config:   testConfig(),
		Renderer: backend,
		Logger:   zapNop(),
	})

	var buf bytes.Buffer
	_, err := renderer.Render(t.Context(), &services.ReportRenderRequest{
		Dataset: &fakeReader{schema: testSchema(), rows: testRows()},
		Sink:    &buf,
		Meta:    testMeta(),
	})
	require.ErrorIs(t, err, services.ErrPDFRendererUnavailable)
	assert.Empty(t, buf.Bytes(), "a failed render must not write a partial artifact")
}

func TestRegistryResolvesRenderers(t *testing.T) {
	registry := NewRegistry(RegistryParams{
		Renderers: []services.ReportRenderer{
			NewCSV(CSVParams{Config: testConfig()}),
			NewJSON(),
		},
	})

	csvRenderer, err := registry.For(report.FormatCSV)
	require.NoError(t, err)
	assert.Equal(t, report.FormatCSV, csvRenderer.Format())

	_, err = registry.For(report.FormatPDF)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no renderer registered")
}

func TestFormatCellEdgeCases(t *testing.T) {
	loc := metaLocation(&services.ReportRunMeta{Timezone: "America/Chicago"})

	epochCol := testColumn("c", "Created", reportcatalog.FieldEpoch, reportcatalog.FormatNone)
	assert.Equal(t, "2026-07-16 00:00", formatCell(&epochCol, int64(1784178000), loc))

	intCol := testColumn("c", "Count", reportcatalog.FieldInt, reportcatalog.FormatCount)
	assert.Equal(t, "42", formatCell(&intCol, int64(42), loc))
	assert.Equal(t, "", formatCell(&intCol, nil, loc))

	strCol := testColumn("c", "Name", reportcatalog.FieldString, reportcatalog.FormatNone)
	assert.Equal(t, strings.Repeat("x", 3), formatCell(&strCol, "xxx", loc))

	// A column with no resolved display renders raw rather than guessing.
	bare := &services.ReportResultColumn{Type: reportcatalog.FieldDecimal}
	assert.Equal(t, "1234.5", formatCell(bare, decimal.RequireFromString("1234.5"), loc))
}

func zapNop() *zap.Logger { return zap.NewNop() }
