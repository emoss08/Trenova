package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

const ParamAllocations = "allocations"

const ParamBillToCustomer = "billToCustomerId"

const ParamAllocMethod = "method"

const ParamAllocPercent = "percent"

const maxChargeAllocations = 20

const CurrencyCodeLength = 3

var allocationMethods = agenttoolschema.Source(
	"shipment.chargeAllocationMethod",
	shipment.ChargeAllocationMethodValues(),
)

func SendsAny(params map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, given := params[key]; given {
			return true
		}
	}

	return false
}
