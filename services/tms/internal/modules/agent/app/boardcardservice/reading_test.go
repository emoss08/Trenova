package boardcardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readingBoard(shipment pulid.ID) map[string]any {
	return map[string]any{
		"title": "Late loads",
		"place": "conversation",
		"readings": []any{map[string]any{
			"key": "late", "label": "Late now",
			"records": []any{shipment.String()}, "measure": "lateness",
		}},
		"verdict": "There are {late} late loads, [on the Dallas lanes][1].",
		"blocks": []any{map[string]any{
			"type": "records", "records": []any{shipment.String()}, "measure": "lateness",
		}},
	}
}

func TestCardKeepsTheAnsweredReadingAndMarksItMoved(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment := pulid.MustNew("shp_")
	f.shipments.known[shipment] = boardreading.Summary{Label: "S2610-0412"}
	f.late.readings = map[pulid.ID]boardreading.Reading{shipment: lateness(0)}
	shown := parsed(t, readingBoard(shipment))
	snapshot := board.Snapshot{Values: map[string]boardcatalog.Value{
		board.ReadingKey("late"): {Kind: boardcatalog.ValueCount, Number: decimal.NewFromInt(1)},
	}}
	payload, err := (&board.Payload{
		Board: *shown, Place: board.PlaceConversation, Snapshot: snapshot, Author: "Dispatch Desk",
	}).Map()
	require.NoError(t, err)
	id := pulid.MustNew("art_")
	f.artifacts.byID[id] = &assistantartifact.Artifact{
		ID: id, ThreadID: f.thread, Kind: assistantartifact.KindBoard, Title: shown.Title,
		Payload: payload, CreatedAt: now.Unix() - 600,
	}

	card, err := f.svc.Card(t.Context(), &assistantports.BoardCardRequest{
		Thread:     conversationports.GetThreadRequest{ID: f.thread},
		ArtifactID: id,
		Actor:      f.actor,
	})

	require.NoError(t, err)
	assert.Equal(t, board.PlaceConversation, card.Filled.Place)
	require.Len(t, card.Filled.Readings, 1)
	reading := card.Filled.Readings[0]
	assert.Equal(t, "0", board.FormatValue(reading.Value))
	assert.Equal(t, "1", board.FormatValue(reading.Was))
	assert.Equal(t, board.FilledSegment{Text: "1", Reading: "late"}, card.Filled.Verdict[1])
	assert.Contains(t, card.Markdown, "There are 1 (now 0) late loads, on the Dallas lanes [1].")
}

func TestFillReadsAReadingsMetricLikeAFigures(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shown := parsed(t, map[string]any{
		"title": "The day",
		"readings": []any{
			map[string]any{"key": "active", "label": "Active", "metric": "activeShipments"},
		},
		"verdict": "There are {active} rolling, [steady today][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{"label": "On time", "metric": "onTimePercent"},
		}}},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	assert.Equal(t, 1, f.dispatch.calls, "a reading and a figure of one metric are one read")
	assert.Equal(t, "41", board.FormatValue(filled.Readings[0].Value))
	assert.Equal(t, "41", filled.Verdict[1].Text)
}

func TestCheckRefusesAReadingOfRecordsTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment := pulid.MustNew("shp_")
	args := readingBoard(shipment)
	args["readings"] = []any{map[string]any{
		"key": "late", "label": "Owed", "records": []any{pulid.MustNew("inv_").String()},
		"measure": "amountDue", "aggregate": "sum",
	}}
	shown := parsed(t, args)

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Contains(
		t,
		mistake.Reason,
		"may not read invoices, so leave out the blocks and readings",
	)
}

func TestCheckRefusesAReadingOfAFigureTheAnalysisNeverShowed(t *testing.T) {
	t.Parallel()

	f := newFixture()
	id := f.analysis(f.thread, &analysis.Display{Figures: []analysis.Figure{{}}})
	shipment := pulid.MustNew("shp_")
	args := readingBoard(shipment)
	args["readings"] = []any{map[string]any{
		"key": "late", "label": "Late", "analysis": id.String(), "figure": 2.0,
	}}
	shown := parsed(t, args)

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Equal(t, "the reading late asks for figure 2 of an analysis that showed 1",
		mistake.Reason)
}
