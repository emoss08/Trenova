package resolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type collectionStopType string

type CollectionStop struct {
	Type                 collectionStopType
	Status               string
	Sequence             int64
	Pieces               *int64
	Weight               *int64
	ScheduledWindowStart int64
	ScheduledWindowEnd   *int64
	ActualArrival        *int64
	Location             *LaneLocation
}

type CollectionMove struct {
	Stops []CollectionStop
}

type CollectionHazmat struct {
	Name string
}

type CollectionCommodity struct {
	Name              string
	FreightClass      string
	Stackable         bool
	HazardousMaterial *CollectionHazmat
}

type CollectionShipmentCommodity struct {
	Weight     int64
	Pieces     int64
	LengthFeet *float64
	WidthFeet  *float64
	HeightFeet *float64
	Commodity  *CollectionCommodity
}

type CollectionShipment struct {
	Moves       []CollectionMove
	Commodities []CollectionShipmentCommodity
}

func ptrInt64(v int64) *int64     { return &v }
func ptrFloat(v float64) *float64 { return &v }

func TestComputeStops_ExposesEveryStopWithItsLocation(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	chicago := laneLocation("Chicago", "60601", "IL")
	chicago.Timezone = "America/Chicago"
	windowEnd := int64(1_780_720_000)
	entity := &CollectionShipment{
		Moves: []CollectionMove{
			{Stops: []CollectionStop{
				{
					Type: "Pickup", Status: "Completed", Sequence: 0,
					Pieces: ptrInt64(10), Weight: ptrInt64(4200),
					ScheduledWindowStart: 1_780_700_000, ScheduledWindowEnd: &windowEnd,
					ActualArrival: ptrInt64(1_780_709_400), Location: chicago,
				},
			}},
			{Stops: []CollectionStop{
				{Type: "Delivery", Status: "New", Sequence: 1, Location: laneLocation("Dallas", "75201", "TX")},
			}},
		},
	}

	raw, err := r.ResolveComputed(entity, "computeStops")
	require.NoError(t, err)
	stops, ok := raw.([]any)
	require.True(t, ok, "stops must be a plain slice expr can map over, got %T", raw)
	require.Len(t, stops, 2)

	first, ok := stops[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Pickup", first["type"])
	assert.Equal(t, "Completed", first["status"])
	assert.Equal(t, 0, first["sequence"])
	assert.InDelta(t, 4200, first["weight"], 0.0001)
	assert.InDelta(t, 10, first["pieces"], 0.0001)
	assert.Equal(t, "IL", first["state"])
	assert.Equal(t, "Chicago", first["city"])
	assert.Equal(t, "60601", first["zip"])
	assert.Equal(t, "America/Chicago", first["timezone"])
	assert.EqualValues(t, 1_780_700_000, first["windowStart"])
	assert.EqualValues(t, 1_780_720_000, first["windowEnd"])
	assert.EqualValues(t, 1_780_709_400, first["actualArrival"])

	second, ok := stops[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Delivery", second["type"])
	assert.Equal(t, "TX", second["state"])
	assert.Nil(t, second["weight"], "an unset weight is empty, not zero")
	assert.Nil(t, second["windowEnd"])
	assert.Nil(t, second["actualArrival"])
	assert.Equal(t, "", second["timezone"])
}

func TestComputeCommodities_ExposesClassDimensionsAndHazmat(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &CollectionShipment{
		Commodities: []CollectionShipmentCommodity{
			{
				Weight: 1200, Pieces: 4,
				LengthFeet: ptrFloat(4), WidthFeet: ptrFloat(4), HeightFeet: ptrFloat(5),
				Commodity: &CollectionCommodity{
					Name: "Batteries", FreightClass: "85", Stackable: false,
					HazardousMaterial: &CollectionHazmat{Name: "UN3480"},
				},
			},
			{
				Weight: 300, Pieces: 1,
				Commodity: &CollectionCommodity{Name: "Paper", FreightClass: "55", Stackable: true},
			},
			{Weight: 50, Pieces: 1},
		},
	}

	raw, err := r.ResolveComputed(entity, "computeCommodities")
	require.NoError(t, err)
	commodities, ok := raw.([]any)
	require.True(t, ok)
	require.Len(t, commodities, 3)

	batteries, ok := commodities[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Batteries", batteries["name"])
	assert.Equal(t, "85", batteries["freightClass"])
	assert.Equal(t, true, batteries["hazmat"])
	assert.Equal(t, false, batteries["stackable"])
	assert.InDelta(t, 1200, batteries["weight"], 0.0001)
	assert.InDelta(t, 4, batteries["pieces"], 0.0001)
	assert.InDelta(t, 320, batteries["cubicFeet"], 0.0001, "4 × 4 × 5 per piece × 4 pieces")
	assert.InDelta(t, 3.75, batteries["density"], 0.0001, "pounds per cubic foot")

	paper, ok := commodities[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, paper["hazmat"])
	assert.Nil(t, paper["cubicFeet"], "no dimensions means no cube")
	assert.Nil(t, paper["density"])

	bare, ok := commodities[2].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "", bare["name"], "a line without a commodity record still lists its weight")
	assert.Equal(t, false, bare["hazmat"])
}

func TestComputeCollections_AreEmptyWithoutData(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &CollectionShipment{}

	stops, err := r.ResolveComputed(entity, "computeStops")
	require.NoError(t, err)
	assert.Equal(t, []any{}, stops)

	commodities, err := r.ResolveComputed(entity, "computeCommodities")
	require.NoError(t, err)
	assert.Equal(t, []any{}, commodities)
}

func TestComputeDimensionRollups(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &CollectionShipment{
		Commodities: []CollectionShipmentCommodity{
			{
				Weight: 1200, Pieces: 4,
				LengthFeet: ptrFloat(4), WidthFeet: ptrFloat(4), HeightFeet: ptrFloat(5),
				Commodity: &CollectionCommodity{Name: "Batteries", FreightClass: "85"},
			},
			{
				Weight: 300, Pieces: 1,
				LengthFeet: ptrFloat(2), WidthFeet: ptrFloat(2), HeightFeet: ptrFloat(2),
				Commodity: &CollectionCommodity{Name: "Paper", FreightClass: "55"},
			},
			{
				Weight: 50, Pieces: 1,
				Commodity: &CollectionCommodity{Name: "Foam", FreightClass: "250"},
			},
		},
	}

	cube, err := r.ResolveComputed(entity, "computeTotalCubicFeet")
	require.NoError(t, err)
	assert.InDelta(t, 328, cube, 0.0001, "320 + 8; a line without dimensions adds no cube")

	density, err := r.ResolveComputed(entity, "computeDensity")
	require.NoError(t, err)
	assert.InDelta(t, 1550.0/328.0, density, 0.0001, "total weight over total cube")

	primary, err := r.ResolveComputed(entity, "computePrimaryFreightClass")
	require.NoError(t, err)
	assert.Equal(t, "85", primary, "the heaviest line's class")

	highest, err := r.ResolveComputed(entity, "computeHighestFreightClass")
	require.NoError(t, err)
	assert.Equal(t, "250", highest, "the least dense class governs a mixed shipment")
}

func TestComputeDimensionRollups_WithoutDimensionsOrClasses(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &CollectionShipment{
		Commodities: []CollectionShipmentCommodity{{Weight: 500, Pieces: 2}},
	}

	cube, err := r.ResolveComputed(entity, "computeTotalCubicFeet")
	require.NoError(t, err)
	assert.InDelta(t, 0, cube, 0.0001)

	density, err := r.ResolveComputed(entity, "computeDensity")
	require.NoError(t, err)
	assert.Nil(t, density, "no cube means no density, not a divide by zero")

	primary, err := r.ResolveComputed(entity, "computePrimaryFreightClass")
	require.NoError(t, err)
	assert.Equal(t, "", primary)
}
