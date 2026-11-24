package boardassemble_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardbar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	credentialKind = recordkind.Kind(permission.ResourceWorkerCredential)
	workerKind     = recordkind.Kind(permission.ResourceWorker)
	expiryKey      = board.MeasureKey{Kind: credentialKind, Measure: "expiry"}
)

func expiring(left int64) boardreading.Reading {
	value := boardcatalog.Days(left / 86400)
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: "{0} left", Value: &value},
		Fact:     &boardreading.Words{Text: "Expires {0}", At: now + left, Day: true},
		Window:   &boardreading.WindowFacts{Left: left, Span: 30 * 86_400},
	}
	if left < 0 {
		reading.Standing.Text = "{0} lapsed"
		reading.Tone = boardcatalog.ToneBad
	}

	return reading
}

func credentialBoard(t *testing.T, credentials []pulid.ID) *board.Board {
	t.Helper()
	records := make([]any, 0, len(credentials))
	for _, id := range credentials {
		records = append(records, id.String())
	}
	parsed, problems := boardparse.Parse(map[string]any{
		"title":   "Credentials running out",
		"verdict": "Two run out [this month][1].",
		"place":   "conversation",
		"blocks": []any{map[string]any{
			"type": "records", "records": records, "measure": "expiresAt",
		}},
	})
	require.Empty(t, problems)

	return parsed
}

func TestAssembleOrdersACountdownSoonestFirstAndOpensTheWorker(t *testing.T) {
	t.Parallel()

	credentials := []pulid.ID{
		pulid.MustNew("wcred_"), pulid.MustNew("wcred_"), pulid.MustNew("wcred_"),
	}
	worker := pulid.MustNew("wrk_")
	layout := credentialBoard(t, credentials)
	assert.Equal(t, board.Measure("expiry"), layout.Blocks[0].Measure)

	summaries := map[pulid.ID]boardreading.Summary{}
	for _, id := range credentials {
		summaries[id] = boardreading.Summary{
			Label:      "Carlos Rivera",
			Subtitle:   "DOT Medical Card",
			Status:     "Expiring soon",
			StatusTone: boardcatalog.ToneWarn,
			Opens:      &boardcatalog.Ref{Kind: workerKind, ID: worker},
		}
	}
	filled := boardassemble.Assemble(layout, &boardassemble.Reads{
		Now:       now,
		Readable:  map[recordkind.Kind]bool{credentialKind: true},
		Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{credentialKind: summaries},
		Readings: map[board.MeasureKey]map[pulid.ID]boardreading.Reading{expiryKey: {
			credentials[0]: expiring(21 * 86400),
			credentials[1]: expiring(-2 * 86400),
			credentials[2]: expiring(3 * 86400),
		}},
	})

	rows := filled.Blocks[0].Rows
	require.Len(t, rows, 3)
	assert.Equal(t, []pulid.ID{credentials[1], credentials[2], credentials[0]},
		[]pulid.ID{rows[0].ID, rows[1].ID, rows[2].ID})
	assert.Equal(t, credentialKind, rows[0].Kind)
	assert.Equal(t, &boardcatalog.Ref{Kind: workerKind, ID: worker}, rows[0].Opens)
	assert.Equal(t, boardcatalog.ToneBad, filled.Blocks[0].Tone)
	require.NotNil(t, rows[0].Measure.Bar)
	assert.Equal(t, boardbar.SegmentBeyond, rows[0].Measure.Bar.Segments[0].Kind)
	assert.Equal(t, boardbar.SegmentUsed, rows[2].Measure.Bar.Segments[0].Kind)
	assert.InDelta(t, 0.1, rows[1].Measure.Bar.Segments[0].Width, 0.0001,
		"three days of a thirty-day window is a tenth of the track")
	assert.Equal(t, boardcatalog.ToneWarn, rows[0].StatusTone)
	assert.Equal(t, "{0} left", rows[1].Measure.Standing.Text)
}

