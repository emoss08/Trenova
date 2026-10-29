package crossref

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
)

func VisibleRecordSources() kernelports.VisibleRecordSources {
	return kernelports.VisibleRecordSources{
		{Kind: permission.ResourceInvoice.String(), ID: invoicedb.Invoices.ID},
		{Kind: permission.ResourceOrder.String(), ID: orderdb.Orders.ID},
		{Kind: permission.ResourceShipment.String(), ID: shipmentdb.Shipments.ID},
		{Kind: permission.ResourceWorker.String(), ID: workerdb.Workers.ID},
	}
}

func OrderReferences() orderports.OrderReferences {
	return orderports.OrderReferences{
		invoicedb.Invoices.OrderID,
		billingqueuedb.BillingQueueItems.OrderID,
	}
}

func TemplateConsumers() formulatemplateports.TemplateConsumers {
	return formulatemplateports.TemplateConsumers{Shipments: shipmentdb.Shipments.FormulaTemplateID}
}

func RotaAssignments() workerscheduleports.RotaAssignments {
	a := shipmentdb.Assignments
	return workerscheduleports.RotaAssignments{
		WorkerID:   a.PrimaryWorkerID,
		CreatedAt:  a.CreatedAt,
		ArchivedAt: a.ArchivedAt,
	}
}

func LocationColumns() equipmentcontinuityports.LocationColumns {
	l := locationdb.Locations
	return equipmentcontinuityports.LocationColumns{
		ID:        l.ID,
		Name:      l.Name,
		Latitude:  l.Latitude,
		Longitude: l.Longitude,
	}
}

func MoveColumns() equipmentcontinuityports.MoveColumns {
	a := shipmentdb.Assignments
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	st := shipmentdb.Stops
	return equipmentcontinuityports.MoveColumns{
		AssignmentMoveID:     a.ShipmentMoveID,
		AssignmentTractorID:  a.TractorID,
		AssignmentTrailerID:  a.TrailerID,
		AssignmentArchivedAt: a.ArchivedAt,
		MoveID:               sm.ID,
		MoveShipmentID:       sm.ShipmentID,
		MoveInProgress:       sm.Status.Eq(shipmentvocab.MoveStatusInTransit),
		ShipmentID:           sp.ID,
		ShipmentProNumber:    sp.ProNumber,
		StopMoveID:           st.ShipmentMoveID,
		StopSequence:         st.Sequence,
		StopWindowStart:      st.ScheduledWindowStart,
		StopWindowEnd:        st.ScheduledWindowEnd,
	}
}
