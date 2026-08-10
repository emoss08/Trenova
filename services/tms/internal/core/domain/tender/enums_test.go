package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestEnumValidity(t *testing.T) {
	assert.True(t, tender.ModeWaterfall.IsValid())
	assert.True(t, tender.ModeSpotBroadcast.IsValid())
	assert.True(t, tender.ModeSpotSequential.IsValid())
	assert.False(t, tender.Mode("Bogus").IsValid())

	assert.True(t, tender.ModeWaterfall.IsSequential())
	assert.True(t, tender.ModeSpotSequential.IsSequential())
	assert.False(t, tender.ModeSpotBroadcast.IsSequential())

	for _, s := range []tender.Status{
		tender.StatusActive,
		tender.StatusAccepted,
		tender.StatusExhausted,
		tender.StatusCanceled,
		tender.StatusNeedsReview,
	} {
		assert.True(t, s.IsValid(), s)
	}
	assert.False(t, tender.Status("Bogus").IsValid())
	assert.False(t, tender.StatusActive.IsTerminal())
	assert.False(t, tender.StatusNeedsReview.IsTerminal())
	assert.True(t, tender.StatusAccepted.IsTerminal())
	assert.True(t, tender.StatusExhausted.IsTerminal())
	assert.True(t, tender.StatusCanceled.IsTerminal())

	for _, s := range []tender.OfferStatus{
		tender.OfferStatusPending,
		tender.OfferStatusSent,
		tender.OfferStatusAccepted,
		tender.OfferStatusDeclined,
		tender.OfferStatusExpired,
		tender.OfferStatusWithdrawn,
		tender.OfferStatusSuperseded,
		tender.OfferStatusSkipped,
		tender.OfferStatusDeliveryFailed,
	} {
		assert.True(t, s.IsValid(), s)
	}
	assert.False(t, tender.OfferStatus("Bogus").IsValid())
	assert.True(t, tender.OfferStatusPending.IsOutstanding())
	assert.True(t, tender.OfferStatusSent.IsOutstanding())
	assert.False(t, tender.OfferStatusAccepted.IsOutstanding())
	assert.True(t, tender.OfferStatusExpired.IsTerminal())

	assert.True(t, tender.ChannelEmail.IsValid())
	assert.True(t, tender.ChannelEDI.IsValid())
	assert.False(t, tender.Channel("Fax").IsValid())

	assert.True(t, tender.ResponseActionAccept.IsValid())
	assert.True(t, tender.ResponseActionDecline.IsValid())
	assert.False(t, tender.ResponseAction("Maybe").IsValid())

	assert.True(t, tender.ResponseSourceEmail.IsValid())
	assert.True(t, tender.ResponseSourceEDI.IsValid())
	assert.True(t, tender.ResponseSourceManual.IsValid())
	assert.False(t, tender.ResponseSource("Fax").IsValid())
}

func TestOfferExpiryAndTokenUsability(t *testing.T) {
	now := timeutils.NowUnix()
	past := now - 60
	future := now + 3600

	sentExpired := &tender.TenderOffer{Status: tender.OfferStatusSent, ExpiresAt: &past}
	assert.True(t, sentExpired.IsExpiredAt(now))

	sentLive := &tender.TenderOffer{Status: tender.OfferStatusSent, ExpiresAt: &future}
	assert.False(t, sentLive.IsExpiredAt(now))

	pendingNoExpiry := &tender.TenderOffer{Status: tender.OfferStatusPending}
	assert.False(t, pendingNoExpiry.IsExpiredAt(now))

	usable := &tender.TenderOfferToken{ExpiresAt: future}
	assert.True(t, usable.IsUsable(now))

	used := &tender.TenderOfferToken{ExpiresAt: future, UsedAt: &past}
	assert.False(t, used.IsUsable(now))

	revoked := &tender.TenderOfferToken{ExpiresAt: future, RevokedAt: &past}
	assert.False(t, revoked.IsUsable(now))

	expired := &tender.TenderOfferToken{ExpiresAt: past}
	assert.False(t, expired.IsUsable(now))
}

func TestTenderValidate(t *testing.T) {
	guideID := pulid.MustNew("rg_")

	validOffer := func(rank int16) *tender.TenderOffer {
		return &tender.TenderOffer{
			CarrierID:       pulid.MustNew("car_"),
			Rank:            rank,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "dispatch@example.com",
			OfferTTLSeconds: tender.DefaultOfferTTLSeconds,
		}
	}

	t.Run("valid waterfall", func(t *testing.T) {
		tnd := &tender.Tender{
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("smv_"),
			RoutingGuideID: &guideID,
			Mode:           tender.ModeWaterfall,
			Offers:         []*tender.TenderOffer{validOffer(1)},
		}
		multiErr := errortypes.NewMultiError()
		tnd.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "errors: %v", multiErr)
	})

	t.Run("waterfall requires guide", func(t *testing.T) {
		tnd := &tender.Tender{
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("smv_"),
			Mode:           tender.ModeWaterfall,
			Offers:         []*tender.TenderOffer{validOffer(1)},
		}
		multiErr := errortypes.NewMultiError()
		tnd.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("email offer requires recipient", func(t *testing.T) {
		offer := validOffer(1)
		offer.RecipientEmail = ""
		tnd := &tender.Tender{
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("smv_"),
			Mode:           tender.ModeSpotBroadcast,
			Offers:         []*tender.TenderOffer{offer},
		}
		multiErr := errortypes.NewMultiError()
		tnd.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("edi offer requires partner", func(t *testing.T) {
		offer := validOffer(1)
		offer.Channel = tender.ChannelEDI
		offer.RecipientEmail = ""
		tnd := &tender.Tender{
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("smv_"),
			Mode:           tender.ModeSpotBroadcast,
			Offers:         []*tender.TenderOffer{offer},
		}
		multiErr := errortypes.NewMultiError()
		tnd.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("find offer", func(t *testing.T) {
		offer := validOffer(1)
		offer.ID = pulid.MustNew("tof_")
		tnd := &tender.Tender{Offers: []*tender.TenderOffer{offer}}
		assert.Equal(t, offer, tnd.FindOffer(offer.ID))
		assert.Nil(t, tnd.FindOffer(pulid.MustNew("tof_")))
	})
}
