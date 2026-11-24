package boardparse_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readingBoard() map[string]any {
	args := lateBoard()
	shipments := args["blocks"].([]any)[1].(map[string]any)["records"]
	args["readings"] = []any{
		map[string]any{
			"key": "late", "label": "Late now", "records": shipments, "measure": "lateness",
		},
		map[string]any{
			"key": "furthest", "label": "Furthest behind", "records": shipments,
			"measure": "lateness", "aggregate": "max",
		},
	}
	args["verdict"] = "There are {late} late loads, the furthest {furthest} behind. " +
		"[Four wait on one driver's hours][3]."

	return args
}

func TestParseReadsReadingsTheWordsPlace(t *testing.T) {
	t.Parallel()

	parsed, problems := boardparse.Parse(readingBoard())

	require.Empty(t, problems)
	require.Len(t, parsed.Readings, 2)
	assert.Equal(t, "late", parsed.Readings[0].Key)
	assert.Equal(t, "Late now", parsed.Readings[0].Label)
	assert.Equal(t, board.Measure("lateness"), parsed.Readings[0].Measure)
	assert.Equal(t, board.AggregateCount, parsed.Readings[0].Aggregate)
	assert.Equal(t, board.AggregateMax, parsed.Readings[1].Aggregate)
	assert.Len(t, parsed.Readings[0].Records.IDs, 6)
}

func TestParsePlacesABoardBesideByDefault(t *testing.T) {
	t.Parallel()

	parsed, problems := boardparse.Parse(lateBoard())

	require.Empty(t, problems)
	assert.Equal(t, board.PlaceBeside, parsed.Place)
}

func TestParsePlacesASmallBoardInTheConversationWhenAsked(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["place"] = "conversation"
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Equal(t, board.PlaceConversation, parsed.Place)
}

func TestParsePlacesABoardBesideWhenAsked(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["place"] = "beside"
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Equal(t, board.PlaceBeside, parsed.Place)
}

func TestParsePutsALargeBoardBesideWithoutRefusingIt(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["place"] = "conversation"
	blocks := args["blocks"].([]any)
	args["blocks"] = append(blocks, map[string]any{
		"type": "prose", "text": "The rest wait on the dock.",
	})
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Len(t, parsed.Blocks, 5)
	assert.Equal(t, board.PlaceBeside, parsed.Place)
}

func TestParseIgnoresAPlaceItDoesNotKnow(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["place"] = "sidebar"
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Equal(t, board.PlaceBeside, parsed.Place)
}

func TestParseDropsAReadingTheWordsNeverPlace(t *testing.T) {
	t.Parallel()

	args := readingBoard()
	args["verdict"] = "There are {late} late loads. [Four wait on one driver's hours][3]."
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	require.Len(t, parsed.Readings, 1)
	assert.Equal(t, "late", parsed.Readings[0].Key)
}

func TestParseReadsReadingsPlacedInProseAndNotes(t *testing.T) {
	t.Parallel()

	args := readingBoard()
	args["verdict"] = "[Four wait on one driver's hours][3]."
	blocks := args["blocks"].([]any)
	blocks[3].(map[string]any)["text"] = "A team keeps the {furthest} one."
	args["blocks"] = append(blocks[:3:3], map[string]any{
		"type": "prose", "text": "Of them, {late} are late.",
	}, blocks[3])
	parsed, problems := boardparse.Parse(args)

	require.Empty(t, problems)
	assert.Len(t, parsed.Readings, 2)
}

