package accountingmappingscore

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func acct(id, name, accountType string) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:         accountingprovider.ReferenceKindAccount,
		ExternalID:   id,
		Name:         name,
		AccountType:  accountType,
		AccountClass: accountingmappingtest.AccountClass(accountType),
		Active:       true,
	}
}

func scoreQBO(
	t *Target,
	refs []*accountingmapping.AccountingReferenceObject,
	index *TokenIndex,
) *accountingmapping.Proposal {
	profile := accountingprovider.MustProfile(integration.TypeQuickBooksOnline)
	t.Kind = profile.KindFor(t.TargetType)
	return Score(t, refs, index)
}

func item(id, name, sku string) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:       accountingprovider.ReferenceKindItem,
		ExternalID: id,
		Name:       name,
		Number:     sku,
		ItemType:   "Service",
		Active:     true,
	}
}

func chartOfAccounts() []*accountingmapping.AccountingReferenceObject {
	return []*accountingmapping.AccountingReferenceObject{
		{
			Kind:         accountingprovider.ReferenceKindAccount,
			ExternalID:   "84",
			Name:         "Accounts Receivable (A/R)",
			Number:       "1200",
			AccountType:  "Accounts Receivable",
			AccountClass: accountingmapping.AccountClassReceivable,
			Active:       true,
		},
		acct("79", "Freight Income", "Income"),
		acct("80", "Fuel Surcharge Income", "Income"),
		acct("35", "Checking", "Bank"),
		{
			Kind:           accountingprovider.ReferenceKindAccount,
			ExternalID:     "4",
			Name:           "Undeposited Funds",
			AccountType:    "Other Current Asset",
			AccountSubType: "UndepositedFunds",
			AccountClass:   accountingmapping.AccountClassUndepositedFunds,
			Active:         true,
		},
		acct("33", "Accounts Payable (A/P)", "Accounts Payable"),
		acct("90", "Bad Debt", "Expense"),
		acct("91", "Purchased Transportation", "Cost of Goods Sold"),
		acct("92", "Office Supplies", "Expense"),
	}
}

func TestScoreAccountRoleMatchesTheAccountNumberFirst(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleAR,
		Names:      []string{"Trade Receivables"},
		Code:       "1200",
	}, chartOfAccounts(), nil)

	assert.Equal(t, "84", p.ExternalID)
	assert.InDelta(t, 0.99, p.Confidence, 1e-9)
	assert.Contains(t, p.Reason, "1200")
}

func TestScoreAccountRoleOnlyConsidersEligibleAccountTypes(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleRevenue,
		Names:      []string{"Freight Revenue"},
	}, chartOfAccounts(), nil)

	assert.Equal(t, "79", p.ExternalID)
	for _, candidate := range p.Candidates {
		assert.Contains(
			t,
			[]string{"79", "80"},
			candidate.ExternalID,
			"only income accounts are candidates",
		)
	}
	assert.GreaterOrEqual(t, p.Confidence, accountingmapping.ConfidentConfidence)
}

func TestScoreTheOnlyEligibleAccountIsConfident(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleAP,
		Names:      []string{"Carrier Payables"},
	}, chartOfAccounts(), nil)

	assert.Equal(t, "33", p.ExternalID)
	assert.GreaterOrEqual(t, p.Confidence, 0.96)
	assert.Contains(t, p.Reason, "only")
}

func TestScoreTheOnlyExpenseAccountIsNotProposedForAGenericRole(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleWriteOff,
		Names:      RoleSynonyms[accountingsync.AccountRoleWriteOff],
	}, []*accountingmapping.AccountingReferenceObject{acct("92", "Office Supplies", "Expense")}, nil)

	assert.Empty(t, p.ExternalID)
}

func TestScoreDepositRoleAcceptsBankOrUndepositedFunds(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleDeposit,
		Names:      RoleSynonyms[accountingsync.AccountRoleDeposit],
	}, chartOfAccounts(), nil)

	assert.Equal(t, "4", p.ExternalID)
	ids := make([]string, 0, len(p.Candidates))
	for _, candidate := range p.Candidates {
		ids = append(ids, candidate.ExternalID)
	}
	assert.ElementsMatch(t, []string{"4", "35"}, ids)
}

