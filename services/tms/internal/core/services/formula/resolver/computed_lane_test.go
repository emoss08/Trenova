package resolver_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type LaneState struct {
	Abbreviation string
}

type LaneLocation struct {
	City       string
	PostalCode string
	Timezone   string
	State      *LaneState
}

type LaneStop struct {
	Location             *LaneLocation
	ActualArrival        *int64
	ActualDeparture      *int64
	ScheduledWindowStart int64
	ScheduledWindowEnd   int64
}

type LaneMove struct {
	Stops []LaneStop
}

type LaneShipment struct {
	Moves []LaneMove
}

func laneResolver() *resolver.Resolver {
	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)
	return r
}

func laneLocation(city, zip, state string) *LaneLocation {
	return &LaneLocation{
		City:       city,
		PostalCode: zip,
		State:      &LaneState{Abbreviation: state},
	}
}

func TestComputeLaneVariables(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: []LaneStop{
				{Location: laneLocation("Atlanta", "30301", "GA")},
				{Location: laneLocation("Macon", "31201", "GA")},
			}},
			{Stops: []LaneStop{
				{Location: laneLocation("Orlando", "32801", "FL")},
				{Location: laneLocation("Miami", "33101", "FL")},
			}},
		},
	}

	tests := []struct {
		function string
		want     string
	}{
		{"computeOriginCity", "Atlanta"},
		{"computeOriginState", "GA"},
		{"computeOriginZip", "30301"},
		{"computeDestinationCity", "Miami"},
		{"computeDestinationState", "FL"},
		{"computeDestinationZip", "33101"},
	}

	for _, tt := range tests {
		t.Run(tt.function, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(entity, tt.function)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeLaneVariables_SkipsEmptyMoves(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: nil},
			{Stops: []LaneStop{
				{Location: laneLocation("Dallas", "75201", "TX")},
			}},
			{Stops: nil},
		},
	}

	originCity, err := r.ResolveComputed(entity, "computeOriginCity")
	require.NoError(t, err)
	assert.Equal(t, "Dallas", originCity)

	destinationZip, err := r.ResolveComputed(entity, "computeDestinationZip")
	require.NoError(t, err)
	assert.Equal(t, "75201", destinationZip)
}

func TestComputeLaneVariables_MissingData(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	tests := []struct {
		name   string
		entity *LaneShipment
	}{
		{name: "nil moves", entity: &LaneShipment{}},
		{name: "no stops", entity: &LaneShipment{Moves: []LaneMove{{Stops: nil}}}},
		{
			name: "stop without location",
			entity: &LaneShipment{
				Moves: []LaneMove{{Stops: []LaneStop{{Location: nil}}}},
			},
		},
	}

	functions := []string{
		"computeOriginCity",
		"computeOriginState",
		"computeOriginZip",
		"computeDestinationCity",
		"computeDestinationState",
		"computeDestinationZip",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, fn := range functions {
				got, err := r.ResolveComputed(tt.entity, fn)
				require.NoError(t, err, fn)
				assert.Equal(t, "", got, fn)
			}
		})
	}
}

func TestComputeLaneVariables_NilState(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: []LaneStop{
				{Location: &LaneLocation{City: "Atlanta", PostalCode: "30301", State: nil}},
			}},
		},
	}

	state, err := r.ResolveComputed(entity, "computeOriginState")
	require.NoError(t, err)
	assert.Equal(t, "", state)

	city, err := r.ResolveComputed(entity, "computeOriginCity")
	require.NoError(t, err)
	assert.Equal(t, "Atlanta", city)
}

func TestComputePickupTemporal_ActualArrival(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	arrival := time.Date(2026, time.June, 5, 14, 30, 0, 0, time.UTC).Unix()
	scheduled := time.Date(2026, time.December, 25, 8, 0, 0, 0, time.UTC).Unix()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: []LaneStop{
				{ActualArrival: &arrival, ScheduledWindowStart: scheduled},
			}},
		},
	}

	dayOfWeek, err := r.ResolveComputed(entity, "computePickupDayOfWeek")
	require.NoError(t, err)
	assert.Equal(t, int(time.Friday), dayOfWeek)

	hour, err := r.ResolveComputed(entity, "computePickupHour")
	require.NoError(t, err)
	assert.Equal(t, 14, hour)

	month, err := r.ResolveComputed(entity, "computePickupMonth")
	require.NoError(t, err)
	assert.Equal(t, int(time.June), month)

	isWeekend, err := r.ResolveComputed(entity, "computeIsWeekendPickup")
	require.NoError(t, err)
	assert.Equal(t, false, isWeekend)
}

func TestComputePickupTemporal_ScheduledFallback(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	scheduled := time.Date(2026, time.December, 25, 8, 15, 0, 0, time.UTC).Unix()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: []LaneStop{
				{ActualArrival: nil, ScheduledWindowStart: scheduled},
			}},
		},
	}

	dayOfWeek, err := r.ResolveComputed(entity, "computePickupDayOfWeek")
	require.NoError(t, err)
	assert.Equal(t, int(time.Friday), dayOfWeek)

	hour, err := r.ResolveComputed(entity, "computePickupHour")
	require.NoError(t, err)
	assert.Equal(t, 8, hour)

	month, err := r.ResolveComputed(entity, "computePickupMonth")
	require.NoError(t, err)
	assert.Equal(t, int(time.December), month)
}

