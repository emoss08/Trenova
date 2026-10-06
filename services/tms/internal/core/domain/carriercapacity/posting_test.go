package carriercapacity

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validPosting() *Posting {
	location := pulid.MustNew("loc_")
	return &Posting{
		CarrierID:        pulid.MustNew("car_"),
		OriginLocationID: &location,
		AvailableFrom:    1_000,
		AvailableTo:      90_000,
		TruckCount:       2,
		RateMethod:       RateMethodPerMile,
		Rate:             decimal.NewNullDecimal(decimal.RequireFromString("2.35")),
		Source:           SourceManual,
	}
}

func TestPostingValidate(t *testing.T) {
	t.Parallel()

	radius := 50
	zero := 0
	state := pulid.MustNew("us_")

	tests := []struct {
		name   string
		mutate func(*Posting)
		field  string
	}{
		{name: "valid", mutate: func(*Posting) {}},
		{name: "valid with radius", mutate: func(p *Posting) { p.OriginRadiusMiles = &radius }},
		{name: "state origin", mutate: func(p *Posting) {
			p.OriginLocationID = nil
			p.OriginStateID = &state
		}},
		{
			name:   "no carrier",
			mutate: func(p *Posting) { p.CarrierID = pulid.Nil },
			field:  "carrierId",
		},
		{
			name:   "no origin",
			mutate: func(p *Posting) { p.OriginLocationID = nil },
			field:  "originLocationId",
		},
		{name: "radius without location", mutate: func(p *Posting) {
			p.OriginLocationID = nil
			p.OriginStateID = &state
			p.OriginRadiusMiles = &radius
		}, field: "originRadiusMiles"},
		{
			name:   "zero radius",
			mutate: func(p *Posting) { p.OriginRadiusMiles = &zero },
			field:  "originRadiusMiles",
		},
		{
			name:   "inverted window",
			mutate: func(p *Posting) { p.AvailableTo = 500 },
			field:  "availableTo",
		},
		{name: "window too long", mutate: func(p *Posting) {
			p.AvailableTo = p.AvailableFrom + MaxPostingWindow + 1
		}, field: "availableTo"},
		{name: "no trucks", mutate: func(p *Posting) { p.TruckCount = 0 }, field: "truckCount"},
		{
			name:   "bad method",
			mutate: func(p *Posting) { p.RateMethod = "Hourly" },
			field:  "rateMethod",
		},
		{name: "bad source", mutate: func(p *Posting) { p.Source = "Fax" }, field: "source"},
		{name: "negative rate", mutate: func(p *Posting) {
			p.Rate = decimal.NewNullDecimal(decimal.NewFromInt(-1))
		}, field: "rate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			posting := validPosting()
			tt.mutate(posting)
			multiErr := errortypes.NewMultiError()
			posting.Validate(multiErr)
			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), multiErr.Error())
				return
			}
			assert.True(t, multiErr.HasErrors())
			assert.Contains(t, multiErr.Error(), tt.field)
		})
	}
}

func TestPostingQuoteAndRate(t *testing.T) {
	t.Parallel()

	perMile := validPosting()
	assert.Equal(t, "940", perMile.QuoteFor(400).Decimal.String())
	assert.Equal(t, "2.35", perMile.RatePerMile(400).Decimal.String())
	assert.True(t, perMile.ActiveAt(5_000))
	assert.False(t, perMile.ActiveAt(100_000))

	flat := validPosting()
	flat.RateMethod = RateMethodFlat
	flat.Rate = decimal.NewNullDecimal(decimal.NewFromInt(1200))
	assert.Equal(t, "1200", flat.QuoteFor(400).Decimal.String())
	assert.Equal(t, "3", flat.RatePerMile(400).Decimal.String())
	assert.False(t, flat.RatePerMile(0).Valid)

	unpriced := validPosting()
	unpriced.Rate = decimal.NullDecimal{}
	assert.False(t, unpriced.QuoteFor(100).Valid)
	assert.False(t, unpriced.RatePerMile(100).Valid)
}
