package board_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_793_000_000)

func lateness(seconds int64, state board.MeasureState) board.RowMeasure {
	tone := board.Tone("")
	if seconds > 0 {
		tone = board.ToneWarn
	}

	return board.RowMeasure{
		Measure: board.MeasureLateness,
		Value:   board.Value{Kind: board.ValueDuration, Number: decimal.NewFromInt(seconds)},
		State:   state,
		Tone:    tone,
	}
}

type fixture struct {
	shipments []pulid.ID
	board     *board.Board
	reads     *board.Reads
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	args := lateBoard()
	parsed, problems := board.Parse(args)
	require.Empty(t, problems)
	shipments := parsed.Blocks[1].Records.IDs

	summaries := map[pulid.ID]board.Summary{}
	behind := []int64{6000, 3300, 0, 0}
	for idx, id := range shipments[:4] {
		summaries[id] = board.Summary{
			Label:    "S2610-04" + string(rune('1'+idx)),
			Subtitle: "Dallas, TX → Memphis, TN",
			Status:   "InTransit",
			Measures: map[board.Measure]board.RowMeasure{
				board.MeasureLateness: lateness(behind[idx], board.MeasureLate),
			},
		}
	}

	return &fixture{
		shipments: shipments,
		board:     parsed,
		reads: &board.Reads{
			Now:      now,
			Readable: map[permission.Resource]bool{permission.ResourceShipment: true},
			Summaries: map[permission.Resource]map[pulid.ID]board.Summary{
				permission.ResourceShipment: summaries,
			},
			Metrics: map[string]*board.Value{
				"activeShipments": {Kind: board.ValueCount, Number: decimal.NewFromInt(41)},
			},
			Events: map[pulid.ID][]board.Event{
				shipments[0]: {
					{At: now - 7200, Operation: "update", Changes: map[string]any{"status": "x"}},
					{At: now - 3600, Operation: "update", Comment: "Driver out of hours"},
				},
			},
		},
	}
}

func TestAssembleFillsEachBlockFromWhatTheViewerMayRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	filled := board.Assemble(f.board, f.reads)

	figures := filled.Blocks[0]
	assert.Equal(t, board.StateReady, figures.State)
	assert.Equal(t, board.SourceMixed, figures.Source.Kind)
	assert.Equal(t, "2", board.FormatValue(figures.Figures[0].Value))
	assert.Equal(t, board.ToneWarn, figures.Figures[0].Tone)
	assert.Equal(t, "1h 40m", board.FormatValue(figures.Figures[1].Value))
	assert.Equal(t, "41", board.FormatValue(figures.Figures[2].Value))

	records := filled.Blocks[1]
	require.Len(t, records.Rows, 6)
	assert.Equal(t, f.shipments[0], records.Rows[0].ID)
	assert.Equal(t, f.shipments[1], records.Rows[1].ID)
	assert.True(t, records.Rows[3].Readable)
	assert.False(t, records.Rows[4].Readable)
	assert.False(t, records.Rows[5].Readable)
	assert.Empty(t, records.Rows[4].Label)
	assert.Equal(t, 4, records.Source.Readable)
	assert.Equal(t, 6, records.Source.Count)
	assert.Equal(t, board.ToneWarn, records.Tone)

	timeline := filled.Blocks[2]
	assert.Equal(t, board.StateReady, timeline.State)
	require.Len(t, timeline.Events, 2)
	assert.Equal(t, "Driver out of hours", board.EventSentence(&timeline.Events[1]))

	assert.Equal(t, []int{1, 2, 3, 0}, []int{
		filled.Blocks[0].Numeral, filled.Blocks[1].Numeral,
		filled.Blocks[2].Numeral, filled.Blocks[3].Numeral,
	})
	assert.Equal(t, []board.FilledSegment{
		{Text: "Four of the six are held behind "},
		{Text: "one driver's hours", Numeral: 3, Tone: board.ToneInfo},
		{Text: ", and two are "},
		{Text: "late at the dock", Numeral: 2, Tone: board.ToneWarn},
		{Text: "."},
	}, filled.Verdict)
}

func TestAssembleDrawsABlockWhoseSourceCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.reads.Readable[permission.ResourceShipment] = false
	delete(f.reads.Metrics, "activeShipments")

	filled := board.Assemble(f.board, f.reads)

	assert.Equal(t, board.StateUnreadable, filled.Blocks[0].State)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[1].State)
	assert.Nil(t, filled.Blocks[1].Rows)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[2].State)
	assert.True(t, filled.Verdict[1].Struck)
	assert.True(t, filled.Verdict[3].Struck)
}

