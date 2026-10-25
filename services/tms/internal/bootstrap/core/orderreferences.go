package core

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
)

func newOrderReferences() orderports.OrderReferences {
	return orderports.OrderReferences{
		invoicedb.Invoices.OrderID,
		billingqueuedb.BillingQueueItems.OrderID,
	}
}