func TestScoreAccessorialMatchesTheSkuOrTheCodeAsName(t *testing.T) {
	t.Parallel()

	items := []*accountingmapping.AccountingReferenceObject{
		item("12", "Detention", "DET"),
		item("13", "Lumper Fee", ""),
		item("14", "DRY", ""),
		{
			Kind:       accountingprovider.ReferenceKindItem,
			ExternalID: "11",
			Name:       "Accessorials",
			ItemType:   "Category",
			Active:     true,
		},
	}

	bySku := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   pulid.MustNew("acc_"),
		Code:       "DET",
		Names:      []string{"Detention time at shipper"},
	}, items, nil)
	assert.Equal(t, "12", bySku.ExternalID)
	assert.InDelta(t, 0.99, bySku.Confidence, 1e-9)

	byName := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   pulid.MustNew("acc_"),
		Code:       "dry",
		Names:      []string{"Dry run"},
	}, items, nil)
	assert.Equal(t, "14", byName.ExternalID)

	byDescription := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   pulid.MustNew("acc_"),
		Code:       "LMP",
		Names:      []string{"Lumper fee"},
	}, items, nil)
	assert.Equal(t, "13", byDescription.ExternalID)
	for _, candidate := range byDescription.Candidates {
		assert.NotEqual(t, "11", candidate.ExternalID, "a category is never a candidate")
	}
}

func TestScoreNoConfidentMatchKeepsCandidatesButProposesNothing(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetLineType,
		Key:        "Memo",
		Names:      LineTypeSynonyms["Memo"],
	}, []*accountingmapping.AccountingReferenceObject{
		item("1", "Hazmat Surcharge", ""),
		item("2", "Tolls", ""),
	}, nil)

	assert.Empty(t, p.ExternalID)
	assert.NotEmpty(t, p.Candidates)
	assert.LessOrEqual(t, len(p.Candidates), accountingmapping.MaxCandidates)
}

func TestScoreCustomerIgnoresLegalSuffixesAndUsesPostalCode(t *testing.T) {
	t.Parallel()

	customers := []*accountingmapping.AccountingReferenceObject{
		accountingmappingtest.Party(
			accountingprovider.ReferenceKindCustomer,
			"58",
			"Peak Distributing LLC",
			"80202",
		),
		accountingmappingtest.Party(
			accountingprovider.ReferenceKindCustomer,
			"59",
			"Peak Distributing Inc",
			"10001",
		),
		accountingmappingtest.Party(
			accountingprovider.ReferenceKindCustomer,
			"60",
			"Summit Foods",
			"80202",
		),
	}

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   pulid.MustNew("cus_"),
		Names:      []string{"Peak Distributing"},
		PostalCode: "80202",
	}, customers, NewTokenIndex(customers))

	assert.Equal(t, "58", p.ExternalID, "the equal postal code breaks the tie")
	assert.InDelta(t, 0.99, p.Confidence, 1e-9)
	assert.Contains(t, p.Reason, "postal code")
}

func TestScoreVendorMatchesAnMCNumberInTheAccountNumber(t *testing.T) {
	t.Parallel()

	vendors := []*accountingmapping.AccountingReferenceObject{
		{
			Kind:       accountingprovider.ReferenceKindVendor,
			ExternalID: "91",
			Name:       "SH Logistics",
			Number:     "MC-123456",
			Active:     true,
		},
		accountingmappingtest.Party(accountingprovider.ReferenceKindVendor, "92", "Swift Haul", ""),
	}

	p := scoreQBO(&Target{
		TargetType:  accountingsync.TargetCarrier,
		ObjectID:    pulid.MustNew("carr_"),
		Names:       []string{"Swift Haul Inc"},
		Identifiers: []string{"123456"},
	}, vendors, NewTokenIndex(vendors))

	assert.Equal(t, "91", p.ExternalID)
	assert.InDelta(t, 0.99, p.Confidence, 1e-9)
	assert.Contains(t, p.Reason, "123456")
}

func TestScoreTermMatchesDueDays(t *testing.T) {
	t.Parallel()

	thirty, zero := 30, 0
	terms := []*accountingmapping.AccountingReferenceObject{
		{
			Kind:       accountingprovider.ReferenceKindTerm,
			ExternalID: "3",
			Name:       "Thirty days",
			DueDays:    &thirty,
			Active:     true,
		},
		{
			Kind:       accountingprovider.ReferenceKindTerm,
			ExternalID: "1",
			Name:       "Upon receipt",
			DueDays:    &zero,
			Active:     true,
		},
	}

	net30 := scoreQBO(
		&Target{TargetType: accountingsync.TargetPaymentTerm, Key: "Net30", DueDays: &thirty},
		terms,
		nil,
	)
	assert.Equal(t, "3", net30.ExternalID)
	assert.InDelta(t, 0.98, net30.Confidence, 1e-9)

	onReceipt := scoreQBO(
		&Target{TargetType: accountingsync.TargetPaymentTerm, Key: "DueOnReceipt", DueDays: &zero},
		terms,
		nil,
	)
	assert.Equal(t, "1", onReceipt.ExternalID)
}

