package tenderescalation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEscalateOffersCapsAPerMileRateOverThePaidDistance(t *testing.T) {
	t.Parallel()

	routed, paid := 400.0, 500.0
	move := &shipment.ShipmentMove{Distance: &routed, PayDistance: &paid}
	offers := []*tender.TenderOffer{{
		RateMethod: shipmentvocab.CarrierRateMethodPerMile,
		Rate:       decimal.RequireFromString("1.70"),
	}}
	policy := &dispatchcontrol.TenderPolicy{
		CounterMarginFloor:     decimal.NewFromInt(10),
		EscalationRateIncrease: decimal.NewFromInt(10),
		EscalationOfferTTLSecs: 900,
	}

	escalateOffers(offers, decimal.NewFromInt(1000), move.PaidDistance(), policy)

	assert.Equal(t, "1.8", offers[0].Rate.String())
	assert.Equal(t, int32(900), offers[0].OfferTTLSeconds)
}
