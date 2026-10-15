package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	day    = int64(86400)
	sentAt = int64(1_790_000_000)
)

func ptr(v int64) *int64 { return &v }

func pendingCandidate(id, invoice string, face, advance, fee int64) *factoring.RemittanceCandidate {
	at := sentAt
	return &factoring.RemittanceCandidate{
		Item: &factoring.SubmissionItem{
			ID:                   pulid.ID(id),
			SubmissionID:         pulid.ID("facsub_1"),
			InvoiceNumber:        invoice,
			FaceMinor:            face,
			ExpectedAdvanceMinor: advance,
			ExpectedFeeMinor:     fee,
			ExpectedReserveMinor: face - advance - fee,
			Status:               factoring.SubmissionItemStatusPending,
		},
		SubmissionNumber: "FS2610-00001",
		SentAt:           &at,
		Recourse:         true,
	}
}

func fundedCandidate(id, invoice string, recourse bool) *factoring.RemittanceCandidate {
	candidate := pendingCandidate(id, invoice, 100_000, 90_000, 3_000)
	funded := sentAt + day
	candidate.Item.Status = factoring.SubmissionItemStatusFunded
	candidate.Item.FundedOn = &funded
	candidate.Item.AdvanceMinor = ptr(90_000)
	candidate.Item.FeeMinor = ptr(3_000)
	candidate.Item.ReserveMinor = ptr(7_000)
	candidate.Recourse = recourse
	return candidate
}

func fundingLine(invoice string, advance int64, on *int64) *factoring.RemittanceLine {
	return &factoring.RemittanceLine{
		ID:            pulid.ID("facrml_" + invoice),
		Kind:          factoring.RemittanceLineFunding,
		InvoiceNumber: invoice,
		AdvanceMinor:  ptr(advance),
		OccurredOn:    on,
		MatchStatus:   factoring.RemittanceLineUnmatched,
	}
}

func TestScoreRemittanceLineCopiesTheBankReceiptBands(t *testing.T) {
	t.Parallel()

	candidate := pendingCandidate("facsit_1", "INV-1001", 100_000, 90_000, 3_000)
	inWindow := ptr(sentAt + 2*day)
	beforeSending := ptr(sentAt - 5*day)

	cases := []struct {
		name   string
		line   *factoring.RemittanceLine
		score  int
		reason factoring.RemittanceMatchReason
	}{
		{"invoice, amount and date", fundingLine("inv 1001", 90_000, inWindow), 100, factoring.MatchReasonInvoiceAmountDate},
		{"invoice and amount, date before the schedule went", fundingLine("INV-1001", 90_000, beforeSending), 95, factoring.MatchReasonInvoiceAmount},
		{"invoice and amount, no date", fundingLine("INV1001", 90_000, nil), 95, factoring.MatchReasonInvoiceAmount},
		{"invoice and an advance within the fee", fundingLine("INV-1001", 88_500, inWindow), 80, factoring.MatchReasonInvoiceNearAmount},
		{"part of the invoice number and the amount", fundingLine("1001", 90_000, inWindow), 75, factoring.MatchReasonPartialInvoiceAmount},
		{"invoice only", fundingLine("INV-1001", 50_000, inWindow), 60, factoring.MatchReasonInvoiceOnly},
		{"part of the invoice and a near amount", fundingLine("1001", 89_000, inWindow), 55, factoring.MatchReasonPartialInvoiceNear},
		{"amount only", fundingLine("ZZ-9", 90_000, inWindow), 40, factoring.MatchReasonAmountOnly},
		{"near amount only", fundingLine("ZZ-9", 91_000, inWindow), 20, factoring.MatchReasonNearAmountOnly},
		{"nothing in common", fundingLine("ZZ-9", 10_000, inWindow), 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			score, reason := factoring.ScoreRemittanceLine(tc.line, candidate)
			assert.Equal(t, tc.score, score)
			assert.Equal(t, tc.reason, reason)
		})
	}
}

func TestScoreRemittanceLineMatchesFaceWhenTheReportHasNoAdvance(t *testing.T) {
	t.Parallel()

	line := &factoring.RemittanceLine{
		Kind:          factoring.RemittanceLineFunding,
		InvoiceNumber: "INV-1001",
		FaceMinor:     ptr(100_000),
	}
	score, _ := factoring.ScoreRemittanceLine(line, pendingCandidate("facsit_1", "INV-1001", 100_000, 90_000, 3_000))
	assert.Equal(t, 95, score)
}

