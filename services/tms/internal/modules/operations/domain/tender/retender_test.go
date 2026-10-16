package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type offerSpec struct {
	carrier string
	rank    int16
	status  tender.OfferStatus
	sent    bool
	reason  string
	from    *pulid.ID
}

func buildTender(status tender.Status, specs ...offerSpec) (*tender.Tender, map[string]pulid.ID) {
	carriers := make(map[string]pulid.ID)
	entity := &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Mode:           tender.ModeWaterfall,
		Status:         status,
	}
	sentAt := int64(1_800_000_000)
	for _, spec := range specs {
		carrierID, ok := carriers[spec.carrier]
		if !ok {
			carrierID = pulid.MustNew("car_")
			carriers[spec.carrier] = carrierID
		}
		offer := &tender.TenderOffer{
			ID:              pulid.MustNew("tof_"),
			OrganizationID:  entity.OrganizationID,
			BusinessUnitID:  entity.BusinessUnitID,
			TenderID:        entity.ID,
			CarrierID:       carrierID,
			Rank:            spec.rank,
			Rate:            decimal.NewFromInt(int64(1000 + spec.rank)),
			OfferTTLSeconds: 3600,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  spec.carrier + "@carrier.example",
			Status:          spec.status,
			DeclineReason:   spec.reason,
			ReofferedFromID: spec.from,
		}
		if spec.sent {
			offer.SentAt = &sentAt
		}
		entity.Offers = append(entity.Offers, offer)
		if spec.status == tender.OfferStatusAccepted {
			id := offer.ID
			entity.AcceptedOfferID = &id
		}
	}
	return entity, carriers
}

func TestBounceReoffersTheRanksTheAcceptanceSkipped(t *testing.T) {
	entity, carriers := buildTender(tender.StatusAccepted,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusDeclined, sent: true},
		offerSpec{carrier: "b", rank: 2, status: tender.OfferStatusAccepted, sent: true},
		offerSpec{carrier: "c", rank: 3, status: tender.OfferStatusSkipped},
		offerSpec{carrier: "d", rank: 4, status: tender.OfferStatusSkipped},
	)
	revoked, ok := entity.CanBounce()
	require.True(t, ok)

	plan := entity.ReofferPlan(revoked)

	require.Len(t, plan, 2)
	assert.Equal(t, carriers["c"], plan[0].CarrierID)
	assert.Equal(t, int16(3), plan[0].Rank)
	assert.Equal(t, carriers["d"], plan[1].CarrierID)
	assert.Equal(t, int16(4), plan[1].Rank)
	for _, offer := range plan {
		assert.Equal(t, tender.OfferStatusPending, offer.Status)
		assert.True(t, offer.ID.IsNil(), "a re-offer is a new row")
		assert.Nil(t, offer.SentAt)
		require.NotNil(t, offer.ReofferedFromID)
		assert.Equal(t, entity.ID, offer.TenderID)
	}
	assert.Equal(t, entity.Offers[2].ID, *plan[0].ReofferedFromID)
	assert.Equal(t, decimal.NewFromInt(1003), plan[0].Rate)
	assert.Equal(t, "c@carrier.example", plan[0].RecipientEmail)
}

func TestBounceLeavesIneligibleAndEarlierRanksAlone(t *testing.T) {
	entity, carriers := buildTender(tender.StatusAccepted,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusSkipped},
		offerSpec{carrier: "b", rank: 2, status: tender.OfferStatusExpired, sent: true},
		offerSpec{carrier: "c", rank: 3, status: tender.OfferStatusAccepted, sent: true},
		offerSpec{
			carrier: "d", rank: 4, status: tender.OfferStatusSkipped,
			reason: "Carrier became ineligible before the offer was sent: insurance lapsed",
		},
		offerSpec{carrier: "e", rank: 5, status: tender.OfferStatusSkipped},
	)
	revoked, ok := entity.CanBounce()
	require.True(t, ok)

	plan := entity.ReofferPlan(revoked)

	require.Len(t, plan, 1)
	assert.Equal(t, carriers["e"], plan[0].CarrierID)
}

