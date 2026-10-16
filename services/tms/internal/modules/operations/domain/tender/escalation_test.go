package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscalationRaisesTheRateByThePercentage(t *testing.T) {
	rate := tender.EscalationRate(
		decimal.NewFromInt(1500),
		decimal.NewFromInt(10),
		decimal.NewNullDecimal(decimal.NewFromInt(5000)),
	)
	assert.Equal(t, "1650", rate.String())
}

func TestEscalationNeverRaisesPastTheMarginFloor(t *testing.T) {
	ceiling := tender.RateCeiling(decimal.NewFromInt(2000), decimal.NewFromInt(10), false, nil)
	require.True(t, ceiling.Valid)
	assert.Equal(t, "1800", ceiling.Decimal.String())

	rate := tender.EscalationRate(decimal.NewFromInt(1700), decimal.NewFromInt(20), ceiling)
	assert.Equal(t, "1800", rate.String())
}

func TestEscalationNeverCutsWhatTheCarrierWasOffered(t *testing.T) {
	ceiling := tender.RateCeiling(decimal.NewFromInt(2000), decimal.NewFromInt(10), false, nil)

	rate := tender.EscalationRate(decimal.NewFromInt(1900), decimal.NewFromInt(20), ceiling)
	assert.Equal(t, "1900", rate.String())
}

func TestEscalationWithoutASellPriceKeepsTheRate(t *testing.T) {
	ceiling := tender.RateCeiling(decimal.Zero, decimal.NewFromInt(10), false, nil)
	assert.False(t, ceiling.Valid)

	rate := tender.EscalationRate(decimal.NewFromInt(1500), decimal.NewFromInt(20), ceiling)
	assert.Equal(t, "1500", rate.String())
}

func TestAPerMileCeilingIsSpreadOverTheDistance(t *testing.T) {
	distance := 500.0
	ceiling := tender.RateCeiling(decimal.NewFromInt(2000), decimal.NewFromInt(10), true, &distance)
	require.True(t, ceiling.Valid)
	assert.Equal(t, "3.6", ceiling.Decimal.String())

	assert.False(
		t,
		tender.RateCeiling(decimal.NewFromInt(2000), decimal.NewFromInt(10), true, nil).Valid,
	)
}

func TestEscalationGoesBackToEveryCarrierButABouncer(t *testing.T) {
	entity, carriers := buildTender(tender.StatusExhausted,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusDeclined, sent: true},
		offerSpec{carrier: "b", rank: 2, status: tender.OfferStatusRevoked, sent: true},
		offerSpec{carrier: "c", rank: 3, status: tender.OfferStatusExpired, sent: true},
		offerSpec{carrier: "c", rank: 3, status: tender.OfferStatusExpired, sent: true},
		offerSpec{carrier: "d", rank: 4, status: tender.OfferStatusDeliveryFailed, sent: true},
	)

	plan := entity.EscalationPlan()

	require.Len(t, plan, 3)
	assert.Equal(t, carriers["a"], plan[0].CarrierID)
	assert.Equal(t, carriers["c"], plan[1].CarrierID)
	assert.Equal(t, carriers["d"], plan[2].CarrierID)
	for idx, offer := range plan {
		assert.Equal(t, int16(idx+1), offer.Rank)
		assert.Equal(t, tender.OfferStatusPending, offer.Status)
		assert.True(t, offer.ID.IsNil())
		assert.True(t, offer.TenderID.IsNil())
		assert.Nil(t, offer.ReofferedFromID)
		assert.Equal(t, tender.ChannelEmail, offer.Channel)
	}
}
