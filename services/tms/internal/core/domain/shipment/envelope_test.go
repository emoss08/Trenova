package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(v float64) *float64 { return &v }

func line(length, width, height *float64, com *commodity.Commodity) *shipment.ShipmentCommodity {
	return &shipment.ShipmentCommodity{
		LengthFeet: length,
		WidthFeet:  width,
		HeightFeet: height,
		Commodity:  com,
	}
}

func TestComputeEnvelope_TakesMaximumAcrossLines(t *testing.T) {
	t.Parallel()

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		line(ptr(20), ptr(8), ptr(6), nil),
		line(ptr(40), ptr(12.5), ptr(9), nil),
		line(ptr(10), ptr(7), ptr(11), nil),
	}, 0)

	assert.InDelta(t, 40.0, envelope.LengthFeet, 0.001)
	assert.InDelta(t, 12.5, envelope.WidthFeet, 0.001)
	assert.InDelta(t, 11.0, envelope.HeightFeet, 0.001)
}

func TestComputeEnvelope_OverallHeightAddsDeck(t *testing.T) {
	t.Parallel()

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		line(ptr(20), ptr(8), ptr(9), nil),
	}, 5)

	assert.InDelta(t, 9.0, envelope.HeightFeet, 0.001)
	assert.InDelta(t, 14.0, envelope.OverallHeightFeet, 0.001)
}

func TestComputeEnvelope_NoHeightMeansNoOverallHeight(t *testing.T) {
	t.Parallel()

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		line(ptr(20), ptr(8), nil, nil),
	}, 5)

	assert.Zero(t, envelope.OverallHeightFeet)
}

func TestComputeEnvelope_FallsBackToCommodityDefaults(t *testing.T) {
	t.Parallel()

	com := &commodity.Commodity{
		LengthPerUnit: ptr(30),
		WidthPerUnit:  ptr(10),
		HeightPerUnit: ptr(8),
	}

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		line(nil, nil, nil, com),
	}, 0)

	assert.InDelta(t, 30.0, envelope.LengthFeet, 0.001)
	assert.InDelta(t, 10.0, envelope.WidthFeet, 0.001)
	assert.InDelta(t, 8.0, envelope.HeightFeet, 0.001)
}

func TestComputeEnvelope_LineOverrideBeatsCommodityDefault(t *testing.T) {
	t.Parallel()

	com := &commodity.Commodity{
		LengthPerUnit: ptr(30),
		WidthPerUnit:  ptr(10),
		HeightPerUnit: ptr(8),
	}

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		line(ptr(45), ptr(14), ptr(9), com),
	}, 0)

	assert.InDelta(t, 45.0, envelope.LengthFeet, 0.001)
	assert.InDelta(t, 14.0, envelope.WidthFeet, 0.001)
	assert.InDelta(t, 9.0, envelope.HeightFeet, 0.001)
}

func TestComputeEnvelope_IgnoresNilLines(t *testing.T) {
	t.Parallel()

	envelope := shipment.ComputeEnvelope([]*shipment.ShipmentCommodity{
		nil,
		line(ptr(12), ptr(9), ptr(7), nil),
		nil,
	}, 0)

	assert.InDelta(t, 12.0, envelope.LengthFeet, 0.001)
}

func TestComputeEnvelope_EmptyIsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, shipment.ComputeEnvelope(nil, 5).IsZero())
}

func TestApplyEnvelope_RoundTrips(t *testing.T) {
	t.Parallel()

	sp := new(shipment.Shipment)
	sp.ApplyEnvelope(shipment.Envelope{
		LengthFeet:        40,
		WidthFeet:         12.5,
		HeightFeet:        9,
		OverallHeightFeet: 14,
	})

	require.NotNil(t, sp.EnvelopeWidthFeet)
	assert.InDelta(t, 12.5, *sp.EnvelopeWidthFeet, 0.001)
	assert.True(t, sp.HasEnvelope())

	current := sp.CurrentEnvelope()
	assert.InDelta(t, 40.0, current.LengthFeet, 0.001)
	assert.InDelta(t, 14.0, current.OverallHeightFeet, 0.001)
}

func TestApplyEnvelope_ZeroClearsStoredValues(t *testing.T) {
	t.Parallel()

	sp := new(shipment.Shipment)
	sp.ApplyEnvelope(shipment.Envelope{WidthFeet: 12})
	require.NotNil(t, sp.EnvelopeWidthFeet)

	sp.ApplyEnvelope(shipment.Envelope{})

	assert.Nil(t, sp.EnvelopeLengthFeet)
	assert.Nil(t, sp.EnvelopeWidthFeet)
	assert.Nil(t, sp.EnvelopeHeightFeet)
	assert.Nil(t, sp.EnvelopeOverallHeightFeet)
	assert.False(t, sp.HasEnvelope())
}

func TestHasDimensions_RequiresAllThree(t *testing.T) {
	t.Parallel()

	assert.True(t, line(ptr(10), ptr(8), ptr(6), nil).HasDimensions())
	assert.False(t, line(ptr(10), ptr(8), nil, nil).HasDimensions())
	assert.False(t, line(nil, ptr(8), ptr(6), nil).HasDimensions())
	assert.False(t, line(nil, nil, nil, nil).HasDimensions())
}