func TestScoreRemittanceLineOnlyOffersItemsTheMovementCanApplyTo(t *testing.T) {
	t.Parallel()

	release := &factoring.RemittanceLine{
		Kind:          factoring.RemittanceLineReserveRelease,
		InvoiceNumber: "INV-2",
		CashMinor:     ptr(7_000),
		OccurredOn:    ptr(sentAt + 30*day),
	}
	score, reason := factoring.ScoreRemittanceLine(release, fundedCandidate("facsit_2", "INV-2", true))
	assert.Equal(t, 100, score, "a release compares its cash with the reserve the factor holds")
	assert.Equal(t, factoring.MatchReasonInvoiceAmountDate, reason)

	score, _ = factoring.ScoreRemittanceLine(release, pendingCandidate("facsit_3", "INV-2", 100_000, 90_000, 3_000))
	assert.Zero(t, score, "an unfunded invoice holds no reserve to release")

	chargeback := &factoring.RemittanceLine{
		Kind:          factoring.RemittanceLineChargeback,
		InvoiceNumber: "INV-2",
		CashMinor:     ptr(-93_000),
	}
	score, _ = factoring.ScoreRemittanceLine(chargeback, fundedCandidate("facsit_2", "INV-2", true))
	assert.Equal(t, 95, score, "a chargeback repays the advance and the fee, whichever way the report signs it")

	score, _ = factoring.ScoreRemittanceLine(chargeback, fundedCandidate("facsit_2", "INV-2", false))
	assert.Zero(t, score, "a non-recourse factor cannot charge an invoice back")

	other := &factoring.RemittanceLine{Kind: factoring.RemittanceLineOther, InvoiceNumber: "INV-2", CashMinor: ptr(7_000)}
	score, _ = factoring.ScoreRemittanceLine(other, fundedCandidate("facsit_2", "INV-2", true))
	assert.Zero(t, score)
}

func TestShouldAutoMatchNeedsAClearWinner(t *testing.T) {
	t.Parallel()

	suggest := func(scores ...int) []*factoring.RemittanceSuggestion {
		out := make([]*factoring.RemittanceSuggestion, 0, len(scores))
		for _, score := range scores {
			out = append(out, &factoring.RemittanceSuggestion{Score: score})
		}
		return out
	}

	assert.True(t, factoring.ShouldAutoMatch(suggest(95)))
	assert.True(t, factoring.ShouldAutoMatch(suggest(100, 85)))
	assert.False(t, factoring.ShouldAutoMatch(suggest(95, 81)), "a lead under 15 points is a person's call")
	assert.False(t, factoring.ShouldAutoMatch(suggest(80)))
	assert.False(t, factoring.ShouldAutoMatch(nil))
}

func TestMatchRemittanceAutoMatchesExactLinesAndRanksTheRest(t *testing.T) {
	t.Parallel()

	candidates := []*factoring.RemittanceCandidate{
		pendingCandidate("facsit_a", "INV-1001", 100_000, 90_000, 3_000),
		pendingCandidate("facsit_b", "INV-1002", 50_000, 45_000, 1_500),
		pendingCandidate("facsit_c", "INV-1003", 50_000, 45_000, 1_500),
	}
	on := ptr(sentAt + day)
	exact := fundingLine("INV-1001", 90_000, on)
	near := fundingLine("INV-1002", 44_200, on)
	amountOnly := fundingLine("UNKNOWN", 45_000, on)
	duplicate := fundingLine("INV-1001", 90_000, on)
	duplicate.ID = "facrml_dup"
	adjustment := &factoring.RemittanceLine{ID: "facrml_adj", Kind: factoring.RemittanceLineOther, CashMinor: ptr(-2_500)}

	lines := []*factoring.RemittanceLine{near, exact, amountOnly, duplicate, adjustment}
	factoring.MatchRemittance(lines, candidates)

	assert.Equal(t, factoring.RemittanceLineMatched, exact.MatchStatus)
	assert.Equal(t, pulid.ID("facsit_a"), exact.SubmissionItemID)
	assert.Equal(t, factoring.RemittanceMatchedByAuto, exact.MatchedBy)
	assert.Equal(t, 100, exact.Score)

	assert.Equal(t, factoring.RemittanceLineSuggested, near.MatchStatus, "80 is below the automatic bar")
	require.NotEmpty(t, near.Suggestions)
	assert.Equal(t, pulid.ID("facsit_b"), near.Suggestions[0].ItemID)

	assert.Equal(t, factoring.RemittanceLineSuggested, amountOnly.MatchStatus)
	require.Len(t, amountOnly.Suggestions, 2, "two invoices carry that advance")

	assert.Equal(t, factoring.RemittanceLineUnmatched, duplicate.MatchStatus)
	assert.Equal(t, factoring.MatchReasonClaimedByAnotherLine, duplicate.Reason)

	assert.Equal(t, factoring.RemittanceLineUnmatched, adjustment.MatchStatus)
	assert.Equal(t, factoring.MatchReasonNotBookedFromRemittance, adjustment.Reason)
}

func TestMatchRemittanceKeepsAPersonsChoice(t *testing.T) {
	t.Parallel()

	candidates := []*factoring.RemittanceCandidate{
		pendingCandidate("facsit_a", "INV-1001", 100_000, 90_000, 3_000),
	}
	chosen := fundingLine("SOMETHING", 12_000, nil)
	chosen.MatchTo(candidates[0].Item, factoring.RemittanceMatchedByPerson, 0, factoring.MatchReasonChosenByPerson)
	rival := fundingLine("INV-1001", 90_000, ptr(sentAt))

	factoring.MatchRemittance([]*factoring.RemittanceLine{chosen, rival}, candidates)

	assert.Equal(t, factoring.RemittanceLineMatched, chosen.MatchStatus)
	assert.Equal(t, factoring.RemittanceMatchedByPerson, chosen.MatchedBy)
	assert.Equal(t, factoring.RemittanceLineUnmatched, rival.MatchStatus)
	assert.Equal(t, factoring.MatchReasonClaimedByAnotherLine, rival.Reason)
}
