package recordlabelrepository

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/shared/pulid"
)

type statusSource func(id pulid.ID) projection

//nolint:exhaustive // only the kinds a wait can name a status of are listed
var statusSources = map[permission.Resource]statusSource{
	permission.ResourceShipment: func(id pulid.ID) projection {
		t := shipmentdb.Shipments
		return t.Pick(t.ID, labelText(t.Status)).Where(t.ID.Eq(id))
	},
	permission.ResourceBillingQueue: func(id pulid.ID) projection {
		t := billingqueuedb.BillingQueueItems
		return t.Pick(t.ID, labelText(t.Status)).Where(t.ID.Eq(id))
	},
	permission.ResourceInvoice: func(id pulid.ID) projection {
		t := invoicedb.Invoices
		return t.Pick(t.ID, labelText(t.Status)).Where(t.ID.Eq(id))
	},
	permission.ResourceOrder: func(id pulid.ID) projection {
		t := orderdb.Orders
		return t.Pick(t.ID, labelText(t.Status)).Where(t.ID.Eq(id))
	},
	permission.ResourceDetention: func(id pulid.ID) projection {
		t := detentiondb.DetentionOccurrences
		return t.Pick(t.ID, labelText(t.Status)).Where(t.ID.Eq(id))
	},
}