func TestAssembleMarksAFigureThatMovedSinceTheAnswer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	answered := board.Assemble(f.board, f.reads)
	snapshot := answered.Snapshot()

	summary := f.reads.Summaries[permission.ResourceShipment][f.shipments[1]]
	summary.Measures[board.MeasureLateness] = lateness(0, board.MeasureDelivered)
	f.reads.Summaries[permission.ResourceShipment][f.shipments[1]] = summary
	now := board.Assemble(f.board, f.reads)
	now.Compare(&snapshot)

	assert.Equal(t, "1", board.FormatValue(now.Blocks[0].Figures[0].Value))
	assert.Equal(t, "2", board.FormatValue(now.Blocks[0].Figures[0].Was))
	assert.Nil(t, now.Blocks[0].Figures[1].Was)
	assert.Nil(t, now.Blocks[0].Figures[2].Was)
}

func TestAssembleComparesRecordsFromAnAnalysisTable(t *testing.T) {
	t.Parallel()

	carriers := []pulid.ID{pulid.MustNew("car_"), pulid.MustNew("car_"), pulid.MustNew("car_")}
	analysisID := pulid.MustNew("art_")
	parsed, problems := board.Parse(map[string]any{
		"title":   "Carriers head to head",
		"verdict": "Blue Line wins [most rows][1].",
		"blocks": []any{map[string]any{
			"type":     "compare",
			"records":  []any{carriers[0].String(), carriers[1].String(), carriers[2].String()},
			"analysis": analysisID.String(),
			"table":    1.0,
			"key":      "carrier_id",
			"rows": []any{
				map[string]any{"column": "on_time", "label": "On-time delivery", "better": "higher"},
				map[string]any{"column": "cpm", "label": "Cost per mile", "better": "lower"},
				map[string]any{"column": "claims", "better": "lower"},
			},
		}},
	})
	require.Empty(t, problems)

	summaries := map[pulid.ID]board.Summary{
		carriers[0]: {Label: "Acme Freight", Status: "Active"},
		carriers[1]: {Label: "Blue Line", Status: "Active"},
	}
	reads := &board.Reads{
		Now:      now,
		Readable: map[permission.Resource]bool{permission.ResourceCarrier: true},
		Summaries: map[permission.Resource]map[pulid.ID]board.Summary{
			permission.ResourceCarrier: summaries,
		},
		Analyses: map[pulid.ID]*board.AnalysisRead{analysisID: {
			Title: "Carrier scorecard",
			Display: analysis.Display{Tables: []analysis.Table{{
				Columns: []analysis.TableColumn{
					{Name: "carrier_id", Kind: analysis.ValueText},
					{Name: "on_time", Kind: analysis.ValuePercent},
					{Name: "cpm", Kind: analysis.ValueMoney, Currency: "USD"},
					{Name: "claims", Kind: analysis.ValueCount},
				},
				Rows: [][]any{
					{carriers[0].String(), 0.918, 2.31, 0.0},
					{carriers[1].String(), 0.974, 2.48, 0.0},
					{carriers[2].String(), 0.931, 2.39, 2.0},
				},
			}}},
		}},
	}

	filled := board.Assemble(parsed, reads)
	compare := filled.Blocks[0].Compare
	require.NotNil(t, compare)

	assert.Equal(t, "Carrier scorecard", filled.Blocks[0].Source.Title)
	assert.False(t, compare.Columns[2].Readable)
	require.Len(t, compare.Lines, 3)
	assert.Equal(t, "On-time delivery", compare.Lines[0].Label)
	assert.Equal(t, []bool{false, true, false}, compare.Lines[0].Wins)
	assert.Equal(t, []bool{true, false, false}, compare.Lines[1].Wins)
	assert.Equal(t, "claims", compare.Lines[2].Label)
	assert.Equal(t, []bool{true, true, false}, compare.Lines[2].Wins)
	assert.Nil(t, compare.Lines[2].Cells[2])
	assert.Equal(t, []int{2, 2, 0}, compare.Wins)
}

