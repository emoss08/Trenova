package resolver_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var computedFunctions = []string{
	"computeTotalDistance",
	"computeTotalStops",
	"computeHasHazmat",
	"computeRequiresTemperatureControl",
	"computeTemperatureDifferential",
	"computeTotalWeight",
	"computeTotalPieces",
	"computeTotalLinearFeet",
	"computeTotalHours",
	"computeStops",
	"computeCommodities",
	"computeTotalCubicFeet",
	"computeDensity",
	"computePrimaryFreightClass",
	"computeHighestFreightClass",
	"computeBaseRate",
	"computeFreightChargeAmount",
	"computeOtherChargeAmount",
	"computeCurrentTotalCharge",
	"computeOriginCity",
	"computeOriginState",
	"computeOriginZip",
	"computeDestinationCity",
	"computeDestinationState",
	"computeDestinationZip",
	"computePickupDayOfWeek",
	"computePickupHour",
	"computePickupMonth",
	"computeIsWeekendPickup",
	"computePickupDate",
	"computeDeliveryDate",
}

func computed(t *testing.T) *resolver.Resolver {
	t.Helper()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)
	return r
}

func compute(t *testing.T, r *resolver.Resolver, entity any, function string) any {
	t.Helper()

	value, err := r.ResolveComputed(entity, function)
	require.NoError(t, err, function)
	return value
}

func floatPtr(value float64) *float64 { return &value }

func int64Ptr(value int64) *int64 { return &value }

func int16Ptr(value int16) *int16 { return &value }

func place(city, zip, state, timezone string) *location.Location {
	loc := &location.Location{City: city, PostalCode: zip, Timezone: timezone}
	if state != "" {
		loc.State = &usstate.UsState{Abbreviation: state}
	}
	return loc
}

func TestRegisterDefaultComputed_RegistersEveryFunction(t *testing.T) {
	t.Parallel()

	r := computed(t)
	for _, name := range computedFunctions {
		_, ok := r.GetComputed(name)
		assert.True(t, ok, name)
	}
}

func TestComputed_RefusesAnythingButAShipment(t *testing.T) {
	t.Parallel()

	r := computed(t)
	for _, name := range computedFunctions {
		_, err := r.ResolveComputed(42, name)
		require.Error(t, err, name)
		assert.ErrorIs(t, err, resolver.ErrUnsupportedEntity, name)

		_, err = r.ResolveComputed((*shipment.Shipment)(nil), name)
		require.Error(t, err, name)

		_, err = r.ResolveComputed(shipment.Shipment{}, name)
		require.NoError(t, err, "a shipment value is accepted as well as a pointer: %s", name)
	}
}

func TestComputed_TotalsReadTheShipmentsOwnAggregates(t *testing.T) {
	t.Parallel()

	r := computed(t)
	entity := &shipment.Shipment{
		Weight: int64Ptr(9_999),
		Pieces: int64Ptr(999),
		Moves: []*shipment.ShipmentMove{
			{Distance: floatPtr(100.5), Stops: []*shipment.Stop{
				{Sequence: 0, Type: shipment.StopTypePickup},
				{Sequence: 1, Type: shipment.StopTypeDelivery, Status: shipment.StopStatusCanceled},
			}},
			{Sequence: 1, Distance: floatPtr(200.25), Stops: []*shipment.Stop{
				{Sequence: 0, Type: shipment.StopTypeDelivery},
			}},
			{Sequence: 2},
		},
		Commodities: []*shipment.ShipmentCommodity{
			{
				Weight:     1_000,
				Pieces:     10,
				LengthFeet: floatPtr(2),
				WidthFeet:  floatPtr(2),
				HeightFeet: floatPtr(2),
				Commodity: &commodity.Commodity{
					FreightClass:      commodity.FreightClass85,
					LinearFeetPerUnit: floatPtr(0.5),
				},
			},
			{
				Weight: 3_000,
				Pieces: 5,
				Commodity: &commodity.Commodity{
					FreightClass:        commodity.FreightClass92_5,
					HazardousMaterialID: pulid.MustNew("hm_"),
				},
			},
		},
	}

	assert.InDelta(t, 300.75, compute(t, r, entity, "computeTotalDistance"), 0.0001)
	assert.Equal(t, 2, compute(t, r, entity, "computeTotalStops"))
	assert.Equal(t, true, compute(t, r, entity, "computeHasHazmat"))
	assert.InDelta(t, 4_000.0, compute(t, r, entity, "computeTotalWeight"), 0.0001)
	assert.Equal(t, int64(15), compute(t, r, entity, "computeTotalPieces"))
	assert.InDelta(t, 5.0, compute(t, r, entity, "computeTotalLinearFeet"), 0.0001)
	assert.InDelta(t, 80.0, compute(t, r, entity, "computeTotalCubicFeet"), 0.0001)
	assert.InDelta(t, 50.0, compute(t, r, entity, "computeDensity"), 0.0001)
	assert.Equal(t, "92.5", compute(t, r, entity, "computePrimaryFreightClass"))
	assert.Equal(t, "92.5", compute(t, r, entity, "computeHighestFreightClass"))

	empty := &shipment.Shipment{}
	assert.Nil(t, compute(t, r, empty, "computeDensity"), "no cube means density is unknown")
	assert.Empty(t, compute(t, r, empty, "computePrimaryFreightClass"))
	assert.Equal(t, false, compute(t, r, empty, "computeHasHazmat"))
	assert.Equal(t, 0, compute(t, r, empty, "computeTotalStops"))
}

