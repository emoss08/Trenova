package xerorefsync

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/xeroref"
	"github.com/stretchr/testify/assert"
)

func TestAccountClassOfEveryXeroType(t *testing.T) {
	t.Parallel()

	cases := map[string]accountingmapping.AccountClass{
		"SALES":       accountingmapping.AccountClassIncome,
		"OTHERINCOME": accountingmapping.AccountClassOtherIncome,
		"OVERHEADS":   accountingmapping.AccountClassExpense,
		"DEPRECIATN":  accountingmapping.AccountClassExpense,
		"FIXED":       accountingmapping.AccountClassAsset,
		"PREPAYMENT":  accountingmapping.AccountClassAsset,
		"INVENTORY":   accountingmapping.AccountClassAsset,
		"NONCURRENT":  accountingmapping.AccountClassAsset,
		"TERMLIAB":    accountingmapping.AccountClassLiability,
		"LIABILITY":   accountingmapping.AccountClassLiability,
		"EQUITY":      accountingmapping.AccountClassEquity,
		"unknown":     "",
	}
	for typ, want := range cases {
		assert.Equal(t, want, AccountClassOf(&xeroref.Account{Type: typ}), typ)
	}
	assert.Equal(t, accountingmapping.AccountClassReceivable,
		AccountClassOf(&xeroref.Account{Type: "CURRENT", SystemAccount: "DEBTORS"}))
}

func TestItemCodeIsTheDraftName(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Line haul",
		itemCode(&accountingconnectorports.AccountingItemDraft{Name: "Line haul"}),
	)
}