func TestBroadcastBounceReoffersTheSupersededCarriers(t *testing.T) {
	entity, carriers := buildTender(tender.StatusAccepted,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusSuperseded, sent: true},
		offerSpec{carrier: "b", rank: 1, status: tender.OfferStatusAccepted, sent: true},
		offerSpec{carrier: "c", rank: 1, status: tender.OfferStatusDeclined, sent: true},
	)
	entity.Mode = tender.ModeSpotBroadcast
	revoked, ok := entity.CanBounce()
	require.True(t, ok)

	plan := entity.ReofferPlan(revoked)

	require.Len(t, plan, 1)
	assert.Equal(t, carriers["a"], plan[0].CarrierID)
}

func TestASecondBounceNeitherRepeatsReoffersNorReturnsToABouncer(t *testing.T) {
	entity, carriers := buildTender(tender.StatusAccepted,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusRevoked, sent: true},
		offerSpec{carrier: "b", rank: 2, status: tender.OfferStatusSkipped},
		offerSpec{carrier: "c", rank: 3, status: tender.OfferStatusSkipped},
	)
	firstB := entity.Offers[1].ID
	firstC := entity.Offers[2].ID
	second, _ := buildTender(
		tender.StatusAccepted,
		offerSpec{
			carrier: "x",
			rank:    2,
			status:  tender.OfferStatusAccepted,
			sent:    true,
			from:    &firstB,
		},
		offerSpec{carrier: "y", rank: 3, status: tender.OfferStatusSkipped, from: &firstC},
	)
	second.Offers[0].CarrierID = carriers["b"]
	second.Offers[1].CarrierID = carriers["c"]
	entity.Offers = append(entity.Offers, second.Offers...)
	accepted := second.Offers[0].ID
	entity.AcceptedOfferID = &accepted
	entity.BounceCount = 1

	revoked, ok := entity.CanBounce()
	require.True(t, ok)
	plan := entity.ReofferPlan(revoked)

	require.Len(t, plan, 1)
	assert.Equal(t, carriers["c"], plan[0].CarrierID)
	assert.Equal(t, second.Offers[1].ID, *plan[0].ReofferedFromID)
}

func TestOnlyAStandingAcceptanceCanBeBounced(t *testing.T) {
	active, _ := buildTender(tender.StatusActive,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusSent, sent: true},
	)
	_, ok := active.CanBounce()
	assert.False(t, ok)

	review, _ := buildTender(tender.StatusNeedsReview,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusAccepted, sent: true},
	)
	offer, ok := review.CanBounce()
	assert.True(t, ok)
	assert.Equal(t, review.Offers[0].ID, offer.ID)

	canceled, _ := buildTender(tender.StatusCanceled,
		offerSpec{carrier: "a", rank: 1, status: tender.OfferStatusAccepted, sent: true},
	)
	_, ok = canceled.CanBounce()
	assert.False(t, ok)
}

func TestAgreedRateFollowsTheCounterUnlessItWasRefused(t *testing.T) {
	offer := &tender.TenderOffer{Rate: decimal.NewFromInt(1500), Status: tender.OfferStatusSent}
	assert.True(t, decimal.NewFromInt(1500).Equal(offer.AgreedRate()))

	offer.CounterRate = decimal.NewNullDecimal(decimal.NewFromInt(1750))
	offer.Status = tender.OfferStatusAccepted
	assert.True(t, decimal.NewFromInt(1750).Equal(offer.AgreedRate()))

	offer.Status = tender.OfferStatusCounterRejected
	assert.True(t, decimal.NewFromInt(1500).Equal(offer.AgreedRate()))
}

func TestCounteredOffersAreStillOutstanding(t *testing.T) {
	assert.True(t, tender.OfferStatusCountered.IsOutstanding())
	assert.True(t, tender.OfferStatusCountered.IsAwaitingAnswer())
	assert.False(t, tender.OfferStatusPending.IsAwaitingAnswer())
	assert.True(t, tender.OfferStatusCounterRejected.IsTerminal())
	assert.True(t, tender.OfferStatusRevoked.IsTerminal())
	assert.True(t, tender.ResponseActionCounter.IsValid())
}