func TestComputed_TemperatureFields(t *testing.T) {
	t.Parallel()

	r := computed(t)
	reefer := &shipment.Shipment{TemperatureMin: int16Ptr(34), TemperatureMax: int16Ptr(40)}
	dry := &shipment.Shipment{}
	oneSided := &shipment.Shipment{TemperatureMax: int16Ptr(40)}

	assert.Equal(t, true, compute(t, r, reefer, "computeRequiresTemperatureControl"))
	assert.InDelta(t, 6.0, compute(t, r, reefer, "computeTemperatureDifferential"), 0.0001)
	assert.Equal(t, false, compute(t, r, dry, "computeRequiresTemperatureControl"))
	assert.InDelta(t, 0.0, compute(t, r, dry, "computeTemperatureDifferential"), 0.0001)
	assert.Equal(t, true, compute(t, r, oneSided, "computeRequiresTemperatureControl"))
	assert.InDelta(t, 0.0, compute(t, r, oneSided, "computeTemperatureDifferential"), 0.0001)
}

func TestComputed_ChargeAmounts(t *testing.T) {
	t.Parallel()

	r := computed(t)
	entity := &shipment.Shipment{
		BaseRate:            decimal.NewNullDecimal(decimal.RequireFromString("2.15")),
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1_000)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(1_250)),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{Method: accessorialcharge.MethodFlat, Amount: decimal.NewFromInt(75), Unit: 2},
			{Method: accessorialcharge.MethodPercentage, Amount: decimal.NewFromInt(10)},
			nil,
		},
	}

	assert.InDelta(t, 2.15, compute(t, r, entity, "computeBaseRate"), 0.0001)
	assert.InDelta(t, 1_000.0, compute(t, r, entity, "computeFreightChargeAmount"), 0.0001)
	assert.InDelta(t, 250.0, compute(t, r, entity, "computeOtherChargeAmount"), 0.0001)
	assert.InDelta(t, 1_250.0, compute(t, r, entity, "computeCurrentTotalCharge"), 0.0001)

	empty := &shipment.Shipment{}
	assert.InDelta(t, 0.0, compute(t, r, empty, "computeBaseRate"), 0.0001)
	assert.InDelta(t, 0.0, compute(t, r, empty, "computeCurrentTotalCharge"), 0.0001)
}

func TestComputed_LaneReadsTheOriginAndDestinationStops(t *testing.T) {
	t.Parallel()

	r := computed(t)
	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{
			Sequence: 2,
			Type:     shipment.StopTypeDelivery,
			Location: place("Austin", "78701", "TX", ""),
		},
		{
			Sequence: 0,
			Type:     shipment.StopTypePickup,
			Location: place("Dallas", "75201", "TX", ""),
		},
		{
			Sequence: 1,
			Type:     shipment.StopTypeDelivery,
			Location: place("Waco", "76701", "", ""),
		},
	}}}}

	assert.Equal(t, "Dallas", compute(t, r, entity, "computeOriginCity"))
	assert.Equal(t, "TX", compute(t, r, entity, "computeOriginState"))
	assert.Equal(t, "75201", compute(t, r, entity, "computeOriginZip"))
	assert.Equal(t, "Austin", compute(t, r, entity, "computeDestinationCity"))
	assert.Equal(t, "TX", compute(t, r, entity, "computeDestinationState"))
	assert.Equal(t, "78701", compute(t, r, entity, "computeDestinationZip"))

	unlocated := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{Type: shipment.StopTypePickup},
	}}}}
	assert.Empty(t, compute(t, r, unlocated, "computeOriginCity"))
	assert.Empty(t, compute(t, r, unlocated, "computeOriginState"))
	assert.Empty(t, compute(t, r, unlocated, "computeDestinationCity"))
}

