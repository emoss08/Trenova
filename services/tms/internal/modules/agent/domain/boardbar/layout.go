package boardbar

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/shopspring/decimal"
)

type SegmentKind string

const (
	SegmentUsed      = SegmentKind("used")
	SegmentBeyond    = SegmentKind("beyond")
	SegmentGhost     = SegmentKind("ghost")
	SegmentProjected = SegmentKind("projected")
	SegmentTail      = SegmentKind("tail")
	SegmentCredit    = SegmentKind("credit")
	SegmentHeld      = SegmentKind("held")
)

const (
	MinWidth       = 0.015
	centre         = 0.5
	warnWithin     = 7 * 24 * 60 * 60
	fractionDigits = 4
)

type Segment struct {
	Kind  SegmentKind       `json:"kind"`
	Start float64           `json:"start"`
	Width float64           `json:"width"`
	Tone  boardcatalog.Tone `json:"tone,omitempty"`
}

type Bar struct {
	Centred  bool      `json:"centred,omitempty"`
	Segments []Segment `json:"segments"`
	Bound    *float64  `json:"bound,omitempty"`
}

func Layout(shape boardcatalog.Shape, readings []*boardreading.Reading) []*Bar {
	bars := make([]*Bar, len(readings))
	if !shape.Bar() {
		return bars
	}
	switch shape {
	case boardcatalog.ShapeTrack:
		layTrack(readings, bars)
	case boardcatalog.ShapeCountdown:
		layCountdown(readings, bars)
	case boardcatalog.ShapeLimit:
		layLimit(readings, bars)
	case boardcatalog.ShapeBounded:
		layBounded(readings, bars)
	case boardcatalog.ShapeCentred:
		layCentred(readings, bars)
	case boardcatalog.ShapeSpread:
		laySpread(readings, bars)
	case boardcatalog.ShapeAmount:
	}

	return bars
}

func round(value float64) float64 {
	scale := math.Pow10(fractionDigits)

	return math.Round(value*scale) / scale
}

func fraction(value, scale float64) float64 {
	if scale <= 0 || value <= 0 {
		return 0
	}

	return math.Min(1, value/scale)
}

func (b *Bar) add(kind SegmentKind, start, width float64, tone boardcatalog.Tone) {
	if width <= 0 {
		return
	}
	width = math.Max(width, MinWidth)
	start = math.Max(0, start)
	if start+width > 1 {
		start = math.Max(0, 1-width)
	}
	b.Segments = append(b.Segments, Segment{
		Kind:  kind,
		Start: round(start),
		Width: round(width),
		Tone:  tone,
	})
}

func (b *Bar) mark(at float64) {
	at = round(math.Max(0, math.Min(1, at)))
	b.Bound = &at
}

func float(value decimal.Decimal) float64 {
	return value.InexactFloat64()
}

func layTrack(readings []*boardreading.Reading, bars []*Bar) {
	scale := 0.0
	for _, reading := range readings {
		if reading != nil && reading.Track != nil {
			scale = math.Max(scale, float64(reading.Track.Behind))
		}
	}
	for idx, reading := range readings {
		if reading == nil || reading.Track == nil {
			continue
		}
		bar := &Bar{Segments: []Segment{}}
		width := fraction(float64(reading.Track.Behind), scale)
		switch {
		case reading.Track.Held:
			bar.add(SegmentHeld, 0, width, "")
		case reading.Track.Projected:
			bar.add(SegmentProjected, 0, width, boardcatalog.ToneWarn)
		default:
			bar.add(SegmentUsed, 0, width, boardcatalog.ToneWarn)
		}
		bars[idx] = bar
	}
}

func layCountdown(readings []*boardreading.Reading, bars []*Bar) {
	horizon, lapse := 0.0, 0.0
	for _, reading := range readings {
		if reading == nil || reading.Window == nil || reading.Window.Span > 0 {
			continue
		}
		if reading.Window.Lapsed() {
			lapse = math.Max(lapse, float64(-reading.Window.Left))
			continue
		}
		horizon = math.Max(horizon, float64(reading.Window.Left))
	}
	for idx, reading := range readings {
		if reading == nil || reading.Window == nil {
			continue
		}
		window := reading.Window
		bar := &Bar{Segments: []Segment{}}
		if window.Lapsed() {
			scale := lapse
			if window.Span > 0 {
				scale = float64(window.Span)
			}
			bar.add(SegmentBeyond, 0, fraction(float64(-window.Left), scale), boardcatalog.ToneBad)
			bars[idx] = bar
			continue
		}
		scale := horizon
		if window.Span > 0 {
			scale = float64(window.Span)
		}
		left := float64(window.Left)
		used := fraction(left, scale)
		var tone boardcatalog.Tone
		if left < warnWithin {
			tone = boardcatalog.ToneWarn
		}
		bar.add(SegmentUsed, 0, used, tone)
		if used < 1 {
			start := 0.0
			if len(bar.Segments) > 0 {
				start = bar.Segments[0].Width
			}
			bar.add(SegmentProjected, start, 1-start, "")
		}
		bars[idx] = bar
	}
}

