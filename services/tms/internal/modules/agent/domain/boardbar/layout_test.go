package boardbar_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardbar"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86_400)

type segment struct {
	kind  boardbar.SegmentKind
	start float64
	width float64
}

func segments(bar *boardbar.Bar) []segment {
	out := make([]segment, 0, len(bar.Segments))
	for _, s := range bar.Segments {
		out = append(out, segment{kind: s.Kind, start: s.Start, width: s.Width})
	}

	return out
}

func assertSegments(t *testing.T, want []segment, bar *boardbar.Bar) {
	t.Helper()
	got := segments(bar)
	require.Len(t, got, len(want))
	for idx := range want {
		assert.Equal(t, want[idx].kind, got[idx].kind, "segment %d", idx)
		assert.InDelta(t, want[idx].start, got[idx].start, 0.001, "segment %d start", idx)
		assert.InDelta(t, want[idx].width, got[idx].width, 0.001, "segment %d width", idx)
	}
}

func d(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value)
}

func TestLayoutDrawsNoBarForAnAmount(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeAmount, []*boardreading.Reading{{}})
	assert.Equal(t, []*boardbar.Bar{nil}, bars)
}

func TestLayoutTrack(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeTrack, []*boardreading.Reading{
		{Track: &boardreading.TrackFacts{Behind: 6000}},
		{Track: &boardreading.TrackFacts{Behind: 3000, Held: true}},
		{Track: &boardreading.TrackFacts{Behind: 1500, Projected: true}},
		{Track: &boardreading.TrackFacts{}},
		nil,
	})

	assertSegments(t, []segment{{boardbar.SegmentUsed, 0, 1}}, bars[0])
	assert.Equal(t, boardcatalog.ToneWarn, bars[0].Segments[0].Tone)
	assertSegments(t, []segment{{boardbar.SegmentHeld, 0, 0.5}}, bars[1])
	assertSegments(t, []segment{{boardbar.SegmentProjected, 0, 0.25}}, bars[2])
	assert.Empty(t, bars[3].Segments, "on time holds the track with nothing on it")
	assert.Nil(t, bars[4])
}

func TestLayoutCountdownAgainstTheOwnersWindowAsTheMockupDrawsIt(t *testing.T) {
	t.Parallel()

	window := 30 * day
	bars := boardbar.Layout(boardcatalog.ShapeCountdown, []*boardreading.Reading{
		{Window: &boardreading.WindowFacts{Left: 3 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: 9 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: 14 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: -6 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: -2 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: -45 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: 40 * day, Span: window}},
	})

	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.1},
		{boardbar.SegmentProjected, 0.1, 0.9},
	}, bars[0])
	assert.Equal(t, boardcatalog.ToneWarn, bars[0].Segments[0].Tone, "under a week is warned")
	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.3},
		{boardbar.SegmentProjected, 0.3, 0.7},
	}, bars[1])
	assert.Empty(t, bars[1].Segments[0].Tone)
	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.4667},
		{boardbar.SegmentProjected, 0.4667, 0.5333},
	}, bars[2])
	assertSegments(t, []segment{{boardbar.SegmentBeyond, 0, 0.2}}, bars[3])
	assertSegments(t, []segment{{boardbar.SegmentBeyond, 0, 0.0667}}, bars[4])
	assertSegments(t, []segment{{boardbar.SegmentBeyond, 0, 1}}, bars[5])
	assertSegments(t, []segment{{boardbar.SegmentUsed, 0, 1}}, bars[6])
}

func TestLayoutCountdownOfTheSameValueIsNotAFullBar(t *testing.T) {
	t.Parallel()

	window := 30 * day
	bars := boardbar.Layout(boardcatalog.ShapeCountdown, []*boardreading.Reading{
		{Window: &boardreading.WindowFacts{Left: 8 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: 8 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: -8 * day, Span: window}},
		{Window: &boardreading.WindowFacts{Left: -8 * day, Span: window}},
	})

	for idx := range 2 {
		assert.InDelta(t, 0.2667, bars[idx].Segments[0].Width, 0.001)
		assert.InDelta(t, 0.2667, bars[idx+2].Segments[0].Width, 0.001)
	}
}

func TestLayoutCountdownWithNoWindowIsScaledAcrossTheBlock(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeCountdown, []*boardreading.Reading{
		{Window: &boardreading.WindowFacts{Left: 3 * day}},
		{Window: &boardreading.WindowFacts{Left: 30 * day}},
		{Window: &boardreading.WindowFacts{Left: -6 * day}},
		{Window: &boardreading.WindowFacts{Left: -2 * day}},
	})

	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.1},
		{boardbar.SegmentProjected, 0.1, 0.9},
	}, bars[0])
	assertSegments(t, []segment{{boardbar.SegmentUsed, 0, 1}}, bars[1])
	assertSegments(t, []segment{{boardbar.SegmentBeyond, 0, 1}}, bars[2])
	assertSegments(t, []segment{{boardbar.SegmentBeyond, 0, 0.3333}}, bars[3])
}

