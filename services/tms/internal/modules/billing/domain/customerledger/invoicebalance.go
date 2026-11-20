package customerledger

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type InvoiceBalance struct {
	keel.Table `keel:"customer_ledger_invoice_balances"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CustomerID     pulid.ID `json:"customerId"     keel:"customer_id,pk"`
	InvoiceID      pulid.ID `json:"invoiceId"      keel:"invoice_id,pk"`
	AmountMinor    int64    `json:"amountMinor"    keel:"amount_minor"`
}
