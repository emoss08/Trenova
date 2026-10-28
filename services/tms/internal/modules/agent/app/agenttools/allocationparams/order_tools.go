package allocationparams

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

const ParamAllocations = "allocations"

const ParamBillToCustomer = "billToCustomerId"

const ParamAllocMethod = "method"

const ParamAllocPercent = "percent"

const maxChargeAllocations = 20

var allocationMethods = agenttoolschema.Source(
	"shipment.chargeAllocationMethod",
	shipment.ChargeAllocationMethodValues(),
)
