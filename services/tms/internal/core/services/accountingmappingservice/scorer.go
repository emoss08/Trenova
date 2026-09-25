package accountingmappingservice

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	scoreIdentifier       = 0.99
	scoreNumber           = 0.99
	scoreCode             = 0.99
	scoreDueDays          = 0.98
	scoreEqualCompany     = 0.97
	scoreOnlyEligible     = 0.96
	postalCodeBonus       = 0.02
	maxDeterministicScore = 0.99
	nameScoreCeiling      = 0.97
	nameRankStep          = 0.02
	nameScoreFloor        = 0.5
	minIdentifierDigits   = 5
	indexCommonShare      = 0.02
	indexMinRecords       = 50
	indexMaxCandidates    = 200

	matcherIdentifier   = "identifier"
	matcherNumber       = "accountNumber"
	matcherCode         = "code"
	matcherName         = "name"
	matcherPostalCode   = "postalCode"
	matcherDueDays      = "dueDays"
	matcherOnlyEligible = "onlyEligible"
)

type target struct {
	TargetType  accountingsync.MappingTargetType
	ObjectID    pulid.ID
	Key         string
	Label       string
	Names       []string
	Code        string
	PostalCode  string
	Identifiers []string
	DueDays     *int
}

type candidateScore struct {
	ref      *accountingsync.AccountingReferenceObject
	score    float64
	reason   string
	matchers []accountingsync.MatcherSignal
}

var roleSynonyms = map[string][]string{
	accountingsync.AccountRoleAR: {"Accounts Receivable", "A/R", "Trade Receivables"},
	accountingsync.AccountRoleRevenue: {
		"Freight Revenue",
		"Freight Income",
		"Revenue",
		"Sales",
		"Income",
	},
	accountingsync.AccountRoleDeposit: {"Undeposited Funds", "Checking", "Operating Account"},
	accountingsync.AccountRoleWriteOff: {
		"Bad Debt",
		"Write Off",
		"Short Pay",
		"Discounts Given",
	},
	accountingsync.AccountRoleAP: {"Accounts Payable", "A/P"},
	accountingsync.AccountRolePurchasedTransportation: {
		"Purchased Transportation",
		"Carrier Expense",
		"Freight Expense",
		"Subcontracted Freight",
		"Cost of Freight",
	},
}

var lineTypeSynonyms = map[string][]string{
	string(invoice.InvoiceLineTypeFreight): {
		"Freight",
		"Linehaul",
		"Line Haul",
		"Freight Charges",
		"Transportation",
	},
	string(invoice.InvoiceLineTypeMemo): {
		"Memo",
		"Adjustment",
		"Miscellaneous",
		"Other Charges",
	},
}

var itemRoleSynonyms = map[string][]string{
	accountingsync.ItemRoleShortPayWriteOff: {
		"Short Pay",
		"Write Off",
		"Bad Debt",
		"Short Payment",
		"Discount",
	},
}

var paymentMethodSynonyms = map[string][]string{
	string(customerpayment.MethodACH): {
		"ACH",
		"EFT",
		"Electronic Funds Transfer",
		"Bank Transfer",
		"Direct Deposit",
	},
	string(customerpayment.MethodCheck): {"Check", "Cheque"},
	string(customerpayment.MethodWire):  {"Wire", "Wire Transfer"},
	string(customerpayment.MethodCard): {
		"Credit Card",
		"Card",
		"Visa",
		"Mastercard",
		"American Express",
		"Amex",
		"Discover",
		"Debit Card",
	},
	string(customerpayment.MethodCash):  {"Cash"},
	string(customerpayment.MethodOther): {"Other"},
}

var termSynonyms = map[string][]string{
	string(customer.PaymentTermNet10):        {"Net 10"},
	string(customer.PaymentTermNet15):        {"Net 15"},
	string(customer.PaymentTermNet30):        {"Net 30"},
	string(customer.PaymentTermNet45):        {"Net 45"},
	string(customer.PaymentTermNet60):        {"Net 60"},
	string(customer.PaymentTermNet90):        {"Net 90"},
	string(customer.PaymentTermDueOnReceipt): {"Due on receipt", "Due upon receipt"},
}