func TestLayoutLimitAsTheMockupDrawsHectorAlvarez(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeLimit, []*boardreading.Reading{
		{Limit: &boardreading.LimitFacts{Used: d(11.5), Planned: d(48.5), Bound: d(40)}},
		{Limit: &boardreading.LimitFacts{Used: d(30), Planned: d(52), Bound: d(40)}},
		{Limit: &boardreading.LimitFacts{Used: d(10), Planned: d(20), Bound: d(40)}},
	})

	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.2212},
		{boardbar.SegmentGhost, 0.2212, 0.5481},
		{boardbar.SegmentProjected, 0.7692, 0.1635},
	}, bars[0])
	require.NotNil(t, bars[0].Bound)
	assert.InDelta(t, 0.7692, *bars[0].Bound, 0.001)
	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.1923},
		{boardbar.SegmentGhost, 0.1923, 0.1923},
	}, bars[2])
}

func TestLayoutBoundedAsTheMockupDrawsPineRidge(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeBounded, []*boardreading.Reading{
		{Bounded: &boardreading.BoundedFacts{Value: d(6.3), Bound: d(6)}},
		{Bounded: &boardreading.BoundedFacts{Value: d(7), Bound: d(7)}},
		{Bounded: &boardreading.BoundedFacts{Value: d(3), Bound: d(6)}},
		{Bounded: &boardreading.BoundedFacts{Value: d(3.5), NoBound: true}},
	})

	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.8571},
		{boardbar.SegmentBeyond, 0.8571, 0.0429},
	}, bars[0])
	assert.InDelta(t, 0.8571, *bars[0].Bound, 0.001)
	assertSegments(t, []segment{{boardbar.SegmentUsed, 0, 0.4286}}, bars[2])
	assert.InDelta(t, 0.8571, *bars[2].Bound, 0.001, "short of the bound leaves a gap")
	assertSegments(t, []segment{{boardbar.SegmentUsed, 0, 0.5}}, bars[3])
	assert.InDelta(t, 0, *bars[3].Bound, 0.001)
}

func TestLayoutCentredAsTheMockupDrawsTexas(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeCentred, []*boardreading.Reading{
		{Centred: &boardreading.CentredFacts{Signed: d(-1352.40)}},
		{Centred: &boardreading.CentredFacts{Signed: d(676.20)}},
		{Centred: &boardreading.CentredFacts{Missing: true}},
	})

	assert.True(t, bars[0].Centred)
	assertSegments(t, []segment{{boardbar.SegmentCredit, 0, 0.5}}, bars[0])
	assertSegments(t, []segment{{boardbar.SegmentUsed, 0.5, 0.25}}, bars[1])
	assert.Empty(t, bars[2].Segments)
	assert.InDelta(t, 0.5, *bars[2].Bound, 0.001)
}

func TestLayoutSpread(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeSpread, []*boardreading.Reading{
		{Spread: &boardreading.SpreadFacts{Allowance: d(120), Median: d(180), Tail: d(400)}},
		{Spread: &boardreading.SpreadFacts{Allowance: d(120), Median: d(90), Tail: d(200)}},
	})

	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.3},
		{boardbar.SegmentBeyond, 0.3, 0.15},
		{boardbar.SegmentTail, 0.45, 0.55},
	}, bars[0])
	assert.InDelta(t, 0.3, *bars[0].Bound, 0.001)
	assertSegments(t, []segment{
		{boardbar.SegmentUsed, 0, 0.225},
		{boardbar.SegmentTail, 0.3, 0.2},
	}, bars[1])
}

func TestLayoutNeverDrawsASegmentThinnerThanASliver(t *testing.T) {
	t.Parallel()

	bars := boardbar.Layout(boardcatalog.ShapeTrack, []*boardreading.Reading{
		{Track: &boardreading.TrackFacts{Behind: 100_000}},
		{Track: &boardreading.TrackFacts{Behind: 1}},
	})

	assert.InDelta(t, boardbar.MinWidth, bars[1].Segments[0].Width, 0.0001)
}

func TestFromAnalysisShapesABarChartAndAPie(t *testing.T) {
	t.Parallel()

	read, form := boardbar.FromAnalysis(&analysis.Chart{
		Kind:      analysis.ChartBar,
		Title:     "Late by customer",
		X:         analysis.ChartAxis{Values: []any{"Northline", "Acme"}},
		Series:    []analysis.ChartSeries{{Name: "Late", Values: []any{4.0, nil}}},
		ValueKind: analysis.ValueCount,
	})
	assert.Equal(t, boardcatalog.FormBars, form)
	require.NotNil(t, read.Chart)
	assert.Equal(t, boardcatalog.ValueCount, read.Chart.Unit)
	assert.Equal(t, "Northline", read.Chart.X[0].Label)
	assert.Equal(t, "4", read.Chart.Series[0].Points[0].String())
	assert.Nil(t, read.Chart.Series[0].Points[1])
	assert.True(t, boardbar.Fits(form, read))

	pie, form := boardbar.FromAnalysis(&analysis.Chart{
		Kind:      analysis.ChartPie,
		X:         analysis.ChartAxis{Values: []any{"Fuel", "Tolls"}},
		Series:    []analysis.ChartSeries{{Name: "Cost", Values: []any{30.0, 10.0}}},
		ValueKind: analysis.ValueMoney,
		Currency:  "USD",
	})
	assert.Equal(t, boardcatalog.FormSplit, form)
	assert.Equal(t, "40", pie.Split.Total.Number.String())
	assert.Len(t, pie.Split.Parts, 2)
}
