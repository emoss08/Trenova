package boardassemble_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_793_000_000)

var (
	shipmentKind = recordkind.Kind(permission.ResourceShipment)
	lateKey      = board.MeasureKey{Kind: shipmentKind, Measure: "lateness"}
	dispatchAsk  = board.AskOf(homeboard.MetricDispatchToday, pulid.Nil, "")
)

func lateness(seconds int64) boardreading.Reading {
	value := boardcatalog.Duration(seconds)
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: "On time"},
		Track:    &boardreading.TrackFacts{Behind: seconds},
	}
	if seconds > 0 {
		reading.Standing = boardreading.Words{Text: "{0} behind", Value: &value}
		reading.Tone = boardcatalog.ToneWarn
	}

	return reading
}

func count(n int64) *boardcatalog.Value {
	value := boardcatalog.Count(n)

	return &value
}

type fixture struct {
	shipments []pulid.ID
	board     *board.Board
	reads     *boardassemble.Reads
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	args := lateBoard()
	parsed, problems := boardparse.Parse(args)
	require.Empty(t, problems)
	shipments := parsed.Blocks[1].Records.IDs

	summaries := map[pulid.ID]boardreading.Summary{}
	readings := map[pulid.ID]boardreading.Reading{}
	behind := []int64{6000, 3300, 0, 0, 0, 0}
	for idx, id := range shipments {
		summaries[id] = boardreading.Summary{
			Label:    "S2610-04" + string(rune('1'+idx)),
			Subtitle: "Dallas, TX → Memphis, TN",
			Status:   "InTransit",
		}
		readings[id] = lateness(behind[idx])
	}

	return &fixture{
		shipments: shipments,
		board:     parsed,
		reads: &boardassemble.Reads{
			Now:      now,
			Readable: map[recordkind.Kind]bool{shipmentKind: true},
			Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{
				shipmentKind: summaries,
			},
			Readings: map[board.MeasureKey]map[pulid.ID]boardreading.Reading{lateKey: readings},
			Metrics: map[board.MetricAsk]*boardcatalog.MetricRead{
				dispatchAsk: {Figures: map[string]*boardcatalog.Value{"activeShipments": count(41)}},
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
	delete(f.reads.Summaries[shipmentKind], f.shipments[4])
	delete(f.reads.Summaries[shipmentKind], f.shipments[5])
	filled := boardassemble.Assemble(f.board, f.reads)

	figures := filled.Blocks[0]
	assert.Equal(t, board.StateReady, figures.State)
	assert.Equal(t, board.SourceMixed, figures.Source.Kind)
	assert.Nil(t, figures.Figures[0].Value,
		"a count over records the viewer cannot all read is withheld, never the visible ones")
	assert.Nil(t, figures.Figures[1].Value)
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
	assert.Equal(t, boardcatalog.ToneWarn, records.Tone)

	timeline := filled.Blocks[2]
	assert.Equal(t, board.StateReady, timeline.State)
	require.Len(t, timeline.Events, 2)
	assert.Equal(t, "Driver out of hours", boardmarkdown.EventSentence(&timeline.Events[1]))

	assert.Equal(t, []int{1, 2, 3, 0}, []int{
		filled.Blocks[0].Numeral, filled.Blocks[1].Numeral,
		filled.Blocks[2].Numeral, filled.Blocks[3].Numeral,
	})
	assert.Equal(t, []board.FilledSegment{
		{Text: "Four of the six are held behind "},
		{Text: "one driver's hours", Numeral: 3, Tone: boardcatalog.ToneInfo},
		{Text: ", and two are "},
		{Text: "late at the dock", Numeral: 2, Tone: boardcatalog.ToneWarn},
		{Text: "."},
	}, filled.Verdict)
}

func TestAssembleDrawsABlockWhoseSourceCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.reads.Readable[shipmentKind] = false
	delete(f.reads.Metrics, dispatchAsk)

	filled := boardassemble.Assemble(f.board, f.reads)

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
	answered := boardassemble.Assemble(f.board, f.reads)
	snapshot := answered.Snapshot()

	f.reads.Readings[lateKey][f.shipments[1]] = lateness(0)
	now := boardassemble.Assemble(f.board, f.reads)
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
	parsed, problems := boardparse.Parse(map[string]any{
		"title":   "Carriers head to head",
		"verdict": "Blue Line wins [most rows][1].",
		"blocks": []any{map[string]any{
			"type":     "compare",
			"records":  []any{carriers[0].String(), carriers[1].String(), carriers[2].String()},
			"analysis": analysisID.String(),
			"table":    1.0,
			"key":      "carrier_id",
			"rows": []any{
				map[string]any{
					"column": "on_time",
					"label":  "On-time delivery",
					"better": "higher",
				},
				map[string]any{"column": "cpm", "label": "Cost per mile", "better": "lower"},
				map[string]any{"column": "claims", "better": "lower"},
			},
		}},
	})
	require.Empty(t, problems)

	carrierKind := recordkind.Kind(permission.ResourceCarrier)
	summaries := map[pulid.ID]boardreading.Summary{
		carriers[0]: {Label: "Acme Freight", Status: "Active"},
		carriers[1]: {Label: "Blue Line", Status: "Active"},
	}
	reads := &boardassemble.Reads{
		Now:      now,
		Readable: map[recordkind.Kind]bool{carrierKind: true},
		Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{
			carrierKind: summaries,
		},
		Analyses: map[pulid.ID]*boardassemble.AnalysisRead{analysisID: {
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

	filled := boardassemble.Assemble(parsed, reads)
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
	parsed, problems := boardparse.Parse(map[string]any{
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

	entryKind := recordkind.Kind(permission.ResourceJournalEntry)
	readableEntries := map[pulid.ID]boardreading.Summary{}
	for _, raw := range entries[:2] {
		id, _ := pulid.Parse(raw.(string))
		readableEntries[id] = boardreading.Summary{
			Label:  "JE-" + id.String()[3:7],
			Status: "Pending",
		}
	}
	reads := &boardassemble.Reads{
		Now: now,
		Readable: map[recordkind.Kind]bool{
			entryKind: true,
			recordkind.Kind(permission.ResourceInvoice): false,
		},
		Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{
			entryKind: readableEntries,
		},
		Metrics: map[board.MetricAsk]*boardcatalog.MetricRead{},
	}

	filled := boardassemble.Assemble(parsed, reads)
	checks := filled.Blocks[0].Checks

	assert.Equal(t, "2", board.FormatValue(checks[0].Count))
	assert.Len(t, checks[0].Records, 2)
	assert.Equal(t, 1, checks[0].Hidden)
	assert.Nil(t, checks[1].Count)
	assert.Equal(t, 4, checks[1].Hidden)
	assert.Nil(t, checks[2].Count)
	assert.Equal(t, boardcatalog.ToneBad, filled.Blocks[0].Tone)
	assert.Equal(t, boardcatalog.ToneWarn, filled.Verdict[1].Tone)

	meter := filled.Blocks[1].Meter
	require.NotNil(t, meter)
	assert.Equal(t, 1, meter.Clear)
	assert.Equal(t, 3, meter.Total)
	assert.Equal(
		t,
		[]boardcatalog.Tone{boardcatalog.ToneOK, boardcatalog.ToneWarn, boardcatalog.ToneBad},
		meter.Tones,
	)
}

func TestMarkdownWritesTheBoardAsItIsReadNow(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	answered := boardassemble.Assemble(f.board, f.reads)
	snapshot := answered.Snapshot()
	f.reads.Readings[lateKey][f.shipments[1]] = lateness(0)
	estimate := boardcatalog.Duration(360)
	f.reads.Readings[lateKey][f.shipments[2]] = boardreading.Reading{
		Value:    estimate,
		Standing: boardreading.Words{Text: "Estimated {0} late", Value: &estimate},
		Fact:     &boardreading.Words{Text: "Due {0}", At: now + 600, Place: "at Seattle, WA"},
		Tone:     boardcatalog.ToneWarn,
		Track:    &boardreading.TrackFacts{Behind: 360, Projected: true},
	}
	f.reads.Readings[lateKey][f.shipments[3]] = boardreading.Reading{
		Value:    boardcatalog.Duration(0),
		Standing: boardreading.Words{Text: "Not estimated"},
		Track:    &boardreading.TrackFacts{},
	}
	filled := boardassemble.Assemble(f.board, f.reads)
	filled.Compare(&snapshot)

	markdown := boardmarkdown.Markdown(filled, boardmarkdown.MarkdownMeta{
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
		"| S2610-041 | Dallas, TX → Memphis, TN | InTransit | 1h 40m behind |\n",
		"| S2610-042 | Dallas, TX → Memphis, TN | InTransit | On time |\n",
		"| S2610-043 | Dallas, TX → Memphis, TN | InTransit | Estimated 6m late (Due ",
		" at Seattle, WA) |\n",
		"| S2610-044 | Dallas, TX → Memphis, TN | InTransit | Not estimated |\n",
		"| S2610-045 | Dallas, TX → Memphis, TN | InTransit | On time |\n",
		"## 3. The history of shipment\n",
		": Driver out of hours\n",
		"**What I'd do: Re-tender the two furthest behind.** A team keeps the appointments.\n",
		"Answered by Dispatch Desk at",
	} {
		assert.True(t, strings.Contains(markdown, want), "want %q in\n%s", want, markdown)
	}
}
