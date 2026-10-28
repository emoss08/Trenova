package record_test

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decode(t *testing.T, in *record.Input) map[string]any {
	t.Helper()

	if in.Rating.At.IsZero() {
		in.Rating = record.Rating{
			At:       time.Date(2026, 12, 20, 9, 0, 0, 0, time.UTC),
			Mode:     record.ModeSell,
			Currency: "USD",
		}
	}
	raw, err := record.Build(in)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &decoded))
	return decoded
}

func shipmentRoot(t *testing.T, decoded map[string]any) map[string]any {
	t.Helper()

	root, ok := decoded["shipment"].(map[string]any)
	require.True(t, ok)
	return root
}

func stopsOf(t *testing.T, decoded map[string]any) []map[string]any {
	t.Helper()

	list, ok := shipmentRoot(t, decoded)["stops"].([]any)
	require.True(t, ok)
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		stop, isStop := item.(map[string]any)
		require.True(t, isStop)
		out = append(out, stop)
	}
	return out
}

func number(value string) map[string]any {
	return map[string]any{"$n": value}
}

func quantity(value, unit string) map[string]any {
	return map[string]any{"$q": value, "u": unit}
}

func TestBuildSendsOnlyTheShipmentsOwnAccessorials(t *testing.T) {
	t.Parallel()

	lift := &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "lift"}
	inside := pulid.MustNew("acc_")
	fuel := pulid.MustNew("acc_")
	uncoded := pulid.MustNew("acc_")
	programID := pulid.MustNew("fsp_")
	agreementAccessorial := pulid.MustNew("raa_")

	entity := fullShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{AccessorialChargeID: lift.ID, AccessorialCharge: lift, Unit: 2},
		nil,
		{AccessorialChargeID: inside, Unit: 1},
		{
			AccessorialChargeID:    fuel,
			IsSystemGenerated:      true,
			FuelSurchargeProgramID: &programID,
			Unit:                   1,
		},
		{AccessorialChargeID: lift.ID, IsSystemGenerated: true, IsTariff: true, Unit: 1},
		{
			AccessorialChargeID:        lift.ID,
			IsSystemGenerated:          true,
			RateAgreementAccessorialID: &agreementAccessorial,
			Unit:                       3,
		},
		{AccessorialChargeID: lift.ID, IsTariff: true, Unit: 4},
		{AccessorialChargeID: uncoded, Unit: 1},
	}

	decoded := decode(t, &record.Input{
		Shipment:    entity,
		ChargeCodes: map[pulid.ID]string{inside: "INSIDE", fuel: "FSC"},
	})

	assert.Equal(t, []any{
		map[string]any{"code": "LIFT", "quantity": number("2")},
		map[string]any{"code": "INSIDE", "quantity": number("1")},
	}, shipmentRoot(t, decoded)["accessorials"])
}

func TestBuildSendsAnEmptyAccessorialListWithoutCharges(t *testing.T) {
	t.Parallel()

	decoded := decode(t, &record.Input{Shipment: fullShipment()})

	assert.Equal(t, []any{}, shipmentRoot(t, decoded)["accessorials"])
}

func TestBuildSendsEachStopsLocationCategory(t *testing.T) {
	t.Parallel()

	cold := &locationcategory.LocationCategory{
		Type:                locationcategory.CategoryWarehouse,
		FacilityType:        locationcategory.FacilityTypeColdStorage,
		RequiresAppointment: true,
	}
	terminal := &locationcategory.LocationCategory{Type: locationcategory.CategoryTerminal}
	entity := &shipment.Shipment{
		ProNumber: "PRO-2",
		Moves: []*shipment.ShipmentMove{{
			Stops: []*shipment.Stop{
				{
					Type:     shipment.StopTypePickup,
					Sequence: 1,
					Location: &location.Location{City: "Dallas", LocationCategory: cold},
				},
				{
					Type:     shipment.StopTypeSplitDelivery,
					Sequence: 2,
					Location: &location.Location{City: "Tulsa", LocationCategory: terminal},
				},
				{
					Type:     shipment.StopTypeSplitDelivery,
					Sequence: 3,
					Location: &location.Location{City: "Wichita"},
				},
				{Type: shipment.StopTypeDelivery, Sequence: 4},
			},
		}},
	}

	stops := stopsOf(t, decode(t, &record.Input{Shipment: entity}))
	require.Len(t, stops, 4)

	want := []map[string]any{
		{"category": "Warehouse", "facilityType": "ColdStorage", "requiresAppointment": true},
		{"category": "Terminal", "facilityType": nil, "requiresAppointment": false},
		{"category": nil, "facilityType": nil, "requiresAppointment": false},
		{"category": nil, "facilityType": nil, "requiresAppointment": false},
	}
	for i, stop := range stops {
		place, ok := stop["location"].(map[string]any)
		require.True(t, ok)
		for field, value := range want[i] {
			assert.Contains(t, place, field, "stop %d location.%s is sent", i+1, field)
			assert.Equal(t, value, place[field], "stop %d location.%s", i+1, field)
		}
	}
}