func TestScorePaymentMethodUsesSynonyms(t *testing.T) {
	t.Parallel()

	methods := []*accountingmapping.AccountingReferenceObject{
		{
			Kind:       accountingprovider.ReferenceKindPaymentMethod,
			ExternalID: "1",
			Name:       "Visa",
			Active:     true,
		},
		{
			Kind:       accountingprovider.ReferenceKindPaymentMethod,
			ExternalID: "2",
			Name:       "Check",
			Active:     true,
		},
		{
			Kind:       accountingprovider.ReferenceKindPaymentMethod,
			ExternalID: "3",
			Name:       "EFT",
			Active:     true,
		},
	}

	card := scoreQBO(
		&Target{
			TargetType: accountingsync.TargetPaymentMethod,
			Key:        "Card",
			Names:      PaymentMethodSynonyms["Card"],
		},
		methods,
		nil,
	)
	assert.Equal(t, "1", card.ExternalID)
	ach := scoreQBO(
		&Target{
			TargetType: accountingsync.TargetPaymentMethod,
			Key:        "ACH",
			Names:      PaymentMethodSynonyms["ACH"],
		},
		methods,
		nil,
	)
	assert.Equal(t, "3", ach.ExternalID)
}

func TestScoreSkipsInactiveAndRemovedRecords(t *testing.T) {
	t.Parallel()

	removedAt := int64(1)
	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleAR,
		Names:      []string{"Accounts Receivable"},
	}, []*accountingmapping.AccountingReferenceObject{
		{
			Kind:         accountingprovider.ReferenceKindAccount,
			ExternalID:   "1",
			Name:         "Accounts Receivable",
			AccountType:  "Accounts Receivable",
			AccountClass: accountingmapping.AccountClassReceivable,
		},
		{
			Kind:         accountingprovider.ReferenceKindAccount,
			ExternalID:   "2",
			Name:         "Accounts Receivable",
			AccountType:  "Accounts Receivable",
			AccountClass: accountingmapping.AccountClassReceivable,
			Active:       true,
			RemovedAt:    &removedAt,
		},
	}, nil)

	assert.Empty(t, p.ExternalID)
	assert.Empty(t, p.Candidates)
}

func TestTokenIndexNarrowsLargeListsWithoutLosingTheMatch(t *testing.T) {
	t.Parallel()

	customers := make([]*accountingmapping.AccountingReferenceObject, 0, 5001)
	for idx := range 5000 {
		customers = append(
			customers,
			accountingmappingtest.Party(
				accountingprovider.ReferenceKindCustomer,
				fmt.Sprintf("c%d", idx),
				fmt.Sprintf("Customer %d Trucking", idx),
				"",
			),
		)
	}
	customers = append(
		customers,
		accountingmappingtest.Party(
			accountingprovider.ReferenceKindCustomer,
			"target",
			"Blue Ridge Produce",
			"",
		),
	)

	index := NewTokenIndex(customers)
	candidates := index.candidates([]string{"Blue Ridge Produce Co"})
	require.NotEmpty(t, candidates)
	assert.Less(
		t,
		len(candidates),
		50,
		"common words like Customer and Trucking do not pull in every record",
	)

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   pulid.MustNew("cus_"),
		Names:      []string{"Blue Ridge Produce Co"},
	}, customers, index)
	assert.Equal(t, "target", p.ExternalID)
}

func TestScoreRecordsWhichMatchersFired(t *testing.T) {
	t.Parallel()

	p := scoreQBO(&Target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleAR,
		Names:      []string{"Accounts Receivable"},
		Code:       "1200",
	}, chartOfAccounts(), nil)

	require.NotEmpty(t, p.Matchers)
	names := make([]string, 0, len(p.Matchers))
	for _, matcher := range p.Matchers {
		names = append(names, matcher.Matcher)
	}
	assert.Contains(t, names, matcherNumber)
}
