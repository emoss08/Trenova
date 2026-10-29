package boardparse_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ids(prefix string, n int) []any {
	out := make([]any, 0, n)
	for range n {
		out = append(out, pulid.MustNew(prefix).String())
	}

	return out
}

func lateBoard() map[string]any {
	shipments := ids("shp_", 6)

	return map[string]any{
		"title":   "Six late shipments on the Dallas lanes",
		"verdict": "Four of the six are held behind [one driver's hours][3], and two are [late at the dock][2].",
		"blocks": []any{
			map[string]any{
				"type":  "figures",
				"title": "Where they stand",
				"figures": []any{
					map[string]any{
						"label":   "Late now",
						"records": shipments,
						"measure": "lateness",
					},
					map[string]any{
						"label":     "Furthest behind",
						"records":   shipments,
						"measure":   "lateness",
						"aggregate": "max",
					},
					map[string]any{"label": "Active shipments", "metric": "activeShipments"},
				},
			},
			map[string]any{
				"type":    "records",
				"title":   "The six, furthest behind first",
				"records": shipments,
				"measure": "lateness",
			},
			map[string]any{"type": "timeline", "record": shipments[0], "half": true},
			map[string]any{
				"type":  "note",
				"tone":  "info",
				"title": "Re-tender the two furthest behind",
				"text":  "A team keeps the appointments.",
				"half":  true,
			},
		},
	}
}

func TestParseReadsABoardOfEveryKindOfSource(t *testing.T) {
	t.Parallel()

	parsed, problems := boardparse.Parse(lateBoard())

	require.Empty(t, problems)
	assert.Equal(t, "Six late shipments on the Dallas lanes", parsed.Title)
	require.Len(t, parsed.Blocks, 4)

	figures := parsed.Blocks[0]
	assert.Equal(t, board.BlockFigures, figures.Type)
	require.Len(t, figures.Figures, 3)
	assert.Equal(t, board.AggregateCount, figures.Figures[0].Aggregate)
	assert.Equal(t, board.AggregateMax, figures.Figures[1].Aggregate)
	assert.Equal(t, "activeShipments", figures.Figures[2].Metric)
	assert.Equal(t, permission.ResourceShipment, figures.Figures[0].Records.Resource)

	records := parsed.Blocks[1]
	assert.Equal(t, board.MeasureLateness, records.Measure)
	assert.Equal(t, board.OrderMeasure, records.Order)
	assert.Len(t, records.Records.IDs, 6)

	assert.Equal(t, board.DefaultTimeline, parsed.Blocks[2].Limit)
	assert.True(t, parsed.Blocks[2].Half)
	assert.Equal(t, board.ToneInfo, parsed.Blocks[3].Tone)
	assert.Equal(t, []int{1, 2, 3, 0}, parsed.Numerals())
	assert.Len(t, parsed.Records()[permission.ResourceShipment], 6)
	assert.Equal(t, []string{"activeShipments"}, parsed.Metrics())
}