func TestComputed_PickupIsReadOnThePickupLocationsClock(t *testing.T) {
	t.Parallel()

	r := computed(t)
	pacific, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	fridayEvening := time.Date(2026, time.October, 9, 18, 0, 0, 0, pacific)

	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{
			Type:                 shipment.StopTypePickup,
			ScheduledWindowStart: fridayEvening.Unix(),
			Location:             place("Fresno", "93650", "CA", "America/Los_Angeles"),
		},
	}}}}

	assert.Equal(t, int(time.Friday), compute(t, r, entity, "computePickupDayOfWeek"))
	assert.Equal(t, 18, compute(t, r, entity, "computePickupHour"))
	assert.Equal(t, int(time.October), compute(t, r, entity, "computePickupMonth"))
	assert.Equal(t, false, compute(t, r, entity, "computeIsWeekendPickup"))

	unknownZone := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{
			Type:                 shipment.StopTypePickup,
			ScheduledWindowStart: fridayEvening.Unix(),
			Location:             place("Fresno", "93650", "CA", "Mars/Olympus"),
		},
	}}}}
	assert.Equal(t, true, compute(t, r, unknownZone, "computeIsWeekendPickup"),
		"a zone the runtime cannot load is read in UTC, where it is already Saturday")

	arrived := fridayEvening.Add(-2 * time.Hour).Unix()
	entity.Moves[0].Stops[0].ActualArrival = &arrived
	assert.Equal(t, 16, compute(t, r, entity, "computePickupHour"),
		"an actual arrival outranks the scheduled window")
}

func TestComputed_PickupAndDeliveryDates(t *testing.T) {
	t.Parallel()

	r := computed(t)
	start := int64(1_800_000_000)
	end := start + 7_200

	assert.Nil(t, compute(t, r, &shipment.Shipment{}, "computePickupDate"))
	assert.Nil(t, compute(t, r, &shipment.Shipment{}, "computeDeliveryDate"))
	assert.Equal(t, 0, compute(t, r, &shipment.Shipment{}, "computePickupHour"))

	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{Sequence: 0, Type: shipment.StopTypePickup, ScheduledWindowStart: start},
		{Sequence: 1, Type: shipment.StopTypeDelivery, ScheduledWindowStart: end},
	}}}}

	pickup, ok := compute(t, r, entity, "computePickupDate").(time.Time)
	require.True(t, ok)
	assert.Equal(t, start, pickup.Unix())

	delivery, ok := compute(t, r, entity, "computeDeliveryDate").(time.Time)
	require.True(t, ok, "a delivery with only a window start is dated by it")
	assert.Equal(t, end, delivery.Unix())

	windowEnd := end + 3_600
	entity.Moves[0].Stops[1].ScheduledWindowEnd = &windowEnd
	delivery, _ = compute(t, r, entity, "computeDeliveryDate").(time.Time)
	assert.Equal(t, windowEnd, delivery.Unix())

	departed := windowEnd + 600
	entity.Moves[0].Stops[1].ActualDeparture = &departed
	delivery, _ = compute(t, r, entity, "computeDeliveryDate").(time.Time)
	assert.Equal(t, departed, delivery.Unix())
}

func TestComputed_TotalHoursSpansTheRatedStops(t *testing.T) {
	t.Parallel()

	r := computed(t)
	start := int64(1_800_000_000)
	pickupEnd := start + 3_600
	deliveryEnd := start + 10*3_600
	canceledEnd := start + 50*3_600

	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{
			Sequence:             0,
			Type:                 shipment.StopTypePickup,
			ScheduledWindowStart: start,
			ScheduledWindowEnd:   &pickupEnd,
		},
		{
			Sequence:             1,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusCanceled,
			ScheduledWindowStart: start + 40*3_600,
			ScheduledWindowEnd:   &canceledEnd,
		},
		{
			Sequence:             2,
			Type:                 shipment.StopTypeDelivery,
			ScheduledWindowStart: start + 9*3_600,
			ScheduledWindowEnd:   &deliveryEnd,
		},
	}}}}

	assert.InDelta(t, 10.0, compute(t, r, entity, "computeTotalHours"), 0.0001)
	assert.InDelta(t, 0.0, compute(t, r, &shipment.Shipment{}, "computeTotalHours"), 0.0001)
}

