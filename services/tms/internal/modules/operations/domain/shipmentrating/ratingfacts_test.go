package shipmentrating_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func floatPtr(value float64) *float64 { return &value }

func int64Ptr(value int64) *int64 { return &value }

func TestShipment_OrderedStopsFollowMoveThenStopSequence(t *testing.T) {
	t.Parallel()

	p1 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipmentvocab.StopTypePickup, Sequence: 0}
	d1 := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipmentvocab.StopTypeDelivery,
		Status:   shipmentvocab.StopStatusCanceled,
		Sequence: 1,
	}
	d2 := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipmentvocab.StopTypeDelivery,
		Sequence: 2,
	}
	p2 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipmentvocab.StopTypePickup, Sequence: 0}
	d3 := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipmentvocab.StopTypeDelivery,
		Sequence: 1,
	}

	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{
		{Sequence: 1, Stops: []*shipment.Stop{d3, p2}},
		{Sequence: 0, Stops: []*shipment.Stop{d2, nil, p1, d1}},
		nil,
	}}

	assert.Equal(t, []*shipment.Stop{p1, d1, d2, p2, d3}, shipmentrating.OrderedStops(entity))
	assert.Equal(t, []*shipment.Stop{p1, d2, p2, d3}, shipmentrating.RatedStops(entity))
	assert.Equal(t, 4, shipmentrating.RatedStopCount(entity))
	assert.Nil(t, shipmentrating.OrderedStops((*shipment.Shipment)(nil)))
	assert.Zero(t, shipmentrating.RatedStopCount((*shipment.Shipment)(nil)))
}

func TestShipment_RatingWeightAndPiecesPreferTheCommodityLines(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{Weight: int64Ptr(5_000), Pieces: int64Ptr(40)}
	assert.True(t, decimal.NewFromInt(5_000).Equal(shipmentrating.Weight(entity)))
	assert.EqualValues(t, 40, shipmentrating.Pieces(entity))

	entity.Commodities = []*shipment.ShipmentCommodity{
		{Weight: 1_200, Pieces: 6},
		nil,
		{Weight: 1_360, Pieces: 4},
	}
	assert.True(t, decimal.NewFromInt(2_560).Equal(shipmentrating.Weight(entity)))
	assert.EqualValues(t, 10, shipmentrating.Pieces(entity))

	assert.True(t, shipmentrating.Weight(&shipment.Shipment{}).IsZero())
	assert.Zero(t, shipmentrating.Pieces(&shipment.Shipment{}))
}

func TestShipment_RatingCubicFeetPrefersTheEnvelope(t *testing.T) {
	t.Parallel()

	lines := []*shipment.ShipmentCommodity{
		{
			Weight:     500,
			Pieces:     2,
			LengthFeet: floatPtr(4),
			WidthFeet:  floatPtr(4),
			HeightFeet: floatPtr(5),
		},
		{
			Weight: 300,
			Pieces: 3,
			Commodity: &commodity.Commodity{
				LengthPerUnit: floatPtr(2),
				WidthPerUnit:  floatPtr(2),
				HeightPerUnit: floatPtr(2),
			},
		},
		{Weight: 100, Pieces: 1},
	}
	entity := &shipment.Shipment{Commodities: lines}

	assert.True(t, decimal.NewFromInt(184).Equal(shipmentrating.CubicFeet(entity)),
		shipmentrating.CubicFeet(entity).String())
	density, ok := shipmentrating.Density(entity)
	require.True(t, ok)
	assert.Equal(t, "4.8913", density.Round(4).String())

	entity.EnvelopeLengthFeet = floatPtr(10)
	entity.EnvelopeWidthFeet = floatPtr(8)
	entity.EnvelopeHeightFeet = floatPtr(9)
	assert.True(t, decimal.NewFromInt(720).Equal(shipmentrating.CubicFeet(entity)))

	_, ok = shipmentrating.Density(&shipment.Shipment{})
	assert.False(t, ok)
}

func TestShipment_HasHazardousMaterialByIDOrRecord(t *testing.T) {
	t.Parallel()

	byID := &shipment.Shipment{Commodities: []*shipment.ShipmentCommodity{
		{Commodity: &commodity.Commodity{HazardousMaterialID: pulid.MustNew("hm_")}},
	}}
	byRecord := &shipment.Shipment{Commodities: []*shipment.ShipmentCommodity{
		{
			Commodity: &commodity.Commodity{
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{},
			},
		},
	}}
	none := &shipment.Shipment{Commodities: []*shipment.ShipmentCommodity{
		{Commodity: &commodity.Commodity{}}, {}, nil,
	}}

	assert.True(t, shipmentrating.HasHazardousMaterial(byID))
	assert.True(t, shipmentrating.HasHazardousMaterial(byRecord))
	assert.False(t, shipmentrating.HasHazardousMaterial(none))
}

func TestShipment_FreightClasses(t *testing.T) {
	t.Parallel()

	line := func(class commodity.FreightClass, weight int64) *shipment.ShipmentCommodity {
		return &shipment.ShipmentCommodity{
			Weight:    weight,
			Commodity: &commodity.Commodity{FreightClass: class},
		}
	}
	entity := &shipment.Shipment{Commodities: []*shipment.ShipmentCommodity{
		line(commodity.FreightClass85, 200),
		line(commodity.FreightClass100, 900),
		line(commodity.FreightClass85, 50),
		line(commodity.FreightClass92_5, 300),
		{Weight: 5_000},
	}}

	assert.Equal(t, []commodity.FreightClass{
		commodity.FreightClass85, commodity.FreightClass100, commodity.FreightClass92_5,
	}, shipmentrating.FreightClasses(entity))
	assert.Equal(t, commodity.FreightClass100, shipmentrating.PrimaryFreightClass(entity))
	assert.Equal(t, commodity.FreightClass100, shipmentrating.HighestFreightClass(entity))
	assert.Empty(t, shipmentrating.HighestFreightClass(&shipment.Shipment{}))
}

func TestShipment_RatingAsOfPrefersTheShipDateThenEntry(t *testing.T) {
	t.Parallel()

	const now int64 = 1_800_000_000
	assert.Equal(t, now, shipmentrating.AsOf((*shipment.Shipment)(nil), now))
	assert.Equal(t, now, shipmentrating.AsOf(&shipment.Shipment{}, now))
	assert.Equal(t, int64(1_700_000_000),
		shipmentrating.AsOf(&shipment.Shipment{CreatedAt: 1_700_000_000}, now))
	assert.Equal(t, int64(1_750_000_000), shipmentrating.AsOf(&shipment.Shipment{
		CreatedAt:      1_700_000_000,
		ActualShipDate: int64Ptr(1_750_000_000),
	}, now))
	assert.Equal(t, int64(1_700_000_000), shipmentrating.AsOf(&shipment.Shipment{
		CreatedAt:      1_700_000_000,
		ActualShipDate: int64Ptr(0),
	}, now))
}
