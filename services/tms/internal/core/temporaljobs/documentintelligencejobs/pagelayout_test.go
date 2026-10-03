package documentintelligencejobs

import (
	"testing"

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

// MuPDF places each line of a native page by its top and left in points; the
// width is estimated from the line's length and its font size.
func TestFitzLayoutReadsLinePositions(t *testing.T) {
	t.Parallel()

	page := `<div id="page0" style="width:600pt;height:800pt">` +
		`<p style="top:80pt;left:60pt;line-height:12pt">` +
		`<span style="font-family:Helvetica;font-size:10pt">Rate: $1,359.56</span></p>` +
		`<p style="top:100pt;left:60pt;line-height:12pt"><span style="font-size:10pt">PO &amp; 77812</span></p>` +
		`</div>`

	lines := fitzLayout(page)

	require.Len(t, lines, 2)
	assert.Equal(t, "Rate: $1,359.56", lines[0].Text)
	assert.InDelta(t, 0.1, lines[0].X, 0.0001)
	assert.InDelta(t, 0.1, lines[0].Y, 0.0001)
	assert.InDelta(t, 15*10*0.5/600, lines[0].W, 0.0001)
	assert.InDelta(t, 0.015, lines[0].H, 0.0001)
	assert.Equal(t, "PO & 77812", lines[1].Text)
	assert.Nil(t, fitzLayout("<p>no page size</p>"))
}
