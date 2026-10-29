package bcconnector

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/shared/businesscentralref"
)

func referenceKindOf(kind businesscentralref.PartyKind) accountingprovider.ReferenceKind {
	if kind == businesscentralref.PartyVendor {
		return accountingprovider.ReferenceKindVendor
	}
	return accountingprovider.ReferenceKindCustomer
}