func TestBuildSendsTheDistanceFromThePreviousStop(t *testing.T) {
	t.Parallel()

	a, b, c, d := pulid.MustNew("loc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
		pulid.MustNew("loc_")
	stop := func(locationID pulid.ID, sequence int64, kind shipment.StopType) *shipment.Stop {
		return &shipment.Stop{LocationID: locationID, Sequence: sequence, Type: kind}
	}
	entity := &shipment.Shipment{
		ProNumber: "PRO-3",
		Moves: []*shipment.ShipmentMove{
			{
				Sequence: 1,
				Distance: new(120.5),
				Stops: []*shipment.Stop{
					stop(a, 1, shipment.StopTypePickup),
					stop(b, 2, shipment.StopTypeDelivery),
				},
			},
			{
				Sequence: 2,
				Distance: new(80.0),
				Stops: []*shipment.Stop{
					stop(b, 1, shipment.StopTypePickup),
					stop(c, 2, shipment.StopTypeDelivery),
				},
			},
			{
				Sequence: 3,
				Distance: new(300.0),
				Stops: []*shipment.Stop{
					stop(d, 1, shipment.StopTypePickup),
					stop(a, 2, shipment.StopTypeSplitDelivery),
					stop(b, 3, shipment.StopTypeDelivery),
				},
			},
			{
				Sequence: 4,
				Stops: []*shipment.Stop{
					stop(b, 1, shipment.StopTypePickup),
					stop(c, 2, shipment.StopTypeDelivery),
				},
			},
		},
	}

	for _, tc := range []struct {
		unit string
		want []any
	}{
		{
			unit: record.DistanceMiles,
			want: []any{
				nil,
				quantity("120.5", "mi"),
				quantity("0", "mi"),
				quantity("80", "mi"),
				nil,
				nil,
				nil,
				quantity("0", "mi"),
				nil,
			},
		},
		{
			unit: record.DistanceKilometers,
			want: []any{
				nil,
				quantity("120.5", "km"),
				quantity("0", "km"),
				quantity("80", "km"),
				nil,
				nil,
				nil,
				quantity("0", "km"),
				nil,
			},
		},
	} {
		stops := stopsOf(t, decode(t, &record.Input{Shipment: entity, DistanceUnit: tc.unit}))
		got := make([]any, 0, len(stops))
		for _, item := range stops {
			require.Contains(t, item, "legDistance")
			got = append(got, item["legDistance"])
		}
		assert.Equal(t, tc.want, got, "leg distances in %s", tc.unit)
	}
}

func TestBuildLeavesTheLegDistanceOfACanceledMoveUnknown(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{
		ProNumber: "PRO-4",
		Moves: []*shipment.ShipmentMove{{
			Status:   shipment.MoveStatusCanceled,
			Distance: new(40.0),
			Stops: []*shipment.Stop{
				{LocationID: pulid.MustNew("loc_"), Sequence: 1, Type: shipment.StopTypePickup},
				{LocationID: pulid.MustNew("loc_"), Sequence: 2, Type: shipment.StopTypeDelivery},
			},
		}},
	}

	stops := stopsOf(t, decode(t, &record.Input{Shipment: entity}))
	require.Len(t, stops, 2)
	assert.Nil(t, stops[0]["legDistance"])
	assert.Nil(t, stops[1]["legDistance"])
}

func TestBuildSendsTheHolidaysItIsGiven(t *testing.T) {
	t.Parallel()

	decoded := decode(t, &record.Input{
		Shipment: fullShipment(),
		Holidays: []record.Holiday{
			{Date: time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC), Name: "Christmas Day"},
			{Date: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), Name: "New Year's Day"},
		},
	})

	rating, ok := decoded["rating"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{
		map[string]any{"date": map[string]any{"$date": "2026-12-25"}, "name": "Christmas Day"},
		map[string]any{"date": map[string]any{"$date": "2027-01-01"}, "name": "New Year's Day"},
	}, rating["holidays"])

	empty, ok := decode(t, &record.Input{Shipment: fullShipment()})["rating"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{}, empty["holidays"])
}

