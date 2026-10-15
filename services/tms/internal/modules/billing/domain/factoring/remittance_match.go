package factoring

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	AutoMatchScore         = 90
	AutoMatchLead          = 15
	minAmountTolerance     = 100
	minPartialInvoiceRunes = 4
	maxRemittanceAgeDays   = 400
)

type RemittanceCandidate struct {
	Item             *SubmissionItem
	SubmissionNumber string
	SentAt           *int64
	Recourse         bool
}

func NormalizeRemittanceReference(value string) string {
	return stringutils.NormalizeIdentifier(strings.TrimSpace(value))
}

func (k RemittanceLineKind) Accepts(candidate *RemittanceCandidate) bool {
	if candidate == nil || candidate.Item == nil {
		return false
	}
	switch k {
	case RemittanceLineFunding, RemittanceLineRejection:
		return candidate.Item.Status == SubmissionItemStatusPending
	case RemittanceLineReserveRelease:
		return candidate.Item.Status == SubmissionItemStatusFunded
	case RemittanceLineChargeback:
		return candidate.Item.Status == SubmissionItemStatusFunded && candidate.Recourse
	case RemittanceLineOther:
	}
	return false
}

func ScoreRemittanceLine(
	line *RemittanceLine,
	candidate *RemittanceCandidate,
) (int, RemittanceMatchReason) {
	if line == nil || !line.Kind.Accepts(candidate) {
		return 0, ""
	}
	invoiceExact, invoicePartial := invoiceFlags(line, candidate.Item)
	amountExact, amountNear := amountFlags(line, candidate.Item)
	datePlausible := datePlausible(line, candidate)
	return resolveRemittanceScore(
		invoiceExact,
		invoicePartial,
		amountExact,
		amountNear,
		datePlausible,
	)
}

func resolveRemittanceScore(
	invoiceExact, invoicePartial, amountExact, amountNear, datePlausible bool,
) (int, RemittanceMatchReason) {
	switch {
	case invoiceExact && amountExact && datePlausible:
		return 100, MatchReasonInvoiceAmountDate
	case invoiceExact && amountExact:
		return 95, MatchReasonInvoiceAmount
	case invoiceExact && amountNear:
		return 80, MatchReasonInvoiceNearAmount
	case invoicePartial && amountExact:
		return 75, MatchReasonPartialInvoiceAmount
	case invoiceExact:
		return 60, MatchReasonInvoiceOnly
	case invoicePartial && amountNear:
		return 55, MatchReasonPartialInvoiceNear
	case amountExact:
		return 40, MatchReasonAmountOnly
	case amountNear:
		return 20, MatchReasonNearAmountOnly
	}
	return 0, ""
}

func invoiceFlags(line *RemittanceLine, item *SubmissionItem) (exact, partial bool) {
	target := NormalizeRemittanceReference(item.InvoiceNumber)
	if target == "" {
		return false, false
	}
	for _, raw := range []string{line.InvoiceNumber, line.FactorReference} {
		value := NormalizeRemittanceReference(raw)
		if value == "" {
			continue
		}
		if value == target {
			return true, false
		}
		if len([]rune(value)) >= minPartialInvoiceRunes &&
			len([]rune(target)) >= minPartialInvoiceRunes &&
			(strings.Contains(value, target) || strings.Contains(target, value)) {
			partial = true
		}
	}
	return false, partial
}

func amountFlags(line *RemittanceLine, item *SubmissionItem) (exact, near bool) {
	got, want, ok := comparableAmounts(line, item)
	if !ok {
		return false, false
	}
	if got == want {
		return true, true
	}
	tolerance := max(int64(minAmountTolerance), item.ExpectedFeeMinor)
	return false, intutils.AbsDiff(got, want) <= tolerance
}

func comparableAmounts(line *RemittanceLine, item *SubmissionItem) (got, want int64, ok bool) {
	switch line.Kind {
	case RemittanceLineFunding:
		switch {
		case line.AdvanceMinor != nil:
			return *line.AdvanceMinor, item.ExpectedAdvanceMinor, true
		case line.FaceMinor != nil:
			return *line.FaceMinor, item.FaceMinor, true
		case line.CashMinor != nil:
			return *line.CashMinor, item.ExpectedAdvanceMinor, true
		}
	case RemittanceLineRejection:
		if line.FaceMinor != nil {
			return *line.FaceMinor, item.FaceMinor, true
		}
	case RemittanceLineReserveRelease:
		held := heldReserve(item)
		switch {
		case line.CashMinor != nil:
			return *line.CashMinor, held, true
		case line.ReserveMinor != nil:
			return *line.ReserveMinor, held, true
		}
	case RemittanceLineChargeback:
		advance, fee, _ := item.Actuals()
		if line.CashMinor != nil {
			return intutils.Abs(*line.CashMinor), advance + fee, true
		}
		if line.FaceMinor != nil {
			return *line.FaceMinor, item.FaceMinor, true
		}
	case RemittanceLineOther:
	}
	return 0, 0, false
}

func heldReserve(item *SubmissionItem) int64 {
	_, _, reserve := item.Actuals()
	return reserve
}

