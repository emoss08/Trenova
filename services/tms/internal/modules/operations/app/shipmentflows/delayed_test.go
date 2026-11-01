package shipmentflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/stretchr/testify/assert"
)

func TestDelayThresholdMinutes_DisablesAutomaticDelayWhenToggleOff(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		shipmentstate.DisabledDelayThresholdMinutes,
		shipmenteffects.DelayThresholdMinutes(nil),
	)
	assert.Equal(
		t,
		shipmentstate.DisabledDelayThresholdMinutes,
		shipmenteffects.DelayThresholdMinutes(&shipmentcontrol.ShipmentControl{
			AutoDelayShipments:          false,
			AutoDelayShipmentsThreshold: new(int16(15)),
		}),
	)
	assert.Equal(
		t,
		int16(15),
		shipmenteffects.DelayThresholdMinutes(&shipmentcontrol.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: new(int16(15)),
		}),
	)
}
