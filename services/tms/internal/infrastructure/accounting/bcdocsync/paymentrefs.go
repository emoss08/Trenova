package bcdocsync

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
)

const (
	DocTypeCustomerPayment = "CustomerPayment"
	DocTypeVendorPayment   = "VendorPayment"
)

func finishPayment(
	result *accountingconnectorports.AccountingDocumentResult,
	number, docType string,
) {
	result.ExternalID = number
	result.DocNumber = number
	result.Refs[accountingsync.ExternalRefDocument] = number
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
}
