package reportfmt

import (
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// MaxBandEdges bounds an explicit boundary list. Banding exists to turn a
// continuous measure into a distribution a person can read at a glance; past a
// couple of dozen ranges a chart of the raw values says more.
const MaxBandEdges = 24

// Band groups a numeric dimension into ranges. It is one value serving two
// layers by design: the compiler groups rows on each band's lower bound — which
// keeps GROUP BY, ORDER BY and drill-through exact numerics — and the display
// contract turns that bound back into the range it stands for ("2h – 4h").
// Both layers must read the same boundaries or the label would lie about the
// rows underneath it, so the type lives here with the rest of the presentation
// vocabulary rather than being mirrored on each side.
//
// A band is either fixed-width (every range the same size, unbounded in both
// directions) or an explicit ascending boundary list, never both.
type Band struct {
	Width float64   `json:"width,omitempty"`
	Edges []float64 `json:"edges,omitempty"`
}

// IsEmpty reports that no band was authored at all. A malformed one — a
// negative width, a single boundary — is deliberately *not* empty, so it
// surfaces as a validation error instead of silently grouping nothing.
func (b *Band) IsEmpty() bool {
	return b == nil || (b.Width == 0 && len(b.Edges) == 0)
}

// IsUsable reports whether the band can be emitted, which is true exactly when
// it validates.
func (b *Band) IsUsable() bool {
	return !b.IsEmpty() && b.Validate() == nil
}

func (b *Band) IsFixedWidth() bool {
	return b != nil && b.Width > 0
}

// IsWhole reports whether every boundary is an integer, which is what a band on
// a whole-number column requires: the grouped value is cast back to that
// column's type, and a fractional boundary would be silently truncated.
func (b *Band) IsWhole() bool {
	if b == nil {
		return true
	}
	if b.Width != math.Trunc(b.Width) {
		return false
	}
	for _, edge := range b.Edges {
		if edge != math.Trunc(edge) {
			return false
		}
	}
	return true
}

func (b *Band) Validate() error {
	if b.IsEmpty() {
		return nil
	}
	if b.Width != 0 && len(b.Edges) > 0 {
		return errors.New("a band takes either a fixed width or a list of boundaries, not both")
	}

	if b.Width != 0 {
		if b.Width <= 0 || math.IsInf(b.Width, 0) || math.IsNaN(b.Width) {
			return errors.New("the band width must be a positive, finite number")
		}
		return nil
	}

	if len(b.Edges) < 2 {
		return errors.New("a banded column needs at least two boundaries")
	}
	if len(b.Edges) > MaxBandEdges {
		return fmt.Errorf("a band can carry at most %d boundaries", MaxBandEdges)
	}
	for i, edge := range b.Edges {
		if math.IsNaN(edge) || math.IsInf(edge, 0) {
			return errors.New("band boundaries must be finite numbers")
		}
		if i > 0 && edge <= b.Edges[i-1] {
			return errors.New("band boundaries must ascend, with no repeats")
		}
	}
	return nil
}

// Bounds returns the range a grouped lower bound stands for. The final explicit
// band is open-ended — "4h and over" is the honest reading of a top bucket, and
// it is also what the query did — which is reported by bounded=false.
func (b *Band) Bounds(value float64) (lower, upper float64, bounded bool) {
	if b.IsEmpty() {
		return value, 0, false
	}
	if b.IsFixedWidth() {
		return value, value + b.Width, true
	}

	index := 0
	for i := range b.Edges {
		if value >= b.Edges[i] {
			index = i
		}
	}
	if index == len(b.Edges)-1 {
		return b.Edges[index], 0, false
	}
	return b.Edges[index], b.Edges[index+1], true
}

// IsLowest reports whether a grouped value fell in the first explicit band.
// Anything below the first boundary is grouped there too, so a drill-through
// into that band must not re-apply the lower bound or it would hide the very
// rows the band counted.
func (b *Band) IsLowest(value float64) bool {
	if b.IsEmpty() || b.IsFixedWidth() {
		return false
	}
	return value <= b.Edges[0]
}

func (b *Band) clone() *Band {
	if b == nil {
		return nil
	}
	cloned := &Band{Width: b.Width}
	if len(b.Edges) > 0 {
		cloned.Edges = append([]float64(nil), b.Edges...)
	}
	return cloned
}

// bandLabel renders a grouped lower bound as the range it covers, formatting
// both ends through the column's own display contract so a banded currency
// column reads "$0 – $500" and a banded duration reads "0:00 – 2:00".
func (r *Resolved) bandLabel(value float64) string {
	lower, upper, bounded := r.Band.Bounds(value)
	if !bounded {
		return r.numeric(decimal.NewFromFloat(lower)) + "+"
	}

	// A trailing unit covers the whole range, so it is written once: "10,000 –
	// 15,000 lbs", not "10,000 lbs – 15,000 lbs". A leading symbol leads each
	// end, which is how money ranges are conventionally written.
	lead := *r
	lead.Suffix = ""

	return lead.numeric(decimal.NewFromFloat(lower)) +
		rangeSeparator +
		r.numeric(decimal.NewFromFloat(upper))
}

const rangeSeparator = " – "