func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

func TestHolidaysForExpandsRecurringHolidaysAcrossTheRatingYears(t *testing.T) {
	t.Parallel()

	rows := []*pto.OrgHoliday{
		{
			Name:           "Christmas Day",
			HolidayDate:    day(2019, time.December, 25).Unix(),
			Kind:           pto.HolidayKindHoliday,
			RecursAnnually: true,
		},
		{
			Name:        "Company picnic",
			HolidayDate: day(2026, time.July, 10).Unix(),
			Kind:        pto.HolidayKindHoliday,
		},
		{
			Name:        "Old closure",
			HolidayDate: day(2020, time.March, 3).Unix(),
			Kind:        pto.HolidayKindHoliday,
		},
		{
			Name:        "Year-end freeze",
			HolidayDate: day(2026, time.December, 28).Unix(),
			Kind:        pto.HolidayKindBlackout,
		},
		{
			Name:           "Recurring freeze",
			HolidayDate:    day(2026, time.December, 30).Unix(),
			Kind:           pto.HolidayKindBlackout,
			RecursAnnually: true,
		},
		{
			Name:           "Leap day",
			HolidayDate:    day(2024, time.February, 29).Unix(),
			Kind:           pto.HolidayKindHoliday,
			RecursAnnually: true,
		},
		nil,
	}

	assert.Equal(t, []record.Holiday{
		{Date: day(2025, time.December, 25), Name: "Christmas Day"},
		{Date: day(2026, time.July, 10), Name: "Company picnic"},
		{Date: day(2026, time.December, 25), Name: "Christmas Day"},
		{Date: day(2027, time.December, 25), Name: "Christmas Day"},
	}, record.HolidaysFor(rows, 2026))

	leap := record.HolidaysFor(rows, 2027)
	assert.Contains(t, leap, record.Holiday{Date: day(2028, time.February, 29), Name: "Leap day"})
	for _, holiday := range leap {
		assert.NotEqual(t, time.March, holiday.Date.Month(), "a leap day never moves to March")
	}
}

func TestHolidaysForSendsOneEntryForADuplicatedHoliday(t *testing.T) {
	t.Parallel()

	rows := []*pto.OrgHoliday{
		{
			Name:           "Independence Day",
			HolidayDate:    day(2020, time.July, 4).Unix(),
			Kind:           pto.HolidayKindHoliday,
			RecursAnnually: true,
		},
		{
			Name:        "Independence Day",
			HolidayDate: day(2026, time.July, 4).Unix(),
			Kind:        pto.HolidayKindHoliday,
		},
	}

	assert.Equal(t, []record.Holiday{
		{Date: day(2025, time.July, 4), Name: "Independence Day"},
		{Date: day(2026, time.July, 4), Name: "Independence Day"},
		{Date: day(2027, time.July, 4), Name: "Independence Day"},
	}, record.HolidaysFor(rows, 2026))
}

func TestHolidayWindowCoversTheYearBeforeToTheYearAfter(t *testing.T) {
	t.Parallel()

	window := record.HolidayWindowFor(2026)

	assert.Equal(t, day(2025, time.January, 1).Unix(), window.From)
	assert.Equal(t, day(2027, time.December, 31).Unix(), window.To)
}
