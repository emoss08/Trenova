package bcrefsync

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/shared/businesscentralref"
	"github.com/stretchr/testify/assert"
)

func TestAccountClassOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		sub      string
		want     accountingmapping.AccountClass
	}{
		{
			businesscentralref.CategoryIncome,
			"Income, Services",
			accountingmapping.AccountClassIncome,
		},
		{businesscentralref.CategoryCostOfGoodsSold, "", accountingmapping.AccountClassCostOfSales},
		{businesscentralref.CategoryExpense, "Rent Expense", accountingmapping.AccountClassExpense},
		{
			businesscentralref.CategoryAssets,
			"Accounts Receivable",
			accountingmapping.AccountClassReceivable,
		},
		{businesscentralref.CategoryAssets, "Cash", accountingmapping.AccountClassBank},
		{businesscentralref.CategoryAssets, "Bank Accounts", accountingmapping.AccountClassBank},
		{businesscentralref.CategoryAssets, "Inventory", accountingmapping.AccountClassAsset},
		{
			businesscentralref.CategoryLiabilities,
			"Accounts Payable",
			accountingmapping.AccountClassPayable,
		},
		{
			businesscentralref.CategoryLiabilities,
			"Long Term Liabilities",
			accountingmapping.AccountClassLiability,
		},
		{
			businesscentralref.CategoryEquity,
			"Retained Earnings",
			accountingmapping.AccountClassEquity,
		},
		{"", "", ""},
	}
	for _, tc := range cases {
		got := AccountClassOf(
			&businesscentralref.Account{Category: tc.category, SubCategory: tc.sub},
		)
		assert.Equal(t, tc.want, got, tc.category+"/"+tc.sub)
	}
	assert.True(t, AccountObjectOf(&businesscentralref.Account{
		AccountType: businesscentralref.AccountTypePosting, DirectPosting: false,
	}).Active)
	assert.False(t, AccountObjectOf(&businesscentralref.Account{
		AccountType: businesscentralref.AccountTypePosting, Blocked: true,
	}).Active)
	assert.False(t, AccountObjectOf(&businesscentralref.Account{AccountType: "Heading"}).Active)
}
