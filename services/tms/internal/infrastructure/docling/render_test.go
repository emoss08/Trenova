package docling

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var goldenFixtures = []string{
	"rate-confirmation.pdf",
	"faxed-bol.pdf",
	"photo-pod.jpg",
	"rate-confirmation.docx",
	"tendered-loads.xlsx",
	"load-tender.eml",
}

func renderFixture(t *testing.T, name string) *kernelports.ConvertedDocument {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name+".response.json"))
	require.NoError(t, err)

	converted, err := convertResult(raw)
	require.NoError(t, err)

	return converted
}

func TestRenderGoldens(t *testing.T) {
	t.Parallel()

	for _, name := range goldenFixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			converted := renderFixture(t, name)
			encoded, err := sonic.ConfigStd.MarshalIndent(converted.Pages, "", "  ")
			require.NoError(t, err)
			encoded = append(encoded, '\n')

			golden := filepath.Join("testdata", name+".golden.json")
			if os.Getenv("TRENOVA_UPDATE_FIXTURES") == "1" {
				require.NoError(t, os.WriteFile(golden, encoded, 0o644))
			}

			want, err := os.ReadFile(golden)
			require.NoError(
				t,
				err,
				"regenerate with TRENOVA_UPDATE_FIXTURES=1 go test ./internal/infrastructure/docling/",
			)
			assert.JSONEq(t, string(want), string(encoded))
			assert.Equal(t, ConverterName, converted.Converter)
			assert.NotEmpty(t, converted.Raw)
		})
	}
}

func TestRenderNativeRateConfirmation(t *testing.T) {
	t.Parallel()

	converted := renderFixture(t, "rate-confirmation.pdf")
	require.Len(t, converted.Pages, 2)

	first := converted.Pages[0]
	assert.Equal(t, 1, first.Number)
	assert.True(t, first.Native)
	assert.False(t, first.OCR)
	assert.InDelta(t, 612.0, first.Width, 0.001)
	assert.InDelta(t, 792.0, first.Height, 0.001)

	assert.True(t, strings.HasPrefix(first.Markdown, PageHeaderMarker+"\n"))
	assert.Contains(t, first.Markdown, "Load # RC-48213")
	assert.Contains(t, first.Markdown, "## Carrier Rate Confirmation")
	assert.Contains(t, first.Markdown, "| # | Type | Facility | Address | Date | Window | Ref |")
	assert.Contains(t, first.Markdown, "| --- | --- |")
	assert.NotContains(t, first.Text, "| --- |")
	assert.Contains(t, first.Text, "Load # RC-48213")

	header := strings.Index(first.Markdown, "Load # RC-48213")
	title := strings.Index(first.Markdown, "Carrier Rate Confirmation")
	assert.Less(t, header, title, "the printed header is rendered before the body")

	pipe := strings.Index(first.Markdown, `53' Dry Van \| Weight`)
	assert.Negative(t, pipe, "paragraph text is not escaped, only table cells are")
}

func TestRenderLayoutLinesAreTopLeftFractions(t *testing.T) {
	t.Parallel()

	converted := renderFixture(t, "rate-confirmation.pdf")
	first := converted.Pages[0]
	require.NotEmpty(t, first.Lines)
	assert.LessOrEqual(t, len(first.Lines), maxLinesPerPage)

	var header, cell *kernelports.ConvertedLine
	for i := range first.Lines {
		line := &first.Lines[i]
		assert.GreaterOrEqual(t, line.X, 0.0)
		assert.GreaterOrEqual(t, line.Y, 0.0)
		assert.LessOrEqual(t, line.X+line.W, 1.0001)
		assert.LessOrEqual(t, line.Y+line.H, 1.0001)
		switch line.Text {
		case "Load # RC-48213":
			header = line
		case "Harvest Foods DC":
			cell = line
		}
	}

	require.NotNil(t, header, "the page header is placed")
	assert.Less(t, header.Y, 0.05, "a header near the top of the page has a small top-left y")
	assert.Greater(t, header.X, 0.7, "the load number sits at the right of the header")

	require.NotNil(t, cell, "table cells carry their own boxes")
	assert.Greater(t, cell.Y, 0.2)
	assert.Less(t, cell.Y, 0.35)

	assert.Equal(t, "#", first.Lines[0].Text, "table cells come before text items")
}

func TestRenderScannedDocumentsAreOCR(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"faxed-bol.pdf", "photo-pod.jpg"} {
		converted := renderFixture(t, name)
		require.Len(t, converted.Pages, 1, name)
		page := converted.Pages[0]
		assert.True(t, page.OCR, name)
		assert.False(t, page.Native, name)
		assert.Greater(t, page.OCRConfidence, 0.5, name)
		assert.NotEmpty(t, page.Lines, name)
	}

	bol := renderFixture(t, "faxed-bol.pdf").Pages[0]
	assert.Contains(t, bol.Markdown, "| Qty | Type | Description | Weight | Class |")
	assert.Contains(t, bol.Text, "HF-2026-10433")
}