func TestParseRefusesWhatABoardCannotDraw(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(map[string]any)
		want   string
	}{
		"no title": {
			mutate: func(args map[string]any) { delete(args, "title") },
			want:   "title is required",
		},
		"unknown block type": {
			mutate: func(args map[string]any) {
				args["blocks"] = []any{map[string]any{"type": "heading"}}
			},
			want: "blocks[1].type must be one of",
		},
		"a field another block takes": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				blocks[1].(map[string]any)["chart"] = 1.0
			},
			want: `blocks[2] is a records block, which takes no "chart"`,
		},
		"a measure of another record": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				blocks[1].(map[string]any)["measure"] = "amountDue"
			},
			want: "blocks[2].measure for a shipment is one of lateness, revenue",
		},
		"mixed records": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				blocks[1].(map[string]any)["records"] = append(ids("shp_", 1), ids("inv_", 1)...)
			},
			want: "mixes shipment and invoice ids",
		},
		"a record a board cannot read": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				blocks[1].(map[string]any)["records"] = ids("doc_", 1)
			},
			want: "which is not a shipment",
		},
		"a citation to a note": {
			mutate: func(args map[string]any) { args["verdict"] = "See [the note][4]." },
			want:   "verdict cites block 4, which is words, not data",
		},
		"a citation past the board": {
			mutate: func(args map[string]any) { args["verdict"] = "See [this][9]." },
			want:   "verdict cites block 9, but the board has 4",
		},
		"words alone": {
			mutate: func(args map[string]any) {
				args["verdict"] = "Nothing to cite."
				args["blocks"] = []any{map[string]any{"type": "prose", "text": "Only words."}}
			},
			want: "at least one block that reads data",
		},
		"too many blocks": {
			mutate: func(args map[string]any) {
				blocks := make([]any, 0, board.MaxBlocks+1)
				for range board.MaxBlocks + 1 {
					blocks = append(blocks, map[string]any{"type": "prose", "text": "x"})
				}
				args["blocks"] = blocks
			},
			want: "at most 8 blocks",
		},
		"too many notes": {
			mutate: func(args map[string]any) {
				note := map[string]any{"type": "note", "tone": "ok", "title": "t", "text": "x"}
				args["blocks"] = append(args["blocks"].([]any), note, note)
			},
			want: "at most 2 notes",
		},
		"an unknown metric": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				figures := blocks[0].(map[string]any)["figures"].([]any)
				figures[2].(map[string]any)["metric"] = "profitToday"
			},
			want: `metric "profitToday" is not one of`,
		},
		"a figure with two sources": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				figures := blocks[0].(map[string]any)["figures"].([]any)
				figures[2].(map[string]any)["records"] = ids("shp_", 1)
			},
			want: "needs exactly one source",
		},
		"a sum without a measure": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				figures := blocks[0].(map[string]any)["figures"].([]any)
				delete(figures[1].(map[string]any), "measure")
			},
			want: "aggregate max needs a measure",
		},
		"a meter of a block that is not checks": {
			mutate: func(args map[string]any) {
				args["blocks"] = append(args["blocks"].([]any),
					map[string]any{"type": "meter", "checksBlock": 2.0})
			},
			want: "names block 2, which is not a checks block",
		},
		"a meter given both a checks block and a metric": {
			mutate: func(args map[string]any) {
				args["blocks"] = append(args["blocks"].([]any),
					map[string]any{"type": "meter", "metric": "onTimePercent", "checksBlock": 1.0})
			},
			want: "keep metric for a percentage such as onTimePercent and leave out checksBlock",
		},
		"a meter of a count": {
			mutate: func(args map[string]any) {
				args["blocks"] = append(args["blocks"].([]any),
					map[string]any{"type": "meter", "metric": "activeShipments"})
			},
			want: "must be a percentage to draw as a meter",
		},
		"a timeline without its record": {
			mutate: func(args map[string]any) {
				blocks := args["blocks"].([]any)
				delete(blocks[2].(map[string]any), "record")
			},
			want: "blocks[3].record is required",
		},
		"an analysis that is not an artifact": {
			mutate: func(args map[string]any) {
				args["blocks"] = append(args["blocks"].([]any),
					map[string]any{"type": "chart", "analysis": "shp_123", "chart": 1.0})
			},
			want: "blocks[5].analysis is the artifactId a tool result gave you",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := lateBoard()
			tc.mutate(args)
			_, problems := boardparse.Parse(args)

			assert.True(t, strings.Contains(strings.Join(problems, "\n"), tc.want),
				"want %q in %v", tc.want, problems)
		})
	}
}

