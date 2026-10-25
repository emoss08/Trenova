package core

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
)

func newVisibleRecordSources() kernelports.VisibleRecordSources {
	return kernelports.VisibleRecordSources{
		{Kind: permission.ResourceInvoice.String(), ID: invoicedb.Invoices.ID},
		{Kind: permission.ResourceOrder.String(), ID: orderdb.Orders.ID},
		{Kind: permission.ResourceShipment.String(), ID: shipmentdb.Shipments.ID},
		{Kind: permission.ResourceWorker.String(), ID: workerdb.Workers.ID},
	}
}
