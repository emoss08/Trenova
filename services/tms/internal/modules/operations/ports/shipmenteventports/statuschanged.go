package shipmenteventports

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

func StatusChanged(
	sh *shipment.Shipment,
	previous shipmentvocab.Status,
	actor kernelports.AuditActor,
) *RecordShipmentEventParams {
	return &RecordShipmentEventParams{
		OrganizationID: sh.OrganizationID,
		BusinessUnitID: sh.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeStatusChanged,
		Severity:       SeverityForShipmentStatus(sh.Status),
		Summary:        "Status updated",
		Metadata: map[string]any{
			"proNumber":      sh.ProNumber,
			"previousStatus": string(previous),
			"newStatus":      string(sh.Status),
		},
		Actor: actor,
	}
}

func SeverityForShipmentStatus(status shipmentvocab.Status) shipmentevent.Severity {
	switch status {
	case shipmentvocab.StatusCanceled, shipmentvocab.StatusDelayed:
		return shipmentevent.SeverityDanger
	case shipmentvocab.StatusInTransit, shipmentvocab.StatusReadyToInvoice:
		return shipmentevent.SeverityBrand
	case shipmentvocab.StatusCompleted, shipmentvocab.StatusInvoiced:
		return shipmentevent.SeveritySuccess
	case shipmentvocab.StatusNew,
		shipmentvocab.StatusPartiallyAssigned,
		shipmentvocab.StatusAssigned,
		shipmentvocab.StatusPartiallyCompleted:
		return shipmentevent.SeverityMuted
	}
	return shipmentevent.SeverityMuted
}
