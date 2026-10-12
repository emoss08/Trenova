package pdfreader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrangeJoinsRunsOnOneLineAndSplitsColumns(t *testing.T) {
	t.Parallel()

	lines, text := arrange([]textRun{
		{text: "Total", left: 300, top: 101, right: 330, bottom: 111},
		{text: "Line", left: 50, top: 100, right: 75, bottom: 110},
		{text: "Haul:", left: 78, top: 100.5, right: 105, bottom: 110.5},
		{text: "$1,200.00", left: 50, top: 120, right: 100, bottom: 130},
	})

	require.Len(t, lines, 3)
	assert.Equal(t, "Line Haul:", lines[0].Text)
	assert.InDelta(t, 50, lines[0].Left, 0.001)
	assert.InDelta(t, 55, lines[0].Width, 0.001)
	assert.InDelta(t, 100, lines[0].Top, 0.001)
	assert.InDelta(t, 10.5, lines[0].Height, 0.001)
	assert.Equal(t, "Total", lines[1].Text)
	assert.Equal(t, "$1,200.00", lines[2].Text)
	assert.Equal(t, "Line Haul:   Total\n$1,200.00", text)

	none, empty := arrange(nil)
	assert.Nil(t, none)
	assert.Empty(t, empty)
}

func TestArrangeReadsARateConfirmationRowByRow(t *testing.T) {
	t.Parallel()

	_, text := arrange([]textRun{
		{text: "Customer Requirements", left: 300, top: 100, right: 420, bottom: 110},
		{text: "08/02/22", left: 560, top: 140.2, right: 600, bottom: 149.8},
		{text: "SHIPPER#1:", left: 40, top: 140, right: 95, bottom: 150},
		{text: "Pick Up Date:", left: 450, top: 140, right: 520, bottom: 150},
		{text: "BEX Logistics-OH Consolidation", left: 190, top: 140.1, right: 360, bottom: 150},
		{text: "Address:", left: 40, top: 152, right: 85, bottom: 162},
		{text: "3685 Alum Creek Dr", left: 190, top: 152, right: 290, bottom: 162},
	})

	assert.Equal(t,
		"Customer Requirements\n\n"+
			"SHIPPER#1:   BEX Logistics-OH Consolidation   Pick Up Date:   08/02/22\n"+
			"Address:   3685 Alum Creek Dr",
		text,
	)
}

func TestBoundedDPIKeepsRendersUnderThePixelBudget(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 150, boundedDPI(612, 792, 150, 25_000_000))
	assert.Equal(t, 150, boundedDPI(612, 792, 150, 0))

	capped := boundedDPI(14400, 14400, 150, 25_000_000)
	side := 14400.0 * float64(capped) / pointsPerInch
	assert.LessOrEqual(t, side*side, 25_000_000.0)
	assert.Greater(t, capped, 0)
	assert.Equal(t, 1, boundedDPI(1e6, 1e6, 150, 1))
}

func TestFlattenOnWhite(t *testing.T) {
	t.Parallel()

	pix := []uint8{0, 0, 0, 0, 255, 0, 0, 255, 0, 0, 0, 128}
	flattenOnWhite(pix)

	assert.Equal(t, []uint8{255, 255, 255, 255, 255, 0, 0, 255, 127, 127, 127, 255}, pix)
}

func TestWordsOfJoinsGlyphsByTheirSpacing(t *testing.T) {
	t.Parallel()

	glyph := func(text string, left, width float64) textRun {
		return textRun{text: text, left: left, top: 100, right: left + width, bottom: 110, em: 10}
	}
	comma := textRun{text: ",", left: 129, top: 107, right: 131, bottom: 112, em: 10}

	words := wordsOf([]textRun{
		glyph("T", 100, 6), glyph("o", 105.5, 5), glyph("t", 111, 3), glyph("a", 114.5, 5),
		glyph("l", 120, 2), glyph("s", 122.5, 5), comma,
		glyph(" ", 131, 3),
		glyph("$", 134, 5), glyph("9", 139.5, 5),
		glyph("A", 400, 6),
	})

	require.Len(t, words, 3)
	assert.Equal(t, "Totals,", words[0].text)
	assert.InDelta(t, 100, words[0].left, 0.001)
	assert.InDelta(t, 131, words[0].right, 0.001)
	assert.Equal(t, "$9", words[1].text)
	assert.Equal(t, "A", words[2].text)
	assert.Empty(t, wordsOf(nil))
}
