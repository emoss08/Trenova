package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Nobody places a tile.

Asking a model for x and y coordinates produces overlapping tiles and gaps,
and the person who described the dashboard then has to go and fix a layout
they never chose. So the caller says what goes on the page and the layout is
computed: left to right, wrapping at the grid's width, each kind at the width
its content wants.
*/

func tile(kind report.TileKind, extra map[string]any) map[string]any {
	out := map[string]any{"kind": string(kind), "definitionId": "rdef_01JABCDEFGHJKMNPQRSTVWXYZ"}
	for key, value := range extra {
		out[key] = value
	}

	return out
}

func TestBuildTiles_LaysThemOutLeftToRight(t *testing.T) {
	t.Parallel()

	tiles, err := buildTiles([]map[string]any{
		tile(report.TileKindKPI, map[string]any{"columnId": "revenue"}),
		tile(report.TileKindKPI, map[string]any{"columnId": "margin"}),
	}, 0)

	require.NoError(t, err)
	require.Len(t, tiles, 2)
	assert.Equal(t, 0, tiles[0].X)
	assert.Equal(t, 3, tiles[1].X, "the second KPI sits beside the first")
	assert.Equal(t, 0, tiles[1].Y)
}

// A row that cannot take another tile wraps, rather than running off the grid
// where nothing renders it.
func TestBuildTiles_WrapsAtTheGridWidth(t *testing.T) {
	t.Parallel()

	tiles, err := buildTiles([]map[string]any{
		tile(report.TileKindChart, nil),
		tile(report.TileKindChart, nil),
		tile(report.TileKindChart, nil),
	}, 0)

	require.NoError(t, err)
	require.Len(t, tiles, 3)
	assert.Equal(t, 0, tiles[0].X)
	assert.Equal(t, 6, tiles[1].X)
	// Two six-wide charts fill twelve columns, so the third starts a new row
	// below the tallest tile on the one above.
	assert.Equal(t, 0, tiles[2].X)
	assert.Equal(t, 4, tiles[2].Y)
}

/*
A KPI is one number and a table is a grid of them.

Giving both the same width produces a page that reads as a mistake, so each
kind gets the width its content wants unless the caller says otherwise.
*/
func TestBuildTiles_SizesEachKindForItsContent(t *testing.T) {
	t.Parallel()

	tiles, err := buildTiles([]map[string]any{
		tile(report.TileKindKPI, map[string]any{"columnId": "revenue"}),
	}, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, tiles[0].W)

	tiles, err = buildTiles([]map[string]any{tile(report.TileKindTable, nil)}, 0)
	require.NoError(t, err)
	assert.Equal(t, 12, tiles[0].W)
}

func TestBuildTiles_HonoursAWidthTheCallerAsked(t *testing.T) {
	t.Parallel()

	tiles, err := buildTiles([]map[string]any{
		tile(report.TileKindKPI, map[string]any{"columnId": "revenue", "width": int64(6)}),
	}, 0)

	require.NoError(t, err)
	assert.Equal(t, 6, tiles[0].W)
}

// A tile wider than the grid would be cut off by the page, so it is clamped
// rather than stored as something that cannot be drawn.
func TestBuildTiles_ClampsAWidthWiderThanTheGrid(t *testing.T) {
	t.Parallel()

	tiles, err := buildTiles([]map[string]any{
		tile(report.TileKindTable, map[string]any{"width": int64(40)}),
	}, 0)

	require.NoError(t, err)
	assert.Equal(t, report.MaxTileSpan, tiles[0].W)
}

// Adding one tile to a dashboard puts it below what is there, not on top of
// it — which is what the starting row is for.
func TestBuildTiles_StartsBelowWhatIsAlreadyThere(t *testing.T) {
	t.Parallel()

	existing := []report.DashboardTile{{X: 0, Y: 0, W: 12, H: 4}}
	tiles, err := buildTiles([]map[string]any{tile(report.TileKindTable, nil)}, nextRow(existing))

	require.NoError(t, err)
	assert.Equal(t, 4, tiles[0].Y)
}

func TestBuildTiles_RefusesSomethingThatIsNotAReportID(t *testing.T) {
	t.Parallel()

	_, err := buildTiles([]map[string]any{
		{"kind": string(report.TileKindTable), "definitionId": "the revenue one"},
	}, 0)

	require.Error(t, err)
}

/*
A tile that cannot draw anything renders as an empty box, and the person who
asked for the dashboard cannot tell an empty box from one that failed to load.
Both are refused on the card instead.
*/
func TestCreateDashboard_RefusesATileThatCannotDrawAnything(t *testing.T) {
	t.Parallel()

	tool := &createDashboardTool{}

	err := tool.validateArgs(map[string]any{
		"name":  "Morning numbers",
		"tiles": []any{map[string]any{"kind": "table"}},
	})
	require.Error(t, err)

	err = tool.validateArgs(map[string]any{
		"name":  "Morning numbers",
		"tiles": []any{map[string]any{"kind": "text"}},
	})
	require.Error(t, err)

	// A KPI with a report but no column has a number to show and no idea
	// which one.
	err = tool.validateArgs(map[string]any{
		"name": "Morning numbers",
		"tiles": []any{
			map[string]any{"kind": "kpi", "definitionId": "rdef_1"},
		},
	})
	require.Error(t, err)
}

func TestCreateDashboard_RefusesADashboardWithNothingOnIt(t *testing.T) {
	t.Parallel()

	tool := &createDashboardTool{}

	require.Error(t, tool.validateArgs(map[string]any{"name": "Empty"}))
	require.Error(t, tool.validateArgs(map[string]any{
		"tiles": []any{map[string]any{"kind": "text", "text": "hello"}},
	}))
}

func TestCreateDashboard_AcceptsAPageItCanDraw(t *testing.T) {
	t.Parallel()

	tool := &createDashboardTool{}

	err := tool.validateArgs(map[string]any{
		"name": "Morning numbers",
		"tiles": []any{
			map[string]any{"kind": "kpi", "definitionId": "rdef_1", "columnId": "revenue"},
			map[string]any{"kind": "text", "text": "Figures are through yesterday."},
		},
	})

	require.NoError(t, err)
}