func TestParseRefusesWhatReadingsCannotBe(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		change func(args map[string]any)
		want   string
	}{
		"a key no reading declares": {
			change: func(args map[string]any) {
				args["verdict"] = "There are {held} held. [Four wait][3]."
			},
			want: "verdict places {held}, which no reading declares",
		},
		"a reading inside cited words": {
			change: func(args map[string]any) {
				args["verdict"] = "There are {late} late. [{furthest} on one driver][3]."
			},
			want: "verdict: place {furthest} outside the cited words",
		},
		"a reading in a title": {
			change: func(args map[string]any) {
				args["title"] = "The {late} late shipments"
			},
			want: "title holds {late}; a reading is placed only in the verdict, a prose block " +
				"or a note",
		},
		"a key that is not a key": {
			change: func(args map[string]any) {
				readings := args["readings"].([]any)
				readings[0].(map[string]any)["key"] = "Late Now"
			},
			want: `readings[1].key "Late Now" is letters, digits and underscores`,
		},
		"a key declared twice": {
			change: func(args map[string]any) {
				readings := args["readings"].([]any)
				readings[1].(map[string]any)["key"] = "late"
			},
			want: `readings[2].key "late" is declared twice`,
		},
		"a reading with no source": {
			change: func(args map[string]any) {
				args["readings"] = []any{map[string]any{"key": "late", "label": "Late now"}}
			},
			want: "readings[1] needs exactly one source",
		},
		"a reading with a figure in its label": {
			change: func(args map[string]any) {
				readings := args["readings"].([]any)
				readings[0].(map[string]any)["label"] = "Late past 2h"
			},
			want: `readings[1].label states "2h"`,
		},
		"too many readings": {
			change: func(args map[string]any) {
				readings := make([]any, 0, board.MaxReadings+1)
				for range board.MaxReadings + 1 {
					readings = append(readings, map[string]any{
						"key": "late", "label": "Late", "metric": "movesLate",
					})
				}
				args["readings"] = readings
			},
			want: "readings is a list of at most 6 readings",
		},
		"a typed figure beside a reading": {
			change: func(args map[string]any) {
				args["verdict"] = "There are {late} late loads, 4 of them held. [Four wait][3]."
			},
			want: `verdict states "4 of"`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			args := readingBoard()
			tc.change(args)
			_, problems := boardparse.Parse(args)
			assert.True(t, strings.Contains(strings.Join(problems, " | "), tc.want),
				"problems: %v", problems)
		})
	}
}

func TestParseCountsAReadingsRecordsAgainstTheBoardsLimit(t *testing.T) {
	t.Parallel()

	args := readingBoard()
	args["readings"] = []any{map[string]any{
		"key": "late", "label": "Late", "records": ids("shp_", board.MaxListRecords),
		"measure": "lateness",
	}}
	blocks := make([]any, 0, 5)
	for range 4 {
		blocks = append(blocks, map[string]any{
			"type": "records", "records": ids("shp_", board.MaxListRecords),
		})
	}
	args["blocks"] = blocks
	args["verdict"] = "There are {late} late loads. [Four wait][1]."
	_, problems := boardparse.Parse(args)

	assert.Contains(t, problems, "the board names 125 records; it may name at most 120")
}

func TestParseDropsAReadingItCannotReadUnlessTheWordsPlaceIt(t *testing.T) {
	t.Parallel()

	unplaced := readingBoard()
	unplaced["readings"] = append(unplaced["readings"].([]any),
		map[string]any{"key": "profit", "label": "Profit", "metric": "profitToday"})
	parsed, problems := boardparse.Parse(unplaced)
	require.Empty(t, problems)
	assert.Len(t, parsed.Readings, 2)

	placed := readingBoard()
	placed["readings"] = append(placed["readings"].([]any),
		map[string]any{"key": "profit", "label": "Profit"})
	placed["verdict"] = placed["verdict"].(string) + " Profit is {profit}."
	_, problems = boardparse.Parse(placed)
	assert.Contains(t, strings.Join(problems, "\n"), "readings[3] needs exactly one source")
}

func TestParseRefusesTimeAddedUpAcrossRecords(t *testing.T) {
	t.Parallel()

	args := readingBoard()
	args["readings"].([]any)[1].(map[string]any)["aggregate"] = "sum"

	_, problems := boardparse.Parse(args)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], "readings[2].aggregate sum would add up lateness across")
	assert.Contains(t, problems[0], "which is not a figure anyone reads")
}