var roleAccountTypes = map[string][]string{
	accountingsync.AccountRoleAR:                      {"Accounts Receivable"},
	accountingsync.AccountRoleRevenue:                 {"Income", "Other Income"},
	accountingsync.AccountRoleDeposit:                 {"Bank"},
	accountingsync.AccountRoleWriteOff:                {"Expense", "Other Expense"},
	accountingsync.AccountRoleAP:                      {"Accounts Payable"},
	accountingsync.AccountRolePurchasedTransportation: {"Cost of Goods Sold", "Expense"},
}

const undepositedFundsSubType = "UndepositedFunds"

var dedicatedTypeRoles = map[string]struct{}{
	accountingsync.AccountRoleAR:      {},
	accountingsync.AccountRoleAP:      {},
	accountingsync.AccountRoleDeposit: {},
}

func eligibleForRole(role string, ref *accountingsync.AccountingReferenceObject) bool {
	if role == accountingsync.AccountRoleDeposit && ref.AccountSubType == undepositedFundsSubType {
		return true
	}
	return slices.Contains(roleAccountTypes[role], ref.AccountType)
}

func eligible(t *target, ref *accountingsync.AccountingReferenceObject) bool {
	if ref.Kind != t.TargetType.ProviderKind() || !ref.Usable() {
		return false
	}
	if t.TargetType == accountingsync.TargetAccountRole {
		return eligibleForRole(t.Key, ref)
	}
	return true
}

func score(
	t *target,
	refs []*accountingsync.AccountingReferenceObject,
	index *tokenIndex,
) *accountingsync.Proposal {
	pool := refs
	if index != nil {
		pool = index.candidates(t.Names, t.Identifiers...)
	}

	eligibleCount := 0
	for _, ref := range refs {
		if eligible(t, ref) {
			eligibleCount++
		}
	}

	scored := make([]candidateScore, 0, min(len(pool), indexMaxCandidates))
	for _, ref := range pool {
		if !eligible(t, ref) {
			continue
		}
		scored = append(scored, scoreCandidate(t, ref))
	}

	if _, dedicated := dedicatedTypeRoles[t.Key]; dedicated && eligibleCount == 1 &&
		len(scored) == 1 &&
		t.TargetType == accountingsync.TargetAccountRole {
		only := &scored[0]
		if only.score < scoreOnlyEligible {
			only.score = scoreOnlyEligible
			only.reason = fmt.Sprintf(
				"%s is the only %s account",
				only.ref.Label(),
				only.ref.AccountType,
			)
			only.matchers = append(only.matchers, accountingsync.MatcherSignal{
				Matcher: matcherOnlyEligible,
				Score:   scoreOnlyEligible,
			})
		}
	}

	slices.SortStableFunc(scored, func(a, b candidateScore) int {
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		return cmp.Compare(a.ref.Name, b.ref.Name)
	})

	proposal := &accountingsync.Proposal{
		Source:     accountingsync.MappingSourceSuggested,
		Candidates: make([]accountingsync.MappingCandidate, 0, accountingsync.MaxCandidates),
	}
	for idx := range scored {
		if len(proposal.Candidates) == accountingsync.MaxCandidates {
			break
		}
		if scored[idx].score <= 0 {
			break
		}
		proposal.Candidates = append(proposal.Candidates, accountingsync.MappingCandidate{
			ExternalID: scored[idx].ref.ExternalID,
			Name:       scored[idx].ref.Label(),
			Score:      round4(scored[idx].score),
			Reason:     scored[idx].reason,
		})
	}

	if len(scored) == 0 {
		proposal.Reason = "No usable record of this kind in the accounting system"
		return proposal
	}
	best := scored[0]
	proposal.Matchers = best.matchers
	if best.score < accountingsync.ConfidentConfidence {
		proposal.Reason = "No confident match"
		return proposal
	}

	proposal.ExternalID = best.ref.ExternalID
	proposal.ExternalName = best.ref.Label()
	proposal.Confidence = round4(best.score)
	proposal.Reason = best.reason
	return proposal
}