func layLimit(readings []*boardreading.Reading, bars []*Bar) {
	scale := 0.0
	for _, reading := range readings {
		if reading != nil && reading.Limit != nil {
			scale = math.Max(
				scale,
				math.Max(float(reading.Limit.Planned), float(reading.Limit.Bound)),
			)
			scale = math.Max(scale, float(reading.Limit.Used))
		}
	}
	for idx, reading := range readings {
		if reading == nil || reading.Limit == nil {
			continue
		}
		facts := reading.Limit
		used := fraction(float(facts.Used), scale)
		planned := fraction(float(facts.Planned), scale)
		bound := fraction(float(facts.Bound), scale)
		bar := &Bar{Segments: []Segment{}}
		var tone boardcatalog.Tone
		if used > bound && bound > 0 {
			tone = boardcatalog.ToneWarn
		}
		bar.add(SegmentUsed, 0, used, tone)
		if ghostEnd := math.Min(planned, bound); ghostEnd > used {
			bar.add(SegmentGhost, used, ghostEnd-used, "")
		}
		if planned > bound && bound > 0 {
			start := math.Max(bound, used)
			bar.add(SegmentProjected, start, planned-start, boardcatalog.ToneWarn)
		}
		if bound > 0 {
			bar.mark(bound)
		}
		bars[idx] = bar
	}
}

func layBounded(readings []*boardreading.Reading, bars []*Bar) {
	scale := 0.0
	for _, reading := range readings {
		if reading != nil && reading.Bounded != nil {
			scale = math.Max(
				scale,
				math.Max(float(reading.Bounded.Value), float(reading.Bounded.Bound)),
			)
		}
	}
	for idx, reading := range readings {
		if reading == nil || reading.Bounded == nil {
			continue
		}
		facts := reading.Bounded
		value := fraction(float(facts.Value), scale)
		bar := &Bar{Segments: []Segment{}}
		if facts.NoBound {
			bar.add(SegmentUsed, 0, value, "")
			bar.mark(0)
			bars[idx] = bar
			continue
		}
		bound := fraction(float(facts.Bound), scale)
		bar.add(SegmentUsed, 0, math.Min(value, bound), "")
		if value > bound {
			bar.add(SegmentBeyond, bound, value-bound, boardcatalog.ToneWarn)
		}
		bar.mark(bound)
		bars[idx] = bar
	}
}

func layCentred(readings []*boardreading.Reading, bars []*Bar) {
	scale := 0.0
	for _, reading := range readings {
		if reading != nil && reading.Centred != nil && !reading.Centred.Missing {
			scale = math.Max(scale, math.Abs(float(reading.Centred.Signed)))
		}
	}
	for idx, reading := range readings {
		if reading == nil || reading.Centred == nil {
			continue
		}
		bar := &Bar{Centred: true, Segments: []Segment{}}
		bar.mark(centre)
		if !reading.Centred.Missing {
			signed := float(reading.Centred.Signed)
			width := centre * fraction(math.Abs(signed), scale)
			switch {
			case signed > 0:
				bar.add(SegmentUsed, centre, width, "")
			case signed < 0:
				bar.add(SegmentCredit, centre-math.Max(width, MinWidth), width, "")
			}
		}
		bars[idx] = bar
	}
}

func laySpread(readings []*boardreading.Reading, bars []*Bar) {
	scale := 0.0
	for _, reading := range readings {
		if reading != nil && reading.Spread != nil {
			scale = math.Max(scale, float(reading.Spread.Tail))
		}
	}
	for idx, reading := range readings {
		if reading == nil || reading.Spread == nil {
			continue
		}
		facts := reading.Spread
		allowance := fraction(float(facts.Allowance), scale)
		median := fraction(float(facts.Median), scale)
		tail := fraction(float(facts.Tail), scale)
		bar := &Bar{Segments: []Segment{}}
		bar.add(SegmentUsed, 0, math.Min(allowance, median), "")
		if median > allowance {
			bar.add(SegmentBeyond, allowance, median-allowance, boardcatalog.ToneWarn)
		}
		if far := math.Max(median, allowance); tail > far {
			bar.add(SegmentTail, far, tail-far, "")
		}
		bar.mark(allowance)
		bars[idx] = bar
	}
}
