package board_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentsSplitsReadingsOutsideCites(t *testing.T) {
	t.Parallel()

	segments := board.Segments(
		"There are {late} late loads, [four on one driver][1], the furthest {furthest} behind.",
	)

	assert.Equal(t, []board.Segment{
		{Text: "There are "},
		{Reading: "late"},
		{Text: " late loads, "},
		{Text: "four on one driver", Cite: 1},
		{Text: ", the furthest "},
		{Reading: "furthest"},
		{Text: " behind."},
	}, segments)
}

func TestSegmentsLeavesBracesThatAreNotAKeyAsWords(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []board.Segment{{Text: "Read {2late} and {} as they are."}},
		board.Segments("Read {2late} and {} as they are."))
	assert.Equal(t, []board.Segment{{Text: "There are "}, {Reading: "lateSix"}, {Text: " late."}},
		board.Segments("There are {lateSix} late."), "a key of either case is placed as written")
}

func TestIsReadingKey(t *testing.T) {
	t.Parallel()

	assert.True(t, board.IsReadingKey("late"))
	assert.True(t, board.IsReadingKey("furthest_behind_2"))
	assert.True(t, board.IsReadingKey("Late"))
	assert.True(t, board.IsReadingKey("lateSix"))
	assert.False(t, board.IsReadingKey("_late"))
	assert.False(t, board.IsReadingKey("2late"))
	assert.False(t, board.IsReadingKey("late-now"))
	assert.False(t, board.IsReadingKey("a234567890123456789012345678901234"))
}

func TestPlaceForPutsABoardInTheConversationOnlyWhenAskedAndSmall(t *testing.T) {
	t.Parallel()

	assert.Equal(t, board.PlaceBeside, board.PlaceFor("", 2), "beside is the default")
	assert.Equal(t, board.PlaceConversation, board.PlaceFor(board.PlaceConversation, 4))
	assert.Equal(t, board.PlaceBeside, board.PlaceFor("sidebar", 4))
	assert.Equal(t, board.PlaceBeside, board.PlaceFor(board.PlaceConversation, 5))
	assert.Equal(t, board.PlaceBeside, board.PlaceFor("", 5))
	assert.Equal(t, board.PlaceBeside, board.PlaceFor(board.PlaceBeside, 1))
}

func TestPayloadOfReadsAStoredBoardWithNoPlaceAsBeside(t *testing.T) {
	t.Parallel()

	payload, err := board.PayloadOf(map[string]any{
		"board": map[string]any{"title": "Board", "verdict": "Words.", "blocks": []any{}},
	})

	require.NoError(t, err)
	assert.Equal(t, board.PlaceBeside, payload.Board.Place)
	assert.Equal(t, board.PlaceBeside, payload.Place)
}

func TestNewPayloadKeepsThePlaceWhereASummaryReadsIt(t *testing.T) {
	t.Parallel()

	layout := &board.Board{Title: "Board", Place: board.PlaceConversation}
	filled := &board.Filled{Blocks: []board.FilledBlock{
		{Rows: make([]board.Row, 3)},
		{Checks: make([]board.FilledCheck, 2)},
	}}
	fields, err := board.NewPayload(layout, filled, "Dispatch desk").Map()

	require.NoError(t, err)
	assert.Equal(t, []any{3.0, 2.0}, fields["sizes"], "each block's drawn rows, for a skeleton")
	assert.Equal(t, "conversation", fields["place"])
	assert.Equal(t, "conversation", fields["board"].(map[string]any)["place"])
}

func count(n int64) *boardcatalog.Value {
	return &boardcatalog.Value{Kind: boardcatalog.ValueCount, Number: decimal.NewFromInt(n)}
}