func scoreCandidate(t *target, ref *accountingsync.AccountingReferenceObject) candidateScore {
	result := candidateScore{ref: ref}
	consider := func(value float64, matcher, reason string) {
		if value <= 0 {
			return
		}
		result.matchers = append(result.matchers, accountingsync.MatcherSignal{
			Matcher: matcher,
			Score:   round4(value),
		})
		if value > result.score {
			result.score = value
			result.reason = reason
		}
	}

	switch t.TargetType {
	case accountingsync.TargetAccountRole, accountingsync.TargetGLAccount:
		if t.Code != "" &&
			strings.EqualFold(strings.TrimSpace(ref.Number), strings.TrimSpace(t.Code)) {
			consider(scoreNumber, matcherNumber, fmt.Sprintf("Account number %s matches", t.Code))
		}
		scoreNames(
			t.Names,
			[]string{ref.Name, ref.FullyQualifiedName},
			stringutils.NameSimilarity,
			consider,
		)
	case accountingsync.TargetAccessorialCharge,
		accountingsync.TargetLineType,
		accountingsync.TargetItemRole:
		if code := stringutils.NormalizeName(t.Code); code != "" {
			switch code {
			case stringutils.NormalizeName(ref.Number):
				consider(
					scoreCode,
					matcherCode,
					fmt.Sprintf("Charge code %s matches the item's SKU", t.Code),
				)
			case stringutils.NormalizeName(ref.Name):
				consider(
					scoreCode,
					matcherCode,
					fmt.Sprintf("Charge code %s matches the item's name", t.Code),
				)
			}
		}
		scoreNames(
			t.Names,
			[]string{ref.Name, ref.Description},
			stringutils.NameSimilarity,
			consider,
		)
	case accountingsync.TargetCustomer, accountingsync.TargetCarrier, accountingsync.TargetDriver:
		scoreParty(t, ref, consider, &result)
	case accountingsync.TargetPaymentTerm:
		if t.DueDays != nil && ref.DueDays != nil && *t.DueDays == *ref.DueDays {
			consider(
				scoreDueDays,
				matcherDueDays,
				fmt.Sprintf("Due in %d days in both systems", *t.DueDays),
			)
		}
		scoreNames(termSynonyms[t.Key], []string{ref.Name}, stringutils.NameSimilarity, consider)
	case accountingsync.TargetPaymentMethod:
		scoreNames(t.Names, []string{ref.Name}, stringutils.NameSimilarity, consider)
	}

	result.score = min(result.score, maxDeterministicScore)
	return result
}

func scoreParty(
	t *target,
	ref *accountingsync.AccountingReferenceObject,
	consider func(float64, string, string),
	result *candidateScore,
) {
	runs := digitRuns(ref.Number)
	for _, identifier := range t.Identifiers {
		digits := stringutils.DigitsOnly(identifier)
		if len(digits) >= minIdentifierDigits && slices.Contains(runs, digits) {
			consider(scoreIdentifier, matcherIdentifier,
				fmt.Sprintf("The account number in the accounting system carries %s", identifier))
		}
	}

	names := []string{ref.Name, ref.CompanyName}
	for _, name := range t.Names {
		for _, other := range names {
			if name == "" || other == "" {
				continue
			}
			if stringutils.NormalizeCompanyName(name) == stringutils.NormalizeCompanyName(other) {
				consider(scoreEqualCompany, matcherName, "Same company name")
			}
		}
	}
	scoreNames(t.Names, names, stringutils.CompanyNameSimilarity, consider)

	if t.PostalCode != "" && ref.PostalCode != "" && result.score > 0 &&
		normalizePostal(t.PostalCode) == normalizePostal(ref.PostalCode) {
		result.matchers = append(result.matchers, accountingsync.MatcherSignal{
			Matcher: matcherPostalCode,
			Score:   postalCodeBonus,
		})
		result.score = min(result.score+postalCodeBonus, maxDeterministicScore)
		result.reason += " and the same postal code"
	}
}

