package pdfreader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupLinesJoinsRunsOnOneLineAndSplitsColumns(t *testing.T) {
	t.Parallel()

	lines := groupLines([]textRun{
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
	assert.Nil(t, groupLines(nil))
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