func TestRenderOfficeAndEmail(t *testing.T) {
	t.Parallel()

	docx := renderFixture(t, "rate-confirmation.docx")
	require.Len(t, docx.Pages, 1)
	assert.True(t, docx.Pages[0].Native)
	assert.False(t, docx.Pages[0].OCR)
	assert.Empty(t, docx.Pages[0].Lines, "a DOCX has no page geometry")
	assert.True(t, strings.HasPrefix(docx.Pages[0].Markdown, PageHeaderMarker))
	assert.Contains(t, docx.Pages[0].Markdown, "- Line haul: $1,240.00\n- Tarp fee: $75.00")
	assert.Contains(t, docx.Pages[0].Markdown, "call | 555-0199 on arrival.")

	xlsx := renderFixture(t, "tendered-loads.xlsx")
	require.Len(t, xlsx.Pages, 2, "one page per sheet")
	assert.Contains(t, xlsx.Pages[0].Markdown, "| Load | Origin | Destination |")
	assert.Contains(t, xlsx.Pages[1].Markdown, "| TONU | Truck ordered not used |")

	eml := renderFixture(t, "load-tender.eml")
	require.Len(t, eml.Pages, 1)
	assert.Contains(t, eml.Pages[0].Markdown, "# Load tender RC-48213")
	assert.Contains(t, eml.Pages[0].Text, "Rate: $2,062.75 all in")
}

func TestBuildGridRepeatsSpannedCells(t *testing.T) {
	t.Parallel()

	grid := buildGrid(&tableData{
		NumRows: 2,
		NumCols: 3,
		TableCells: []tableCell{
			{StartRow: 0, EndRow: 1, StartCol: 0, EndCol: 2, Text: "Stop"},
			{StartRow: 0, EndRow: 1, StartCol: 2, EndCol: 3, Text: "Date"},
			{StartRow: 1, EndRow: 2, StartCol: 0, EndCol: 1, Text: "1"},
			{StartRow: 1, EndRow: 2, StartCol: 1, EndCol: 2, Text: "Pick | up"},
			{StartRow: 1, EndRow: 2, StartCol: 2, EndCol: 3, Text: "10/21"},
		},
	})

	assert.Equal(t, [][]string{{"Stop", "Stop", "Date"}, {"1", "Pick | up", "10/21"}}, grid)
	assert.Equal(
		t,
		"| Stop | Stop | Date |\n| --- | --- | --- |\n| 1 | Pick \\| up | 10/21 |",
		pipeTable(grid),
	)
	assert.Equal(t, "Stop   Date\n1   Pick | up   10/21", plainTable(grid))
}

func TestBuildGridDropsEmptyTables(t *testing.T) {
	t.Parallel()

	assert.Nil(t, buildGrid(&tableData{NumRows: 2, NumCols: 2}))
	assert.Nil(t, buildGrid(&tableData{}))
}

func TestToLineConvertsBothOrigins(t *testing.T) {
	t.Parallel()

	size := pageSize{Width: 600, Height: 800}

	bottomLeft, ok := toLine(
		"Total",
		&bbox{L: 60, T: 780, R: 120, B: 760, CoordOrigin: "BOTTOMLEFT"},
		size,
	)
	require.True(t, ok)
	assert.Equal(
		t,
		kernelports.ConvertedLine{Text: "Total", X: 0.1, Y: 0.025, W: 0.1, H: 0.025},
		bottomLeft,
	)

	topLeft, ok := toLine("Total", &bbox{L: 60, T: 20, R: 120, B: 40, CoordOrigin: "TOPLEFT"}, size)
	require.True(t, ok)
	assert.Equal(t, bottomLeft, topLeft)

	_, ok = toLine("Total", &bbox{L: 60, T: 20, R: 120, B: 40}, pageSize{})
	assert.False(t, ok, "a page without a size cannot place a line")

	_, ok = toLine("   ", &bbox{L: 60, T: 20, R: 120, B: 40}, size)
	assert.False(t, ok)
}

func TestLinesCapPrefersCellsAndShortItems(t *testing.T) {
	t.Parallel()

	builder := &pageBuilder{}
	for range maxLinesPerPage {
		builder.items = append(
			builder.items,
			kernelports.ConvertedLine{Text: "a long paragraph line"},
		)
	}
	builder.items = append(builder.items, kernelports.ConvertedLine{Text: "RC-1"})
	builder.cells = []kernelports.ConvertedLine{{Text: "$1,850.00"}}

	lines := builder.lines()
	require.Len(t, lines, maxLinesPerPage)
	assert.Equal(t, "$1,850.00", lines[0].Text)
	assert.Equal(t, "RC-1", lines[1].Text)
}

func TestSourceFlags(t *testing.T) {
	t.Parallel()

	score := 0.9
	cases := []struct {
		name       string
		profile    sourceProfile
		confidence *confidenceScores
		ocr        bool
		native     bool
	}{
		{"native pdf", sourcePDF, &confidenceScores{ParseScore: &score}, false, true},
		{"scanned pdf", sourcePDF, &confidenceScores{OCRScore: &score}, true, false},
		{
			"mixed pdf",
			sourcePDF,
			&confidenceScores{ParseScore: &score, OCRScore: &score},
			true,
			true,
		},
		{"pdf without scores", sourcePDF, nil, false, true},
		{"photo", sourceImage, &confidenceScores{OCRScore: &score}, true, false},
		{"office", sourceOther, nil, false, true},
	}
	for _, tc := range cases {
		ocr, native, _ := sourceFlags(tc.profile, tc.confidence)
		assert.Equal(t, tc.ocr, ocr, tc.name)
		assert.Equal(t, tc.native, native, tc.name)
	}
}
