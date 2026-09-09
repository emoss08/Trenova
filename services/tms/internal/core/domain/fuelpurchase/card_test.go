package fuelpurchase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCard() *fuelpurchase.FuelCard {
	return &fuelpurchase.FuelCard{
		Provider: fuelpurchase.CardProviderComdata,
		LastFour: "1234",
		Label:    "Truck 101 primary",
		Status:   fuelpurchase.CardStatusActive,
	}
}

func TestFuelCard_ValidCardPasses(t *testing.T) {
	t.Parallel()

	card := validCard()
	card.Normalize()

	multiErr := errortypes.NewMultiError()
	card.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, card.IsActive())
	assert.True(t, card.CanRecordPurchase(1_800_000_000))
}

func TestFuelCard_ValidateRejections(t *testing.T) {
	t.Parallel()

	cancelledAt := int64(1_800_000_000)

	tests := []struct {
		name   string
		mutate func(c *fuelpurchase.FuelCard)
		field  string
	}{
		{
			name:   "three-digit last four",
			mutate: func(c *fuelpurchase.FuelCard) { c.LastFour = "123" },
			field:  "lastFour",
		},
		{
			name:   "letters in last four",
			mutate: func(c *fuelpurchase.FuelCard) { c.LastFour = "12A4" },
			field:  "lastFour",
		},
		{
			name:   "missing label",
			mutate: func(c *fuelpurchase.FuelCard) { c.Label = "" },
			field:  "label",
		},
		{
			name:   "unknown provider",
			mutate: func(c *fuelpurchase.FuelCard) { c.Provider = "Fleetcor" },
			field:  "provider",
		},
		{
			name: "cancelled without a reason",
			mutate: func(c *fuelpurchase.FuelCard) {
				c.Status = fuelpurchase.CardStatusCancelled
				c.CancelledAt = &cancelledAt
			},
			field: "cancelReason",
		},
		{
			name: "cancelled with a short reason",
			mutate: func(c *fuelpurchase.FuelCard) {
				c.Status = fuelpurchase.CardStatusCancelled
				c.CancelledAt = &cancelledAt
				c.CancelReason = "lost"
			},
			field: "cancelReason",
		},
		{
			name: "cancelled without a date",
			mutate: func(c *fuelpurchase.FuelCard) {
				c.Status = fuelpurchase.CardStatusCancelled
				c.CancelReason = "Card reported lost by the driver"
			},
			field: "cancelledAt",
		},
		{
			name:   "active card with a cancellation date",
			mutate: func(c *fuelpurchase.FuelCard) { c.CancelledAt = &cancelledAt },
			field:  "cancelledAt",
		},
		{
			name:   "non-positive expiry",
			mutate: func(c *fuelpurchase.FuelCard) { zero := int64(0); c.ExpiresAt = &zero },
			field:  "expiresAt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			card := validCard()
			tt.mutate(card)

			multiErr := errortypes.NewMultiError()
			card.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestFuelCard_Cancel(t *testing.T) {
	t.Parallel()

	card := validCard()
	card.Cancel(1_800_000_000, "  Card reported stolen in Amarillo  ")

	multiErr := errortypes.NewMultiError()
	card.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, card.IsCancelled())
	assert.False(t, card.CanRecordPurchase(1_800_000_001))
	assert.Equal(t, "Card reported stolen in Amarillo", card.CancelReason)
	require.NotNil(t, card.CancelledAt)
	assert.Equal(t, int64(1_800_000_000), *card.CancelledAt)
}

func TestFuelCard_ExpiryBlocksPurchases(t *testing.T) {
	t.Parallel()

	expires := int64(1_800_000_000)
	card := validCard()
	card.ExpiresAt = &expires

	assert.True(t, card.CanRecordPurchase(expires))
	assert.False(t, card.CanRecordPurchase(expires+1))
	assert.True(t, card.IsExpiredAt(expires+1))
}
