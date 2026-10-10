package detentionservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hourlySnapshot() *detention.PolicySnapshot {
	return &detention.PolicySnapshot{
		PolicyID:                pulid.MustNew("dtp_"),
		PolicyCode:              "STD-2HR",
		PolicyName:              "Standard 2 Hour Free",
		ClockStartBasis:         detention.ClockStartBasisArrival,
		LateArrivalRule:         detention.LateArrivalRuleNoEffect,
		FreeMinutes:             120,
		PayFreeMinutes:          120,
		BillingIncrementMinutes: 15,
		RoundingMode:            detention.RoundingModeUp,
		RateSource:              detention.RateSourceAccessorial,
		AccessorialCode:         "DET",
		FlatRate:                decimal.RequireFromString("75.00"),
		FlatRateUnit:            detention.TierRateUnitHour,
		DayBoundaryMode:         detention.CapScopePerStop,
		NotificationRequirement: detention.NotificationRequirementNone,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
		Currency:                "USD",
	}
}

func TestPriceShipmentClocks_PricesARunningClockAsOfNow(t *testing.T) {
	t.Parallel()

	arrived := int64(1767225600)
	now := arrived + 10*3600
	shipmentID := pulid.MustNew("shp_")

	running := &detention.DetentionOccurrence{
		ID:              pulid.MustNew("dto_"),
		ShipmentID:      shipmentID,
		PolicySnapshot:  hourlySnapshot(),
		StopType:        shipment.StopTypeDelivery,
		ScheduleType:    shipment.StopScheduleTypeAppointment,
		ArrivedAt:       &arrived,
		ClockStartAt:    arrived,
		IsOpen:          true,
		Status:          detention.OccurrenceStatusAccruing,
		BillableMinutes: 15,
		BillableAmount:  decimal.RequireFromString("18.75"),
	}
	departed := arrived + 3*3600
	stopped := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		ShipmentID:     shipmentID,
		PolicySnapshot: hourlySnapshot(),
		StopType:       shipment.StopTypePickup,
		ScheduleType:   shipment.StopScheduleTypeAppointment,
		ArrivedAt:      &arrived,
		DepartedAt:     &departed,
		ClockStartAt:   arrived - 3600,
		Status:         detention.OccurrenceStatusAccruing,
	}

	priced := map[pulid.ID]ComputeResult{}
	priceShipmentClocks(
		[]*detention.DetentionOccurrence{running, nil, stopped}, now, time.UTC, priced,
	)

	require.Contains(t, priced, running.ID)
	assert.NotContains(t, priced, stopped.ID, "a stopped clock keeps its own figure")
	assert.Equal(t, int32(480), priced[running.ID].BillableMinutes)
	assert.True(t, priced[running.ID].BillableAmount.Equal(decimal.RequireFromString("600")),
		priced[running.ID].BillableAmount.String())
}

func TestPriceShipmentClocks_LeavesAFrozenChargeAlone(t *testing.T) {
	t.Parallel()

	arrived := int64(1767225600)
	frozen := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		PolicySnapshot: hourlySnapshot(),
		ArrivedAt:      &arrived,
		IsOpen:         true,
		Status:         detention.OccurrenceStatusBilled,
		BillableAmount: decimal.RequireFromString("75"),
	}
	require.True(t, frozen.IsFrozen())

	priced := map[pulid.ID]ComputeResult{}
	priceShipmentClocks([]*detention.DetentionOccurrence{frozen}, arrived+10*3600, time.UTC, priced)

	assert.Empty(t, priced)
}