func scoreNames(
	names []string,
	others []string,
	similarity func(a, b string) float64,
	consider func(float64, string, string),
) {
	best, bestOther, bestExact := 0.0, "", false
	for idx, name := range names {
		ceiling := max(nameScoreCeiling-nameRankStep*float64(idx), nameScoreFloor)
		for _, other := range others {
			if name == "" || other == "" {
				continue
			}
			raw := similarity(name, other)
			if value := raw * ceiling; value > best {
				best, bestOther, bestExact = value, other, raw >= 1
			}
		}
	}
	if best <= 0 {
		return
	}
	if bestExact {
		consider(best, matcherName, fmt.Sprintf("Same name as %s", bestOther))
		return
	}
	consider(best, matcherName, fmt.Sprintf("Name is similar to %s", bestOther))
}

func normalizePostal(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func round4(value float64) float64 {
	rounded, err := strconv.ParseFloat(strconv.FormatFloat(value, 'f', 4, 64), 64)
	if err != nil {
		return value
	}
	return rounded
}

type tokenIndex struct {
	refs     []*accountingsync.AccountingReferenceObject
	byToken  map[string][]int
	byDigits map[string][]int
	common   map[string]struct{}
}

func newTokenIndex(refs []*accountingsync.AccountingReferenceObject) *tokenIndex {
	index := &tokenIndex{
		refs:     refs,
		byToken:  make(map[string][]int),
		byDigits: make(map[string][]int),
		common:   make(map[string]struct{}),
	}
	for idx, ref := range refs {
		for _, run := range digitRuns(ref.Number) {
			index.byDigits[run] = append(index.byDigits[run], idx)
		}
		seen := make(map[string]struct{})
		for _, token := range partyTokens(ref.Name, ref.CompanyName) {
			if _, dup := seen[token]; dup {
				continue
			}
			seen[token] = struct{}{}
			index.byToken[token] = append(index.byToken[token], idx)
		}
	}
	if len(refs) >= indexMinRecords {
		limit := max(int(float64(len(refs))*indexCommonShare), 1)
		for token, postings := range index.byToken {
			if len(postings) > limit {
				index.common[token] = struct{}{}
			}
		}
	}
	return index
}

func (i *tokenIndex) candidates(
	names []string,
	identifiers ...string,
) []*accountingsync.AccountingReferenceObject {
	hits := make(map[int]int)
	for _, identifier := range identifiers {
		for _, idx := range i.byDigits[stringutils.DigitsOnly(identifier)] {
			hits[idx] += len(names) + 1
		}
	}
	for _, token := range partyTokens(names...) {
		if _, common := i.common[token]; common {
			continue
		}
		for _, idx := range i.byToken[token] {
			hits[idx]++
		}
	}

	ranked := make([]int, 0, len(hits))
	for idx := range hits {
		ranked = append(ranked, idx)
	}
	slices.SortFunc(ranked, func(a, b int) int {
		if c := cmp.Compare(hits[b], hits[a]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	if len(ranked) > indexMaxCandidates {
		ranked = ranked[:indexMaxCandidates]
	}

	out := make([]*accountingsync.AccountingReferenceObject, 0, len(ranked))
	for _, idx := range ranked {
		out = append(out, i.refs[idx])
	}
	return out
}

func digitRuns(value string) []string {
	runs := strings.FieldsFunc(value, func(r rune) bool { return r < '0' || r > '9' })
	return slices.DeleteFunc(runs, func(run string) bool { return len(run) < minIdentifierDigits })
}

func partyTokens(values ...string) []string {
	tokens := make([]string, 0, len(values)*3)
	for _, value := range values {
		tokens = append(tokens, strings.Fields(stringutils.NormalizeCompanyName(value))...)
	}
	return tokens
}
