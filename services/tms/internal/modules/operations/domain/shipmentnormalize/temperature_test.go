package shipmentnormalize

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func degrees(value int16) *int16 {
	return &value
}

func TestTemperature_TakesTheLoadsRangeFromItsLiveStopsAndLines(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{
		TemperatureMin: degrees(20),
		TemperatureMax: degrees(25),
		Moves: []*shipment.ShipmentMove{
			{
				Status: shipmentvocab.MoveStatusNew,
				Stops: []*shipment.Stop{
					{Sequence: 0, Status: shipmentvocab.StopStatusNew},
					{
						Sequence:       1,
						Status:         shipmentvocab.StopStatusNew,
						TemperatureMin: degrees(-10),
						TemperatureMax: degrees(0),
					},
					{
						Sequence:       2,
						Status:         shipmentvocab.StopStatusCanceled,
						TemperatureMin: degrees(60),
						TemperatureMax: degrees(70),
					},
				},
			},
			{
				Status: shipmentvocab.MoveStatusCanceled,
				Stops: []*shipment.Stop{{
					Status:         shipmentvocab.StopStatusNew,
					TemperatureMin: degrees(80),
				}},
			},
		},
		Commodities: []*shipment.ShipmentCommodity{
			{TemperatureMin: degrees(34), TemperatureMax: degrees(38)},
			{},
		},
	}

	require.True(t, Temperature(entity))
	assert.Equal(t, degrees(-10), entity.TemperatureMin)
	assert.Equal(t, degrees(38), entity.TemperatureMax)
}

func TestTemperature_KeepsTheOverlapWhenEveryLineShares(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{
		Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
			{TemperatureMin: degrees(28), TemperatureMax: degrees(38)},
			{Sequence: 1, TemperatureMin: degrees(32), TemperatureMax: degrees(40)},
		}}},
	}

	require.True(t, Temperature(entity))
	assert.Equal(t, degrees(32), entity.TemperatureMin)
	assert.Equal(t, degrees(38), entity.TemperatureMax)
}

func TestTemperature_LeavesAHandEnteredRangeWhenNoLineHasOne(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{
		TemperatureMin: degrees(33),
		TemperatureMax: degrees(36),
		Moves:          []*shipment.ShipmentMove{{Stops: []*shipment.Stop{{}, {Sequence: 1}}}},
		Commodities:    []*shipment.ShipmentCommodity{{}},
	}

	require.False(t, Temperature(entity))
	assert.Equal(t, degrees(33), entity.TemperatureMin)
	assert.Equal(t, degrees(36), entity.TemperatureMax)
	require.False(t, Temperature(nil))
}
