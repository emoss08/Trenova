package documentintelligencejobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The words tesseract recognised on one line are one box: from the first
// word's left edge to the last word's right, as a fraction of the image.
func TestTesseractLayoutGroupsWordsIntoLines(t *testing.T) {
	t.Parallel()

	output := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"5\t1\t1\t1\t1\t1\t100\t200\t80\t20\t96\tLine\n" +
		"5\t1\t1\t1\t1\t2\t190\t198\t110\t24\t95\tHaul:\n" +
		"5\t1\t1\t1\t2\t1\t100\t260\t60\t20\t91\t$1,200.00\n" +
		"4\t1\t1\t1\t2\t0\t100\t260\t60\t20\t-1\t\n"

	lines := tesseractLayout(output, 1000, 2000)

	require.Len(t, lines, 2)
	assert.Equal(t, "Line Haul:", lines[0].Text)
	assert.InDelta(t, 0.1, lines[0].X, 0.0001)
	assert.InDelta(t, 0.099, lines[0].Y, 0.0001)
	assert.InDelta(t, 0.2, lines[0].W, 0.0001)
	assert.InDelta(t, 0.012, lines[0].H, 0.0001)
	assert.Equal(t, "$1,200.00", lines[1].Text)
}

// A native PDF page gives each line's box in points from the top left; the
// layout keeps it as a fraction of the page.
func TestPDFLayoutPlacesLinesOnThePage(t *testing.T) {
	t.Parallel()

	lines := pdfLayout(&services.PDFPageLayout{
		Width:  600,
		Height: 800,
		Lines: []services.PDFTextLine{
			{Text: "Rate:  $1,359.56", Left: 60, Top: 80, Width: 75, Height: 12},
			{Text: "   ", Left: 60, Top: 90, Width: 10, Height: 12},
			{Text: "PO & 77812", Left: 590, Top: 795, Width: 50, Height: 12},
		},
	})

	require.Len(t, lines, 2)
	assert.Equal(t, "Rate: $1,359.56", lines[0].Text)
	assert.InDelta(t, 0.1, lines[0].X, 0.0001)
	assert.InDelta(t, 0.1, lines[0].Y, 0.0001)
	assert.InDelta(t, 0.125, lines[0].W, 0.0001)
	assert.InDelta(t, 0.015, lines[0].H, 0.0001)
	assert.Equal(t, "PO & 77812", lines[1].Text)
	assert.LessOrEqual(t, lines[1].X+lines[1].W, 1.0)
	assert.LessOrEqual(t, lines[1].Y+lines[1].H, 1.0)
	assert.Nil(t, pdfLayout(&services.PDFPageLayout{}))
	assert.Nil(t, pdfLayout(nil))
}
