package accountingmappingtest

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
)

func AccountClass(accountType string) accountingmapping.AccountClass {
	switch accountType {
	case "Accounts Receivable":
		return accountingmapping.AccountClassReceivable
	case "Accounts Payable":
		return accountingmapping.AccountClassPayable
	case "Bank":
		return accountingmapping.AccountClassBank
	case "Income":
		return accountingmapping.AccountClassIncome
	case "Other Income":
		return accountingmapping.AccountClassOtherIncome
	case "Cost of Goods Sold":
		return accountingmapping.AccountClassCostOfSales
	case "Expense":
		return accountingmapping.AccountClassExpense
	case "Other Expense":
		return accountingmapping.AccountClassOtherExpense
	default:
		return accountingmapping.AccountClassAsset
	}
}

func Party(
	kind accountingprovider.ReferenceKind,
	id, name, postal string,
) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:       kind,
		ExternalID: id,
		Name:       name,
		PostalCode: postal,
		Active:     true,
	}
}
