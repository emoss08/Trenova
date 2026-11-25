package shipmentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var whereClock = filtercatalog.Clock{Now: 1_800_000_000, Timezone: "America/Los_Angeles"}

func appliedOptions(t *testing.T, extras map[string]any) shipmentports.ShipmentOptions {
	t.Helper()

	applied, err := applyExtras(extras, whereClock)
	require.NoError(t, err)
	var options shipmentports.ShipmentOptions
	applied.apply(&options)

	return options
}

/*
"New reefer loads for Sunbelt picking up in Los Angeles" named the city, the stop type and
the day; the window, the city and the type are one stop, so a load that delivers in Los
Angeles and picks up elsewhere that day is not one of them.
*/
func TestApplyExtras_TheStopsWindowCityAndTypeAreOnePlace(t *testing.T) {
	t.Parallel()

	options := appliedOptions(t, map[string]any{
		extraStopCity: "Los Angeles",
		extraStopType: "Pickup",
		extraStopFrom: "2027-01-15",
		extraStopTo:   "2027-01-16",
	})

	require.Len(t, options.StopPlaces, 1)
	place := options.StopPlaces[0]
	assert.Equal(t, []string{"Los Angeles"}, place.Cities)
	assert.Equal(t, []shipmentvocab.StopType{shipmentvocab.StopTypePickup}, place.Types)
	from, ok := whereClock.Day("2027-01-15")
	require.True(t, ok)
	to, ok := whereClock.Day("2027-01-16")
	require.True(t, ok)
	assert.Equal(t, from, place.WindowFrom, "the window opens at midnight in the zone")
	assert.Equal(t, whereClock.DayEnd(to), place.WindowTo, "and closes at the end of the last day")
}

func TestApplyExtras_RefusesADayItCannotRead(t *testing.T) {
	t.Parallel()

	_, err := applyExtras(map[string]any{extraStopFrom: "next tuesday"}, whereClock)
	require.ErrorContains(t, err, "stops.scheduledFrom takes a day as YYYY-MM-DD")
}

func TestApplyExtras_EquipmentTakesWordsOrTypeIDs(t *testing.T) {
	t.Parallel()

	options := appliedOptions(t, map[string]any{extraEquipment: "reefer"})
	assert.Equal(t, "reefer", options.EquipmentTypeText)
	assert.Empty(t, options.EquipmentTypeIDs)

	typeID := pulid.MustNew("et_")
	options = appliedOptions(t, map[string]any{extraEquipment: []any{typeID.String()}})
	assert.Equal(t, []pulid.ID{typeID}, options.EquipmentTypeIDs)
	assert.Empty(t, options.EquipmentTypeText)

	_, err := applyExtras(map[string]any{extraEquipment: []any{"reefer", "van"}}, whereClock)
	require.ErrorContains(t, err, "equipment takes one description")
}

func TestApplyExtras_AgesFlagsAndCommodity(t *testing.T) {
	t.Parallel()

	options := appliedOptions(t, map[string]any{
		extraStatusAgeDays: float64(3),
		extraUnassigned:    false,
		extraBillingReady:  false,
		extraCommodityName: "frozen chicken",
	})

	assert.Equal(t, whereClock.Now-3*secondsPerDayExtra, options.InStatusSince)
	require.NotNil(t, options.Unassigned)
	assert.False(t, *options.Unassigned, "false asks for the covered loads, not for no filter")
	require.NotNil(t, options.BillingReady)
	assert.False(t, *options.BillingReady)
	assert.Equal(t, "frozen chicken", options.CommodityText)

	_, err := applyExtras(map[string]any{extraStatusAgeDays: "three"}, whereClock)
	require.ErrorContains(t, err, "status.ageDays takes a number of days")
}

func TestApplyExtras_NamesAFilterItDoesNotKnow(t *testing.T) {
	t.Parallel()

	_, err := applyExtras(map[string]any{"stops.zip": "90001"}, whereClock)
	require.ErrorContains(t, err, `"stops.zip" is not a shipment filter`)
}

func TestApplyExtras_RefusesAnIDItCannotRead(t *testing.T) {
	t.Parallel()

	_, err := applyExtras(map[string]any{extraDriver: "Bob Smith"}, whereClock)
	require.ErrorContains(t, err, `driverId takes record ids, and "Bob Smith" is not one`)
}

func TestWhereExtras_DescribeEveryFilterApplyExtrasReads(t *testing.T) {
	t.Parallel()

	tool := NewShipmentListTool(nil, &PartySources{})
	for _, extra := range tool.WhereExtras() {
		_, err := applyExtras(map[string]any{extra.Name: sampleExtraValue(extra.Kind)}, whereClock)
		assert.NoError(t, err, extra.Name)
	}
}

func sampleExtraValue(kind string) any {
	switch kind {
	case "boolean":
		return true
	case "number":
		return float64(2)
	case "date":
		return "2027-01-15"
	case "enum":
		return "Pickup"
	case "id":
		return pulid.MustNew("wrk_").String()
	default:
		return "x"
	}
}
