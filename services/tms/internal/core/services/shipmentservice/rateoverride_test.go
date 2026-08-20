package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An override with no amount would price the shipment at nothing without
// saying so, and a negative one is not a rate anybody negotiated.
func TestValidateRateOverrideRequest(t *testing.T) {
	t.Parallel()

	t.Run("an override needs an amount", func(t *testing.T) {
		t.Parallel()

		multiErr := validateRateOverrideRequest(&services.SetRateOverrideRequest{})

		require.NotNil(t, multiErr)
	})

	t.Run("a negative amount is refused", func(t *testing.T) {
		t.Parallel()

		multiErr := validateRateOverrideRequest(&services.SetRateOverrideRequest{
			Amount: decimal.NewNullDecimal(decimal.NewFromInt(-5)),
		})

		require.NotNil(t, multiErr)
	})

	t.Run("clearing needs no amount", func(t *testing.T) {
		t.Parallel()

		multiErr := validateRateOverrideRequest(&services.SetRateOverrideRequest{Clear: true})

		require.Nil(t, multiErr)
	})
}

// The lock is deliberately left off until after the recalculation: an override
// that arrived already locked would be frozen out by its own lock, keeping the
// old rate and recording nothing.
func TestApplyOverride_LeavesTheLockForAfterTheRecalculation(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{RateLocked: true}
	userID := pulid.MustNew("usr_")

	applyOverride(entity, &services.SetRateOverrideRequest{
		Amount:     decimal.NewNullDecimal(decimal.NewFromInt(999)),
		Reason:     "Negotiated spot rate",
		RateLocked: true,
	}, userID)

	assert.False(t, entity.RateLocked)
	assert.True(t, entity.RateOverrideAmount.Valid)
	assert.Equal(t, "Negotiated spot rate", entity.RateOverrideReason)
	require.NotNil(t, entity.RateOverrideByID)
	assert.Equal(t, userID, *entity.RateOverrideByID)
	require.NotNil(t, entity.RateOverrideAt)
}

// Clearing removes every trace of the override, so the next recalculation
// belongs to the contract again.
func TestApplyOverride_ClearRemovesEverything(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	at := int64(100)
	entity := &shipment.Shipment{
		RateOverrideAmount: decimal.NewNullDecimal(decimal.NewFromInt(999)),
		RateOverrideReason: "Negotiated spot rate",
		RateOverrideByID:   &userID,
		RateOverrideAt:     &at,
		RateLocked:         true,
	}

	applyOverride(entity, &services.SetRateOverrideRequest{Clear: true}, userID)

	assert.False(t, entity.RateOverrideAmount.Valid)
	assert.Empty(t, entity.RateOverrideReason)
	assert.Nil(t, entity.RateOverrideByID)
	assert.Nil(t, entity.RateOverrideAt)
	assert.False(t, entity.RateLocked)
}
