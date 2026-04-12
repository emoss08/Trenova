package customerledger

import "github.com/emoss08/trenova/shared/pulid"

type Entry struct {
	ID               pulid.ID `json:"id"`
	OrganizationID   pulid.ID `json:"organizationId"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"`
	CustomerID       pulid.ID `json:"customerId"`
	SourceObjectType string   `json:"sourceObjectType"`
	SourceObjectID   string   `json:"sourceObjectId"`
	SourceEventType  string   `json:"sourceEventType"`
	RelatedInvoiceID pulid.ID `json:"relatedInvoiceId"`
	DocumentNumber   string   `json:"documentNumber"`
	TransactionDate  int64    `json:"transactionDate"`
	LineNumber       int      `json:"lineNumber"`
	AmountMinor      int64    `json:"amountMinor"`
	CreatedByID      pulid.ID `json:"createdById"`
}
