package shipmenttools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func TargetShipment(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}
