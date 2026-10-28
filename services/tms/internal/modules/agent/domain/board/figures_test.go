package board_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/stretchr/testify/assert"
)

func TestFiguresFindsWhatTheBoardShouldReadLive(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		text string
		want []string
	}{
		"durations": {
			"the delivery window closed 22h 33m ago, 1557 minutes past it",
			[]string{"22h", "33m", "1557 minutes"},
		},
		"money and percentages": {"it owes $1,240.50, 15% of the total", []string{"$1,240.50", "15%"}},
		"a count":               {"the board marks all 40 moves as Late", []string{"40 moves"}},
		"identifiers": {
			"SEED-SHP-010 and S26100294798288 on I-35 with a 3PL in Q3",
			[]string{},
		},
		"times of day and dates": {
			"the window closed at 6:00 PM, arrival 5:05 AM on Oct 30, 2026, or 10/31 at 7pm",
			[]string{},
		},
		"words for numbers": {"Four loads are late and two are held", []string{}},
		"a span of time named from now": {
			"The board covers the next 24 hours from 9:23 AM, against the past 7 days",
			[]string{},
		},
		"a span beside a reading still finds the reading": {
			"over the last 3 days, 40 moves ran 22h late",
			[]string{"40 moves", "22h"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, board.Figures(tc.text))
		})
	}
}

func TestParseRefusesFiguresWrittenIntoTheWords(t *testing.T) {
	t.Parallel()

	args := lateBoard()
	args["verdict"] = "SEED-SHP-010 is [22h 33m behind][2]."

	_, problems := board.Parse(args)

	assert.Equal(t, []string{
		`verdict states "22h", "33m"; the board reads every figure live from its blocks, ` +
			"so take it out of the words. Where the board should show it, add a block that " +
			"reads it (records or checks for the loads it counts, figures for a metric) and " +
			"cite that block; never point at a block the board does not have",
	}, problems)
}
