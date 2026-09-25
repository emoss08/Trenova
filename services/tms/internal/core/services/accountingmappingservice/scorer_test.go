package accountingmappingservice

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func acct(id, name, accountType string) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:        accountingsync.ReferenceKindAccount,
		ExternalID:  id,
		Name:        name,
		AccountType: accountType,
		Active:      true,
	}
}

func item(id, name, sku string) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:       accountingsync.ReferenceKindItem,
		ExternalID: id,
		Name:       name,
		Number:     sku,
		ItemType:   "Service",
		Active:     true,
	}
}

func party(kind accountingsync.ReferenceKind, id, name, postal string) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:       kind,
		ExternalID: id,
		Name:       name,
		PostalCode: postal,
		Active:     true,
	}
}

func chartOfAccounts() []*accountingsync.AccountingReferenceObject {
	return []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "84", Name: "Accounts Receivable (A/R)", Number: "1200", AccountType: "Accounts Receivable", Active: true},
		acct("79", "Freight Income", "Income"),
		acct("80", "Fuel Surcharge Income", "Income"),
		acct("35", "Checking", "Bank"),
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "4", Name: "Undeposited Funds", AccountType: "Other Current Asset", AccountSubType: "UndepositedFunds", Active: true},
		acct("33", "Accounts Payable (A/P)", "Accounts Payable"),
		acct("90", "Bad Debt", "Expense"),
		acct("91", "Purchased Transportation", "Cost of Goods Sold"),
		acct("92", "Office Supplies", "Expense"),
	}
}

func TestScoreAccountRoleMatchesTheAccountNumberFirst(t *testing.T) {
	t.Parallel()

	p := score(&target{
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

	p := score(&target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleRevenue,
		Names:      []string{"Freight Revenue"},
	}, chartOfAccounts(), nil)

	assert.Equal(t, "79", p.ExternalID)
	for _, candidate := range p.Candidates {
		assert.Contains(t, []string{"79", "80"}, candidate.ExternalID, "only income accounts are candidates")
	}
	assert.GreaterOrEqual(t, p.Confidence, accountingsync.ConfidentConfidence)
}

func TestScoreTheOnlyEligibleAccountIsConfident(t *testing.T) {
	t.Parallel()

	p := score(&target{
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

	p := score(&target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleWriteOff,
		Names:      roleSynonyms[accountingsync.AccountRoleWriteOff],
	}, []*accountingsync.AccountingReferenceObject{acct("92", "Office Supplies", "Expense")}, nil)

	assert.Empty(t, p.ExternalID)
}

func TestScoreDepositRoleAcceptsBankOrUndepositedFunds(t *testing.T) {
	t.Parallel()

	p := score(&target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleDeposit,
		Names:      roleSynonyms[accountingsync.AccountRoleDeposit],
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

	items := []*accountingsync.AccountingReferenceObject{
		item("12", "Detention", "DET"),
		item("13", "Lumper Fee", ""),
		item("14", "DRY", ""),
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "11", Name: "Accessorials", ItemType: "Category", Active: true},
	}

	bySku := score(&target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   pulid.MustNew("acc_"),
		Code:       "DET",
		Names:      []string{"Detention time at shipper"},
	}, items, nil)
	assert.Equal(t, "12", bySku.ExternalID)
	assert.InDelta(t, 0.99, bySku.Confidence, 1e-9)

	byName := score(&target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   pulid.MustNew("acc_"),
		Code:       "dry",
		Names:      []string{"Dry run"},
	}, items, nil)
	assert.Equal(t, "14", byName.ExternalID)

	byDescription := score(&target{
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

	p := score(&target{
		TargetType: accountingsync.TargetLineType,
		Key:        "Memo",
		Names:      lineTypeSynonyms["Memo"],
	}, []*accountingsync.AccountingReferenceObject{
		item("1", "Hazmat Surcharge", ""),
		item("2", "Tolls", ""),
	}, nil)

	assert.Empty(t, p.ExternalID)
	assert.NotEmpty(t, p.Candidates)
	assert.LessOrEqual(t, len(p.Candidates), accountingsync.MaxCandidates)
}

func TestScoreCustomerIgnoresLegalSuffixesAndUsesPostalCode(t *testing.T) {
	t.Parallel()

	customers := []*accountingsync.AccountingReferenceObject{
		party(accountingsync.ReferenceKindCustomer, "58", "Peak Distributing LLC", "80202"),
		party(accountingsync.ReferenceKindCustomer, "59", "Peak Distributing Inc", "10001"),
		party(accountingsync.ReferenceKindCustomer, "60", "Summit Foods", "80202"),
	}

	p := score(&target{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   pulid.MustNew("cus_"),
		Names:      []string{"Peak Distributing"},
		PostalCode: "80202",
	}, customers, newTokenIndex(customers))

	assert.Equal(t, "58", p.ExternalID, "the equal postal code breaks the tie")
	assert.InDelta(t, 0.99, p.Confidence, 1e-9)
	assert.Contains(t, p.Reason, "postal code")
}

func TestScoreVendorMatchesAnMCNumberInTheAccountNumber(t *testing.T) {
	t.Parallel()

	vendors := []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindVendor, ExternalID: "91", Name: "SH Logistics", Number: "MC-123456", Active: true},
		party(accountingsync.ReferenceKindVendor, "92", "Swift Haul", ""),
	}

	p := score(&target{
		TargetType:  accountingsync.TargetCarrier,
		ObjectID:    pulid.MustNew("carr_"),
		Names:       []string{"Swift Haul Inc"},
		Identifiers: []string{"123456"},
	}, vendors, newTokenIndex(vendors))

	assert.Equal(t, "91", p.ExternalID)
	assert.InDelta(t, 0.99, p.Confidence, 1e-9)
	assert.Contains(t, p.Reason, "123456")
}