func TestCompareKeepsTheAnsweredValueInTheSentence(t *testing.T) {
	t.Parallel()

	answered := &board.Filled{
		Verdict:  []board.FilledSegment{{Text: "There are "}, {Reading: "late"}, {Text: " late."}},
		Readings: []board.FilledReading{{Key: "late", Label: "Late now", Value: count(6)}},
	}
	answered.WriteReadings()
	assert.Equal(t, "6", answered.Verdict[1].Text)
	snapshot := answered.Snapshot()
	assert.Equal(t, "6", snapshot.Values[board.ReadingKey("late")].Number.String())

	now := &board.Filled{
		Verdict:  []board.FilledSegment{{Text: "There are "}, {Reading: "late"}, {Text: " late."}},
		Readings: []board.FilledReading{{Key: "late", Label: "Late now", Value: count(2)}},
	}
	now.WriteReadings()
	now.Compare(&snapshot)

	require.NotNil(t, now.Readings[0].Was)
	assert.Equal(t, "6", now.Readings[0].Was.Number.String())
	assert.Equal(t, "2", now.Readings[0].Value.Number.String())
	assert.Equal(t, "6", now.Verdict[1].Text, "the sentence keeps what was answered")
}

func TestCompareLeavesAnUnmovedOrUnreadReadingAlone(t *testing.T) {
	t.Parallel()

	snapshot := &board.Snapshot{Values: map[string]boardcatalog.Value{
		board.ReadingKey("late"):   *count(6),
		board.ReadingKey("locked"): *count(3),
	}}
	now := &board.Filled{
		Verdict: []board.FilledSegment{{Reading: "late"}, {Text: " and "}, {Reading: "locked"}},
		Readings: []board.FilledReading{
			{Key: "late", Value: count(6)},
			{Key: "locked"},
		},
	}
	now.Compare(snapshot)

	assert.Nil(t, now.Readings[0].Was)
	assert.Nil(t, now.Readings[1].Was)
	assert.Equal(t, "6", now.Verdict[0].Text)
	assert.Empty(t, now.Verdict[2].Text, "a reading the reader may not read writes nothing in")
}

func TestBoardCountsAReadingsSourcesAsItsOwn(t *testing.T) {
	t.Parallel()

	shipment := ids("shp_", 1)[0].(string)
	layout := &board.Board{Readings: []board.Reading{
		{Key: "late", FigureSource: board.FigureSource{
			Metric: homeboard.MetricDispatchToday,
			Figure: "movesLate",
		}},
	}}
	assert.Equal(t, []board.MetricAsk{{Metric: homeboard.MetricDispatchToday, Period: "today"}},
		layout.Metrics())

	parsed := lateBoard()
	parsed["readings"] = []any{map[string]any{
		"key": "held", "label": "Held", "records": []any{shipment}, "measure": "lateness",
	}}
	parsed["verdict"] = "There are {held} held, [behind one driver][3]."
	layout, problems := boardparse.Parse(parsed)
	require.Empty(t, problems)
	assert.Len(t, layout.Records()[recordkind.Kind(permission.ResourceShipment)], 7)
}

func TestMeasureSpecReadsAToolResultsFieldAsTheMeasureItMeans(t *testing.T) {
	t.Parallel()

	shipment := recordkind.Kind(permission.ResourceShipment)
	invoice := recordkind.Kind(permission.ResourceInvoice)
	named := func(kind recordkind.Kind, name string) string {
		spec, _ := board.Measure(name).Spec(kind)
		return spec.Name
	}
	assert.Equal(t, "revenue", named(shipment, "totalCharge"))
	assert.Equal(t, "lateness", named(shipment, "lateness"))
	assert.Equal(t, "amountDue", named(invoice, "balanceDue"))
	assert.Empty(t, named(invoice, "totalCharge"),
		"a name maps only for the kind whose field it is")
	assert.Empty(t, named(invoice, "daysPastDue"),
		"days past due is not the invoice's age, which counts from the invoice date")
	assert.Empty(t, named(recordkind.Kind(permission.ResourceCarrier), "revenue"))
}