func TestComputed_StopsListEveryStopInSequence(t *testing.T) {
	t.Parallel()

	r := computed(t)
	windowEnd := int64(1_800_003_600)
	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
		{
			Sequence: 1,
			Type:     shipment.StopTypeDelivery,
			Status:   shipment.StopStatusCanceled,
			Location: place("Waco", "76701", "TX", "America/Chicago"),
		},
		{
			Sequence:             0,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			ScheduleType:         shipment.StopScheduleTypeOpen,
			Pieces:               int64Ptr(12),
			Weight:               int64Ptr(4_000),
			ScheduledWindowStart: 1_800_000_000,
			ScheduledWindowEnd:   &windowEnd,
			Location:             place("Dallas", "75201", "TX", "America/Chicago"),
		},
	}}}}

	stops, ok := compute(t, r, entity, "computeStops").([]any)
	require.True(t, ok)
	require.Len(t, stops, 2)

	first, ok := stops[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Pickup", first["type"])
	assert.Equal(t, "New", first["status"])
	assert.Equal(t, "Open", first["scheduleType"])
	assert.Equal(t, 0, first["sequence"])
	assert.Equal(t, 12.0, first["pieces"])
	assert.Equal(t, 4_000.0, first["weight"])
	assert.Equal(t, int64(1_800_000_000), first["windowStart"])
	assert.Equal(t, windowEnd, first["windowEnd"])
	assert.Nil(t, first["actualArrival"])
	assert.Equal(t, "Dallas", first["city"])
	assert.Equal(t, "TX", first["state"])
	assert.Equal(t, "75201", first["zip"])
	assert.Equal(t, "America/Chicago", first["timezone"])

	second, ok := stops[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Canceled", second["status"], "a canceled stop is listed with its status")
	assert.Nil(t, second["pieces"])
	assert.Nil(t, second["windowStart"])
}

func TestComputed_CommoditiesCarryClassAndDimensions(t *testing.T) {
	t.Parallel()

	r := computed(t)
	entity := &shipment.Shipment{Commodities: []*shipment.ShipmentCommodity{
		{
			Weight:     800,
			Pieces:     2,
			LengthFeet: floatPtr(4),
			WidthFeet:  floatPtr(4),
			HeightFeet: floatPtr(5),
			Commodity: &commodity.Commodity{
				Name:                "Paint",
				FreightClass:        commodity.FreightClass77_5,
				Stackable:           true,
				HazardousMaterialID: pulid.MustNew("hm_"),
			},
		},
		{Weight: 100, Pieces: 1},
		nil,
	}}

	lines, ok := compute(t, r, entity, "computeCommodities").([]any)
	require.True(t, ok)
	require.Len(t, lines, 2)

	paint, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Paint", paint["name"])
	assert.Equal(t, "77.5", paint["freightClass"])
	assert.Equal(t, true, paint["stackable"])
	assert.Equal(t, true, paint["hazmat"])
	assert.Equal(t, 800.0, paint["weight"])
	assert.Equal(t, 2.0, paint["pieces"])
	assert.Equal(t, 4.0, paint["lengthFeet"])
	assert.InDelta(t, 160.0, paint["cubicFeet"], 0.0001)
	assert.InDelta(t, 5.0, paint["density"], 0.0001)

	bare, ok := lines[1].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, bare["freightClass"])
	assert.Nil(t, bare["lengthFeet"])
	assert.Nil(t, bare["cubicFeet"])
	assert.Nil(t, bare["density"])
}

func TestMemoize_IsTransparent(t *testing.T) {
	t.Parallel()

	r := computed(t)
	entity := &shipment.Shipment{
		Weight: int64Ptr(500),
		Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
			{Type: shipment.StopTypePickup, Location: place("Dallas", "75201", "TX", "")},
			{Sequence: 1, Type: shipment.StopTypeDelivery},
		}}},
	}

	memo := resolver.Memoize(entity)
	assert.Same(t, memo, resolver.Memoize(memo), "wrapping twice returns the same view")
	assert.Same(t, entity, resolver.Unwrap(memo))
	assert.Same(t, entity, resolver.Unwrap(entity))

	for _, name := range computedFunctions {
		direct, directErr := r.ResolveComputed(entity, name)
		viaMemo, memoErr := r.ResolveComputed(memo, name)
		require.NoError(t, directErr, name)
		require.NoError(t, memoErr, name)
		assert.Equal(t, direct, viaMemo, name)
	}

	_, err := r.ResolveComputed(resolver.Memoize(42), "computeTotalStops")
	require.ErrorIs(t, err, resolver.ErrUnsupportedEntity)
}
