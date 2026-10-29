package shipmentpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
)

var operationDateTimes = map[string]assistantartifact.DisplayType{
	changepreview.FieldScheduledWindowStart: assistantartifact.DisplayDateTime,
	changepreview.FieldScheduledWindowEnd:   assistantartifact.DisplayDateTime,
	"firstPickupAt":                         assistantartifact.DisplayDateTime,
	"lastDeliveryAt":                        assistantartifact.DisplayDateTime,
	"startedAt":                             assistantartifact.DisplayDateTime,
	"effectiveFrom":                         assistantartifact.DisplayDate,
	"effectiveTo":                           assistantartifact.DisplayDate,
	"shiftDate":                             assistantartifact.DisplayDate,
	"counterpartyShiftDate":                 assistantartifact.DisplayDate,
	"startDate":                             assistantartifact.DisplayDate,
	"endDate":                               assistantartifact.DisplayDate,
}
