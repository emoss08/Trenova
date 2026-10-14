package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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

	p1 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup, Sequence: 0}
	d1 := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusCanceled,
		Sequence: 1,
	}
	d2 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery, Sequence: 2}
	p2 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup, Sequence: 0}
	d3 := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery, Sequence: 1}

	entity := &shipment.Shipment{Moves: []*shipment.ShipmentMove{
		{Sequence: 1, Stops: []*shipment.Stop{d3, p2}},
		{Sequence: 0, Stops: []*shipment.Stop{d2, nil, p1, d1}},
		nil,
	}}

	assert.Equal(t, []*shipment.Stop{p1, d1, d2, p2, d3}, entity.OrderedStops())
	assert.Equal(t, []*shipment.Stop{p1, d2, p2, d3}, entity.RatedStops())
	assert.Equal(t, 4, entity.RatedStopCount())
	assert.Nil(t, (*shipment.Shipment)(nil).OrderedStops())
	assert.Zero(t, (*shipment.Shipment)(nil).RatedStopCount())
}

func TestShipment_RatingWeightAndPiecesPreferTheCommodityLines(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{Weight: int64Ptr(5_000), Pieces: int64Ptr(40)}
	assert.True(t, decimal.NewFromInt(5_000).Equal(entity.RatingWeight()))
	assert.EqualValues(t, 40, entity.RatingPieces())

	entity.Commodities = []*shipment.ShipmentCommodity{
		{Weight: 1_200, Pieces: 6},
		nil,
		{Weight: 1_360, Pieces: 4},
	}
	assert.True(t, decimal.NewFromInt(2_560).Equal(entity.RatingWeight()))
	assert.EqualValues(t, 10, entity.RatingPieces())

	assert.True(t, (&shipment.Shipment{}).RatingWeight().IsZero())
	assert.Zero(t, (&shipment.Shipment{}).RatingPieces())
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

	assert.True(t, decimal.NewFromInt(184).Equal(entity.RatingCubicFeet()),
		entity.RatingCubicFeet().String())
	density, ok := entity.RatingDensity()
	require.True(t, ok)
	assert.Equal(t, "4.8913", density.Round(4).String())

	entity.EnvelopeLengthFeet = floatPtr(10)
	entity.EnvelopeWidthFeet = floatPtr(8)
	entity.EnvelopeHeightFeet = floatPtr(9)
	assert.True(t, decimal.NewFromInt(720).Equal(entity.RatingCubicFeet()))

	_, ok = (&shipment.Shipment{}).RatingDensity()
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

	assert.True(t, byID.HasHazardousMaterial())
	assert.True(t, byRecord.HasHazardousMaterial())
	assert.False(t, none.HasHazardousMaterial())
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
	}, entity.FreightClasses())
	assert.Equal(t, commodity.FreightClass100, entity.PrimaryFreightClass())
	assert.Equal(t, commodity.FreightClass100, entity.HighestFreightClass())
	assert.Empty(t, (&shipment.Shipment{}).HighestFreightClass())
}

func TestShipment_RatingAsOfPrefersTheShipDateThenEntry(t *testing.T) {
	t.Parallel()

	const now int64 = 1_800_000_000
	assert.Equal(t, now, (*shipment.Shipment)(nil).RatingAsOf(now))
	assert.Equal(t, now, (&shipment.Shipment{}).RatingAsOf(now))
	assert.Equal(t, int64(1_700_000_000),
		(&shipment.Shipment{CreatedAt: 1_700_000_000}).RatingAsOf(now))
	assert.Equal(t, int64(1_750_000_000), (&shipment.Shipment{
		CreatedAt:      1_700_000_000,
		ActualShipDate: int64Ptr(1_750_000_000),
	}).RatingAsOf(now))
	assert.Equal(t, int64(1_700_000_000), (&shipment.Shipment{
		CreatedAt:      1_700_000_000,
		ActualShipDate: int64Ptr(0),
	}).RatingAsOf(now))
}
