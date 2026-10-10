package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int16Ptr(value int16) *int16 { return &value }

func TestApplyShipmentPatch_RefusesPlaceholderTemperaturesOnADryLoad(t *testing.T) {
	t.Parallel()

	dry := &shipment.Shipment{}
	err := applyShipmentPatch(dry, shipmentPatch{
		TemperatureMin: int16Ptr(0),
		TemperatureMax: int16Ptr(0),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frozen load")
	assert.Nil(t, dry.TemperatureMin, "nothing was applied")

	reefer := &shipment.Shipment{TemperatureMin: int16Ptr(-10), TemperatureMax: int16Ptr(0)}
	require.NoError(t, applyShipmentPatch(reefer, shipmentPatch{
		TemperatureMin: int16Ptr(0),
		TemperatureMax: int16Ptr(0),
	}), "a load already under temperature control may be set to 0 to 0")
}

func TestApplyShipmentPatch_RefusesAnUpsideDownRange(t *testing.T) {
	t.Parallel()

	err := applyShipmentPatch(&shipment.Shipment{}, shipmentPatch{
		TemperatureMin: int16Ptr(40),
		TemperatureMax: int16Ptr(34),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "above the maximum")
}
