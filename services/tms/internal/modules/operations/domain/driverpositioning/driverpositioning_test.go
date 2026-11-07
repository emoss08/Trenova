package driverpositioning_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/stretchr/testify/assert"
)

func TestLive_LastsUntilTwelveHoursAfterItsWindowWhileActive(t *testing.T) {
	t.Parallel()

	slot := int64(1_793_577_600)
	positioning := &driverpositioning.Positioning{
		SlotStart: slot,
		Status:    driverpositioning.StatusActive,
	}
	expires := slot + dispatchmarket.SlotSeconds + 12*3600

	assert.Equal(t, expires, positioning.ExpiresAt())
	assert.True(t, positioning.Live(expires-1))
	assert.False(t, positioning.Live(expires))

	for _, status := range []driverpositioning.Status{
		driverpositioning.StatusCanceled, driverpositioning.StatusReplaced,
	} {
		ended := &driverpositioning.Positioning{SlotStart: slot, Status: status}
		assert.False(t, ended.Live(slot), "%s is never live", status)
	}
	var none *driverpositioning.Positioning
	assert.False(t, none.Live(slot))
}
