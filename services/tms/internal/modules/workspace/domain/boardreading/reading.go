package boardreading

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
)

type Summary struct {
	Label      string
	Subtitle   string
	Status     string
	StatusTone boardcatalog.Tone
	Opens      *boardcatalog.Ref
}

type Words struct {
	Text  string              `json:"text"`
	Value *boardcatalog.Value `json:"value,omitempty"`
	At    int64               `json:"at,omitempty"`
	Day   bool                `json:"day,omitempty"`
	Place string              `json:"place,omitempty"`
}

type TrackFacts struct {
	Behind    int64
	Held      bool
	Projected bool
}

type WindowFacts struct {
	Left int64
	Span int64
}

func (w *WindowFacts) Lapsed() bool {
	return w.Left < 0
}

type LimitFacts struct {
	Used    decimal.Decimal
	Planned decimal.Decimal
	Bound   decimal.Decimal
}

type BoundedFacts struct {
	Value   decimal.Decimal
	Bound   decimal.Decimal
	NoBound bool
}

type CentredFacts struct {
	Signed  decimal.Decimal
	Missing bool
}

type SpreadFacts struct {
	Allowance decimal.Decimal
	Median    decimal.Decimal
	Tail      decimal.Decimal
}

type Reading struct {
	Value    boardcatalog.Value
	Standing Words
	Fact     *Words
	Tone     boardcatalog.Tone
	Track    *TrackFacts
	Window   *WindowFacts
	Limit    *LimitFacts
	Bounded  *BoundedFacts
	Centred  *CentredFacts
	Spread   *SpreadFacts
}

func (r *Reading) Estimated() bool {
	return r.Track != nil && r.Track.Projected
}