func TestAssembleCountsChecksAndDrawsTheirMeter(t *testing.T) {
	t.Parallel()

	entries := ids("je_", 3)
	invoices := ids("inv_", 4)
	parsed, problems := board.Parse(map[string]any{
		"title":   "October close",
		"verdict": "Two things hold [October open][1].",
		"blocks": []any{
			map[string]any{"type": "checks", "checks": []any{
				map[string]any{"title": "Entries waiting", "tone": "bad", "records": entries},
				map[string]any{"title": "Invoices not synced", "tone": "warn", "records": invoices},
				map[string]any{"title": "Bank matched", "tone": "ok", "metric": "arOverdue"},
			}},
			map[string]any{"type": "meter", "checksBlock": 1.0},
		},
	})
	require.Empty(t, problems)

	readableEntries := map[pulid.ID]board.Summary{}
	for _, raw := range entries[:2] {
		id, _ := pulid.Parse(raw.(string))
		readableEntries[id] = board.Summary{Label: "JE-" + id.String()[3:7], Status: "Pending"}
	}
	reads := &board.Reads{
		Now: now,
		Readable: map[permission.Resource]bool{
			permission.ResourceJournalEntry: true,
			permission.ResourceInvoice:      false,
		},
		Summaries: map[permission.Resource]map[pulid.ID]board.Summary{
			permission.ResourceJournalEntry: readableEntries,
		},
		Metrics: map[string]*board.Value{},
	}

	filled := board.Assemble(parsed, reads)
	checks := filled.Blocks[0].Checks

	assert.Equal(t, "2", board.FormatValue(checks[0].Count))
	assert.Len(t, checks[0].Records, 2)
	assert.Equal(t, 1, checks[0].Hidden)
	assert.Nil(t, checks[1].Count)
	assert.Nil(t, checks[2].Count)
	assert.Equal(t, board.ToneBad, filled.Blocks[0].Tone)
	assert.Equal(t, board.ToneWarn, filled.Verdict[1].Tone)

	meter := filled.Blocks[1].Meter
	require.NotNil(t, meter)
	assert.Equal(t, 1, meter.Clear)
	assert.Equal(t, 3, meter.Total)
	assert.Equal(t, []board.Tone{board.ToneOK, board.ToneWarn, board.ToneBad}, meter.Tones)
}

func TestMarkdownWritesTheBoardAsItIsReadNow(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	answered := board.Assemble(f.board, f.reads)
	snapshot := answered.Snapshot()
	summary := f.reads.Summaries[permission.ResourceShipment][f.shipments[1]]
	summary.Measures[board.MeasureLateness] = lateness(0, board.MeasureDelivered)
	f.reads.Summaries[permission.ResourceShipment][f.shipments[1]] = summary
	projected := lateness(360, board.MeasureProjected)
	projected.Place = "Seattle, WA"
	f.reads.Summaries[permission.ResourceShipment][f.shipments[2]].Measures[board.MeasureLateness] =
		projected
	f.reads.Summaries[permission.ResourceShipment][f.shipments[3]].Measures[board.MeasureLateness] =
		lateness(0, board.MeasureDue)
	filled := board.Assemble(f.board, f.reads)
	filled.Compare(&snapshot)

	markdown := filled.Markdown(board.MarkdownMeta{
		Question:   "Why are these 6 shipments late?",
		Author:     "Dispatch Desk",
		AnsweredAt: now - 600,
		Location:   time.UTC,
	})

	for _, want := range []string{
		"# Six late shipments on the Dallas lanes\n",
		"> Why are these 6 shipments late?\n",
		"Four of the six are held behind one driver's hours [3], and two are late at the dock [2].",
		"## 1. Where they stand\n",
		"- Late now: 1 (2 when answered)\n",
		"- Furthest behind: 1h 40m\n",
		"## 2. The six, furthest behind first\n",
		"| Shipment | Detail | Status | Behind |\n",
		"| S2610-041 | Dallas, TX → Memphis, TN | InTransit | 1h 40m |\n",
		"| S2610-042 | Dallas, TX → Memphis, TN | InTransit | on time |\n",
		"| S2610-043 | Dallas, TX → Memphis, TN | InTransit | estimated 6m late (Seattle, WA) |\n",
		"| S2610-044 | Dallas, TX → Memphis, TN | InTransit | not estimated |\n",
		"2 shipments could not be read.\n",
		"## 3. The history of shipment\n",
		": Driver out of hours\n",
		"**What I'd do: Re-tender the two furthest behind.** A team keeps the appointments.\n",
		"Answered by Dispatch Desk at",
	} {
		assert.True(t, strings.Contains(markdown, want), "want %q in\n%s", want, markdown)
	}
}