func TestScoreTermMatchesDueDays(t *testing.T) {
	t.Parallel()

	thirty, zero := 30, 0
	terms := []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindTerm, ExternalID: "3", Name: "Thirty days", DueDays: &thirty, Active: true},
		{Kind: accountingsync.ReferenceKindTerm, ExternalID: "1", Name: "Upon receipt", DueDays: &zero, Active: true},
	}

	net30 := score(&target{TargetType: accountingsync.TargetPaymentTerm, Key: "Net30", DueDays: &thirty}, terms, nil)
	assert.Equal(t, "3", net30.ExternalID)
	assert.InDelta(t, 0.98, net30.Confidence, 1e-9)

	onReceipt := score(&target{TargetType: accountingsync.TargetPaymentTerm, Key: "DueOnReceipt", DueDays: &zero}, terms, nil)
	assert.Equal(t, "1", onReceipt.ExternalID)
}

func TestScorePaymentMethodUsesSynonyms(t *testing.T) {
	t.Parallel()

	methods := []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindPaymentMethod, ExternalID: "1", Name: "Visa", Active: true},
		{Kind: accountingsync.ReferenceKindPaymentMethod, ExternalID: "2", Name: "Check", Active: true},
		{Kind: accountingsync.ReferenceKindPaymentMethod, ExternalID: "3", Name: "EFT", Active: true},
	}

	card := score(&target{TargetType: accountingsync.TargetPaymentMethod, Key: "Card", Names: paymentMethodSynonyms["Card"]}, methods, nil)
	assert.Equal(t, "1", card.ExternalID)
	ach := score(&target{TargetType: accountingsync.TargetPaymentMethod, Key: "ACH", Names: paymentMethodSynonyms["ACH"]}, methods, nil)
	assert.Equal(t, "3", ach.ExternalID)
}

func TestScoreSkipsInactiveAndRemovedRecords(t *testing.T) {
	t.Parallel()

	removedAt := int64(1)
	p := score(&target{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleAR,
		Names:      []string{"Accounts Receivable"},
	}, []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "1", Name: "Accounts Receivable", AccountType: "Accounts Receivable"},
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "2", Name: "Accounts Receivable", AccountType: "Accounts Receivable", Active: true, RemovedAt: &removedAt},
	}, nil)

	assert.Empty(t, p.ExternalID)
	assert.Empty(t, p.Candidates)
}

func TestTokenIndexNarrowsLargeListsWithoutLosingTheMatch(t *testing.T) {
	t.Parallel()

	customers := make([]*accountingsync.AccountingReferenceObject, 0, 5001)
	for idx := range 5000 {
		customers = append(customers, party(accountingsync.ReferenceKindCustomer, fmt.Sprintf("c%d", idx), fmt.Sprintf("Customer %d Trucking", idx), ""))
	}
	customers = append(customers, party(accountingsync.ReferenceKindCustomer, "target", "Blue Ridge Produce", ""))

	index := newTokenIndex(customers)
	candidates := index.candidates([]string{"Blue Ridge Produce Co"})
	require.NotEmpty(t, candidates)
	assert.Less(t, len(candidates), 50, "common words like Customer and Trucking do not pull in every record")

	p := score(&target{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   pulid.MustNew("cus_"),
		Names:      []string{"Blue Ridge Produce Co"},
	}, customers, index)
	assert.Equal(t, "target", p.ExternalID)
}

func TestScoreRecordsWhichMatchersFired(t *testing.T) {
	t.Parallel()

	p := score(&target{
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