func datePlausible(line *RemittanceLine, candidate *RemittanceCandidate) bool {
	if line.OccurredOn == nil {
		return false
	}
	on := *line.OccurredOn
	var from *int64
	switch line.Kind {
	case RemittanceLineFunding, RemittanceLineRejection:
		from = candidate.SentAt
	case RemittanceLineReserveRelease, RemittanceLineChargeback:
		from = candidate.Item.FundedOn
	case RemittanceLineOther:
	}
	if from == nil {
		return false
	}
	earliest := *from - timeutils.SecondsPerDay
	latest := *from + int64(maxRemittanceAgeDays)*timeutils.SecondsPerDay
	return on >= earliest && on <= latest
}

func RankRemittanceLine(
	line *RemittanceLine,
	candidates []*RemittanceCandidate,
) []*RemittanceSuggestion {
	ranked := make([]*RemittanceSuggestion, 0, min(len(candidates), 8))
	for _, candidate := range candidates {
		score, reason := ScoreRemittanceLine(line, candidate)
		if score == 0 {
			continue
		}
		ranked = append(ranked, &RemittanceSuggestion{
			ItemID:        candidate.Item.ID,
			SubmissionID:  candidate.Item.SubmissionID,
			InvoiceNumber: candidate.Item.InvoiceNumber,
			Score:         score,
			Reason:        reason,
		})
	}
	slices.SortStableFunc(ranked, func(a, b *RemittanceSuggestion) int {
		if byScore := cmp.Compare(b.Score, a.Score); byScore != 0 {
			return byScore
		}
		return cmp.Compare(a.ItemID.String(), b.ItemID.String())
	})
	return ranked
}

func ShouldAutoMatch(suggestions []*RemittanceSuggestion) bool {
	if len(suggestions) == 0 || suggestions[0].Score < AutoMatchScore {
		return false
	}
	if len(suggestions) == 1 {
		return true
	}
	return suggestions[0].Score-suggestions[1].Score >= AutoMatchLead
}

type claimKey struct {
	item pulid.ID
	kind RemittanceLineKind
}

type rankedLine struct {
	line        *RemittanceLine
	suggestions []*RemittanceSuggestion
}

func MatchRemittance(lines []*RemittanceLine, candidates []*RemittanceCandidate) {
	byItem := make(map[pulid.ID]*RemittanceCandidate, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.Item != nil {
			byItem[candidate.Item.ID] = candidate
		}
	}
	claimed := settledClaims(lines)
	for _, entry := range rankOpenLines(lines, candidates) {
		assignLine(entry, byItem, claimed)
	}
}

func settledClaims(lines []*RemittanceLine) map[claimKey]struct{} {
	claimed := make(map[claimKey]struct{}, len(lines))
	for _, line := range lines {
		if line == nil || line.Matchable() {
			continue
		}
		if line.MatchStatus == RemittanceLineMatched || line.MatchStatus == RemittanceLineApplied {
			claimed[claimKey{item: line.SubmissionItemID, kind: line.Kind}] = struct{}{}
		}
	}
	return claimed
}

func rankOpenLines(lines []*RemittanceLine, candidates []*RemittanceCandidate) []rankedLine {
	open := make([]rankedLine, 0, len(lines))
	for _, line := range lines {
		if line == nil || !line.Matchable() {
			continue
		}
		if !line.Kind.Books() {
			line.Suggest(nil, MatchReasonNotBookedFromRemittance)
			continue
		}
		open = append(
			open,
			rankedLine{line: line, suggestions: RankRemittanceLine(line, candidates)},
		)
	}
	slices.SortStableFunc(open, func(a, b rankedLine) int {
		return cmp.Compare(topScore(b.suggestions), topScore(a.suggestions))
	})
	return open
}

func assignLine(
	entry rankedLine,
	byItem map[pulid.ID]*RemittanceCandidate,
	claimed map[claimKey]struct{},
) {
	line := entry.line
	available := make([]*RemittanceSuggestion, 0, len(entry.suggestions))
	takenBest := false
	for idx, suggestion := range entry.suggestions {
		if _, taken := claimed[claimKey{item: suggestion.ItemID, kind: line.Kind}]; taken {
			takenBest = takenBest || idx == 0
			continue
		}
		available = append(available, suggestion)
	}
	if !takenBest && ShouldAutoMatch(entry.suggestions) {
		best := entry.suggestions[0]
		line.MatchTo(byItem[best.ItemID].Item, RemittanceMatchedByAuto, best.Score, best.Reason)
		line.Suggestions = trimSuggestions(available)
		claimed[claimKey{item: best.ItemID, kind: line.Kind}] = struct{}{}
		return
	}
	reason := RemittanceMatchReason("")
	switch {
	case takenBest:
		reason = MatchReasonClaimedByAnotherLine
	case len(available) == 0:
		reason = MatchReasonNoOpenItem
	}
	line.Suggest(available, reason)
}

func topScore(suggestions []*RemittanceSuggestion) int {
	if len(suggestions) == 0 {
		return 0
	}
	return suggestions[0].Score
}