func TestAssembleWithholdsAProtectedMeasureAndItsFigures(t *testing.T) {
	t.Parallel()

	credentials := []pulid.ID{pulid.MustNew("wcred_"), pulid.MustNew("wcred_")}
	layout := credentialBoard(t, credentials)
	layout.Readings = []board.Reading{{Key: "soonest", FigureSource: board.FigureSource{
		Label:     "Soonest",
		Records:   &board.RecordSet{Kind: credentialKind, IDs: credentials},
		Measure:   "expiry",
		Aggregate: board.AggregateMax,
	}}}
	summaries := map[pulid.ID]boardreading.Summary{
		credentials[0]: {Label: "A"},
		credentials[1]: {Label: "B"},
	}
	filled := boardassemble.Assemble(layout, &boardassemble.Reads{
		Now:       now,
		Readable:  map[recordkind.Kind]bool{credentialKind: true},
		Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{credentialKind: summaries},
		Withheld:  map[board.MeasureKey]bool{expiryKey: true},
	})

	for _, row := range filled.Blocks[0].Rows {
		require.NotNil(t, row.Measure)
		assert.Nil(t, row.Measure.Value)
		assert.Nil(t, row.Measure.Bar)
	}
	assert.Nil(t, filled.Reading("soonest").Value)
}

func TestAssembleTakesTheSoonestAsTheLargestOfACountdown(t *testing.T) {
	t.Parallel()

	credentials := []pulid.ID{pulid.MustNew("wcred_"), pulid.MustNew("wcred_")}
	layout := credentialBoard(t, credentials)
	layout.Readings = []board.Reading{{Key: "soonest", FigureSource: board.FigureSource{
		Label:     "Soonest",
		Records:   &board.RecordSet{Kind: credentialKind, IDs: credentials},
		Measure:   "expiry",
		Aggregate: board.AggregateMax,
	}}}
	filled := boardassemble.Assemble(layout, &boardassemble.Reads{
		Now:      now,
		Readable: map[recordkind.Kind]bool{credentialKind: true},
		Summaries: map[recordkind.Kind]map[pulid.ID]boardreading.Summary{credentialKind: {
			credentials[0]: {Label: "A"},
			credentials[1]: {Label: "B"},
		}},
		Readings: map[board.MeasureKey]map[pulid.ID]boardreading.Reading{expiryKey: {
			credentials[0]: expiring(9 * 86400),
			credentials[1]: expiring(3 * 86400),
		}},
	})

	assert.Equal(t, "3 days", board.FormatValue(filled.Reading("soonest").Value))
}

func TestAssembleLeavesAMetricBlockTheReaderMayNotReadUnreadable(t *testing.T) {
	t.Parallel()

	layout, problems := boardparse.Parse(map[string]any{
		"title":   "Today",
		"verdict": "Here is [today][1].",
		"blocks":  []any{map[string]any{"type": "metric", "metric": "dispatchToday"}},
	})
	require.Empty(t, problems)
	assert.Equal(t, "today", layout.Blocks[0].Period)

	unread := boardassemble.Assemble(layout, &boardassemble.Reads{Now: now})
	assert.Equal(t, board.StateUnreadable, unread.Blocks[0].State)
	assert.True(t, unread.Verdict[1].Struck)

	read := boardassemble.Assemble(layout, &boardassemble.Reads{
		Now: now,
		Metrics: map[board.MetricAsk]*boardcatalog.MetricRead{dispatchAsk: {
			Title:   "Today's dispatch",
			Figures: map[string]*boardcatalog.Value{"activeShipments": count(41)},
		}},
	})
	assert.Equal(t, board.StateReady, read.Blocks[0].State)
	require.NotNil(t, read.Blocks[0].Metric)
	assert.Equal(t, boardcatalog.FormFigures, read.Blocks[0].Metric.Form)
	assert.Equal(t, "Today's dispatch", read.Blocks[0].Source.Title)
}
