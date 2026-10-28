package shipmentpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type duplicateCopyView struct {
	Source            string              `json:"source"`
	CustomerID        pulid.ID            `json:"customerId"`
	BOL               string              `json:"bol"`
	FreightTerms      string              `json:"freightTerms"`
	FormulaTemplateID pulid.ID            `json:"formulaTemplateId"`
	FirstPickupAt     int64               `json:"firstPickupAt"`
	LastDeliveryAt    int64               `json:"lastDeliveryAt"`
	Stops             int                 `json:"stops"`
	Commodities       int                 `json:"commodities"`
	Charges           int                 `json:"charges"`
	TotalChargeAmount decimal.NullDecimal `json:"totalChargeAmount"`
}

func duplicateViewOf(source string, copied *shipment.Shipment) *duplicateCopyView {
	view := &duplicateCopyView{
		Source:            source,
		CustomerID:        copied.CustomerID,
		BOL:               copied.BOL,
		FreightTerms:      string(copied.FreightTerms),
		FormulaTemplateID: copied.FormulaTemplateID,
		Commodities:       len(copied.Commodities),
		Charges:           len(copied.AdditionalCharges),
		TotalChargeAmount: copied.TotalChargeAmount,
	}
	for _, move := range copied.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			view.Stops++
			if view.FirstPickupAt == 0 {
				view.FirstPickupAt = stop.ScheduledWindowStart
			}
			view.LastDeliveryAt = stop.ScheduledWindowStart
		}
	}

	return view
}

func ShipmentCopyChange(
	label, source string,
	copied *shipment.Shipment,
) (*agent.RecordChange, error) {
	return toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipment, Label: label},
		duplicateViewOf(source, copied),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldCustomerID:        permission.ResourceCustomer,
			changepreview.FieldFormulaTemplateID: permission.ResourceFormulaTemplate,
		}),
		toolpreview.Types(operationDateTimes),
		toolpreview.SensitiveAs("totalChargeAmount"),
	)
}
