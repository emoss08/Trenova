package boardassemble_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReadingFixture(t *testing.T) *fixture {
	t.Helper()

	f := newFixture(t)
	records := make([]any, 0, len(f.shipments))
	for _, id := range f.shipments {
		records = append(records, id.String())
	}
	args := lateBoard()
	args["place"] = "conversation"
	args["blocks"].([]any)[1].(map[string]any)["records"] = records
	args["blocks"].([]any)[2].(map[string]any)["record"] = records[0]
	args["blocks"].([]any)[0].(map[string]any)["figures"] = []any{
		map[string]any{"label": "Late now", "records": records, "measure": "lateness"},
	}
	args["readings"] = []any{
		map[string]any{
			"key":     "late",
			"label":   "Late now",
			"records": records,
			"measure": "lateness",
		},
		map[string]any{
			"key": "furthest", "label": "Furthest behind", "records": records,
			"measure": "lateness", "aggregate": "max",
		},
		map[string]any{"key": "active", "label": "Active", "metric": "activeShipments"},
	}
	args["verdict"] = "There are {late} late loads, the furthest {furthest} behind, " +
		"[held on one driver's hours][3]."
	args["blocks"].([]any)[3].(map[string]any)["text"] = "Of {active} active, re-tender these."
	parsed, problems := boardparse.Parse(args)
	require.Empty(t, problems)
	f.board = parsed

	return f
}

func TestAssembleFillsEachReadingTheWordsPlace(t *testing.T) {
	t.Parallel()

	f := newReadingFixture(t)
	filled := boardassemble.Assemble(f.board, f.reads)

	assert.Equal(t, board.PlaceConversation, filled.Place)
	require.Len(t, filled.Readings, 3)
	late := filled.Reading("late")
	require.NotNil(t, late)
	assert.Equal(t, "2", board.FormatValue(late.Value))
	assert.Equal(t, "Late now", late.Label)
	assert.Equal(t, board.MeasureLateness, late.Measure)
	assert.Equal(t, board.AggregateCount, late.Aggregate)
	assert.Equal(t, board.Source{
		Kind: board.SourceRecords, Resource: permission.ResourceShipment, Count: 6, Readable: 4,
	}, late.Source)
	assert.Equal(t, "1h 40m", board.FormatValue(filled.Reading("furthest").Value))
	active := filled.Reading("active")
	assert.Equal(t, "41", board.FormatValue(active.Value))
	assert.Equal(t, board.SourceMetrics, active.Source.Kind)

	assert.Equal(t, []board.FilledSegment{
		{Text: "There are "},
		{Text: "2", Reading: "late"},
		{Text: " late loads, the furthest "},
		{Text: "1h 40m", Reading: "furthest"},
		{Text: " behind, "},
		{Text: "held on one driver's hours", Numeral: 3, Tone: board.ToneInfo},
		{Text: "."},
	}, filled.Verdict)
	assert.Equal(t, []board.FilledSegment{
		{Text: "Of "},
		{Text: "41", Reading: "active"},
		{Text: " active, re-tender these."},
	}, filled.Blocks[3].Text)
	assert.Equal(t, "41",
		filled.Snapshot().Values[board.ReadingKey("active")].Number.String())
}

func TestAssembleLeavesAReadingTheReaderMayNotReadWithoutAValue(t *testing.T) {
	t.Parallel()

	f := newReadingFixture(t)
	f.reads.Readable[permission.ResourceShipment] = false
	delete(f.reads.Metrics, "activeShipments")
	filled := boardassemble.Assemble(f.board, f.reads)

	late := filled.Reading("late")
	assert.Nil(t, late.Value)
	assert.Equal(t, 6, late.Source.Count)
	assert.Zero(t, late.Source.Readable)
	assert.Nil(t, filled.Reading("active").Value)
	assert.Equal(t, board.FilledSegment{Reading: "late"}, filled.Verdict[1])
	assert.Equal(t, "There are — late loads, the furthest — behind, held on one driver's hours.",
		boardmarkdown.Answer(filled))
}

func TestAssembleMarksAReadingThatMovedAndKeepsTheAnsweredValue(t *testing.T) {
	t.Parallel()

	f := newReadingFixture(t)
	answered := boardassemble.Assemble(f.board, f.reads)
	snapshot := answered.Snapshot()
	assert.Equal(t, "There are 2 late loads, the furthest 1h 40m behind, held on one driver's "+
		"hours.", boardmarkdown.Answer(answered))

	summary := f.reads.Summaries[permission.ResourceShipment][f.shipments[1]]
	summary.Measures[board.MeasureLateness] = lateness(0, board.MeasureDelivered)
	f.reads.Summaries[permission.ResourceShipment][f.shipments[1]] = summary
	now := boardassemble.Assemble(f.board, f.reads)
	now.Compare(&snapshot)

	late := now.Reading("late")
	assert.Equal(t, "1", board.FormatValue(late.Value))
	assert.Equal(t, "2", board.FormatValue(late.Was))
	assert.Nil(t, now.Reading("furthest").Was)
	assert.Equal(t, "2", now.Verdict[1].Text, "the sentence keeps what was answered")

	markdown := boardmarkdown.Markdown(now, boardmarkdown.MarkdownMeta{Location: time.UTC})
	assert.True(t, strings.Contains(markdown,
		"There are 2 (now 1) late loads, the furthest 1h 40m behind, held on one driver's "+
			"hours [3]."), markdown)
	assert.True(t, strings.Contains(markdown,
		"**What I'd do: Re-tender the two furthest behind.** Of 41 active, re-tender these."),
		markdown)
}

func TestAssembleFillsAReadingInABlocksCaption(t *testing.T) {
	t.Parallel()

	f := newReadingFixture(t)
	f.board.Blocks[1].Caption = "The furthest is {furthest} behind."
	filled := boardassemble.Assemble(f.board, f.reads)

	assert.Equal(t, []board.FilledSegment{
		{Text: "The furthest is "},
		{Text: "1h 40m", Reading: "furthest"},
		{Text: " behind."},
	}, filled.Blocks[1].Caption)
	assert.Empty(t, filled.Blocks[0].Caption)
	assert.Contains(t, boardmarkdown.Markdown(filled, boardmarkdown.MarkdownMeta{}),
		"_The furthest is 1h 40m behind._")

	snapshot := filled.Snapshot()
	f.reads.Now++
	again := boardassemble.Assemble(f.board, f.reads)
	again.Compare(&snapshot)
	assert.Equal(t, "1h 40m", again.Blocks[1].Caption[1].Text,
		"the caption keeps what was answered, as the verdict does")
}