func TestComputeIsWeekendPickup_Weekend(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{
			name: "saturday",
			date: time.Date(2026, time.June, 6, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "sunday",
			date: time.Date(2026, time.June, 7, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "wednesday",
			date: time.Date(2026, time.June, 3, 10, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			arrival := tt.date.Unix()
			entity := &LaneShipment{
				Moves: []LaneMove{
					{Stops: []LaneStop{{ActualArrival: &arrival}}},
				},
			}

			got, err := r.ResolveComputed(entity, "computeIsWeekendPickup")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputePickupTemporal_NoStops(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	entity := &LaneShipment{}

	for fn, want := range map[string]any{
		"computePickupDayOfWeek": 0,
		"computePickupHour":      0,
		"computePickupMonth":     0,
		"computeIsWeekendPickup": false,
	} {
		got, err := r.ResolveComputed(entity, fn)
		require.NoError(t, err, fn)
		assert.Equal(t, want, got, fn)
	}
}

func TestComputePickupTemporal_NoTimestamps(t *testing.T) {
	t.Parallel()

	r := laneResolver()

	entity := &LaneShipment{
		Moves: []LaneMove{
			{Stops: []LaneStop{{ActualArrival: nil, ScheduledWindowStart: 0}}},
		},
	}

	dayOfWeek, err := r.ResolveComputed(entity, "computePickupDayOfWeek")
	require.NoError(t, err)
	assert.Equal(t, 0, dayOfWeek)

	isWeekend, err := r.ResolveComputed(entity, "computeIsWeekendPickup")
	require.NoError(t, err)
	assert.Equal(t, false, isWeekend)
}

// Saturday 2026-06-06 01:30 UTC is Friday 2026-06-05 20:30 in Chicago. A
// pickup at that instant is a weekday evening for the shipper, and pricing
// weekend surcharges by UTC would charge them for a Saturday they never saw.
const saturdayEarlyUTC int64 = 1780709400

func TestLaneTimeVariablesUseThePickupLocationTimezone(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	arrival := saturdayEarlyUTC
	departure := saturdayEarlyUTC + 6*3600

	chicago := laneLocation("Chicago", "60601", "IL")
	chicago.Timezone = "America/Chicago"
	entity := &LaneShipment{
		Moves: []LaneMove{{Stops: []LaneStop{
			{Location: chicago, ActualArrival: &arrival},
			{Location: laneLocation("Dallas", "75201", "TX"), ActualDeparture: &departure},
		}}},
	}

	hour, err := r.ResolveComputed(entity, "computePickupHour")
	require.NoError(t, err)
	assert.Equal(t, 20, hour, "hour is local to the pickup location")

	weekend, err := r.ResolveComputed(entity, "computeIsWeekendPickup")
	require.NoError(t, err)
	assert.Equal(t, false, weekend, "Friday evening in Chicago is not a weekend pickup")

	day, err := r.ResolveComputed(entity, "computePickupDayOfWeek")
	require.NoError(t, err)
	assert.Equal(t, int(time.Friday), day)

	pickupDate, err := r.ResolveComputed(entity, "computePickupDate")
	require.NoError(t, err)
	pickup, ok := pickupDate.(time.Time)
	require.True(t, ok, "pickupDate is a real date for expr, got %T", pickupDate)
	assert.Equal(t, "America/Chicago", pickup.Location().String())
	assert.Equal(t, saturdayEarlyUTC, pickup.Unix())

	deliveryDate, err := r.ResolveComputed(entity, "computeDeliveryDate")
	require.NoError(t, err)
	delivery, ok := deliveryDate.(time.Time)
	require.True(t, ok)
	assert.Equal(t, departure, delivery.Unix())
	assert.Equal(t, "UTC", delivery.Location().String(), "a location without a timezone stays UTC")
}

func TestLaneTimeVariables_UnknownTimezoneFallsBackToUTC(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	arrival := saturdayEarlyUTC
	somewhere := laneLocation("Nowhere", "00000", "XX")
	somewhere.Timezone = "Mars/Olympus_Mons"
	entity := &LaneShipment{
		Moves: []LaneMove{{Stops: []LaneStop{{Location: somewhere, ActualArrival: &arrival}}}},
	}

	hour, err := r.ResolveComputed(entity, "computePickupHour")
	require.NoError(t, err)
	assert.Equal(t, 1, hour)

	weekend, err := r.ResolveComputed(entity, "computeIsWeekendPickup")
	require.NoError(t, err)
	assert.Equal(t, true, weekend)
}

func TestLaneDates_AreNilWithoutStops(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &LaneShipment{}

	pickupDate, err := r.ResolveComputed(entity, "computePickupDate")
	require.NoError(t, err)
	assert.Nil(t, pickupDate, "no stop means no date, not year one")

	deliveryDate, err := r.ResolveComputed(entity, "computeDeliveryDate")
	require.NoError(t, err)
	assert.Nil(t, deliveryDate)
}