func TestParseCountsEachRecordOnceAcrossTheBoard(t *testing.T) {
	t.Parallel()

	shared := ids("inv_", board.MaxListRecords)
	blocks := []any{map[string]any{"type": "records", "records": shared}}
	for range 4 {
		blocks = append(blocks, map[string]any{"type": "records", "records": shared})
	}
	args := map[string]any{"title": "Invoices", "verdict": "Open.", "blocks": blocks}
	_, problems := boardparse.Parse(args)
	require.Empty(t, problems)

	for idx := 1; idx < len(blocks); idx++ {
		blocks[idx] = map[string]any{
			"type": "records", "records": ids("inv_", board.MaxListRecords),
		}
	}
	_, problems = boardparse.Parse(args)

	assert.Contains(t, strings.Join(problems, "\n"), "the board names 125 records")
}

func TestSegmentsSplitsCitationsOutOfText(t *testing.T) {
	t.Parallel()

	segments := board.Segments(
		"Four are held behind [one driver's hours][3], two [at the dock][2].",
	)

	assert.Equal(t, []board.Segment{
		{Text: "Four are held behind "},
		{Text: "one driver's hours", Cite: 3},
		{Text: ", two "},
		{Text: "at the dock", Cite: 2},
		{Text: "."},
	}, segments)
	assert.Equal(t, "Four are held behind one driver's hours, two at the dock.",
		board.PlainText("Four are held behind [one driver's hours][3], two [at the dock][2]."))
}

func TestParseWritesTheWordsPlainAndHoldsANoteToItsOwnLimits(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["verdict"] = "**Four of the six** are held behind __[one driver's hours][3]__."
	blocks := args["blocks"].([]any)
	note := blocks[3].(map[string]any)
	note["title"] = strings.Repeat("a", board.MaxNoteTitleRunes)
	note["text"] = "Call `S2610-041` first."

	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Equal(t, "Four of the six are held behind [one driver's hours][3].", parsed.Verdict)
	assert.Equal(t, strings.Repeat("a", board.MaxNoteTitleRunes), parsed.Blocks[3].Title)
	assert.Equal(t, "Call S2610-041 first.", parsed.Blocks[3].Text)

	note["title"] = strings.Repeat("a", board.MaxNoteTitleRunes+1)
	_, problems = boardparse.Parse(args)
	assert.Equal(t, []string{"blocks[4].title is longer than 80 characters"}, problems)

	blocks[0].(map[string]any)["title"] = strings.Repeat("a", board.MaxBlockTitleRunes+1)
	note["title"] = "Re-tender the two furthest behind"
	_, problems = boardparse.Parse(args)
	assert.Equal(t, []string{"blocks[1].title is longer than 60 characters"}, problems)
}

/*
Every operations board asked for five figures, and show_board refused each one
for holding more than four, so every board cost a retry. A figures block holds
up to eight, which the card draws in rows of at most four; nine is refused.
*/
func TestParseTakesUpToEightFiguresInABlock(t *testing.T) {
	t.Parallel()

	figuresBoard := func(n int) map[string]any {
		metrics := []string{
			"activeShipments", "onTimePercent", "atRisk", "unassigned", "revenueToday",
		}
		figures := make([]any, 0, n)
		for idx := range n {
			figures = append(figures, map[string]any{
				"label":  "Figure " + string(rune('A'+idx)),
				"metric": metrics[idx%len(metrics)],
			})
		}

		return map[string]any{
			"title":   "Operations board",
			"verdict": "Today's operations figures.",
			"blocks": []any{
				map[string]any{"type": "figures", "title": "Today", "figures": figures},
			},
		}
	}

	parsed, problems := boardparse.Parse(figuresBoard(5))
	require.Empty(t, problems)
	assert.Len(t, parsed.Blocks[0].Figures, 5)

	_, problems = boardparse.Parse(figuresBoard(board.MaxFigures))
	assert.Empty(t, problems)

	_, problems = boardparse.Parse(figuresBoard(board.MaxFigures + 1))
	assert.Contains(t, strings.Join(problems, "\n"), "blocks[1].figures needs 1 to 8 figures")
}
