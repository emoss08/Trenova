package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staging() *factoring.RemittanceStaging {
	return &factoring.RemittanceStaging{
		DocumentID:  pulid.ID("doc_1"),
		FileName:    "rts-10-07.csv",
		Format:      factoring.RemittanceFormatCsv,
		DefaultKind: factoring.RemittanceLineFunding,
		Headers:     []string{"Invoice", "Advance"},
		Mapping:     map[string]int{"invoiceNumber": 0, "advance": 1},
		At:          sentAt,
	}
}

func parsedRemittance(t *testing.T, lines ...*factoring.RemittanceLine) *factoring.Remittance {
	t.Helper()
	remittance := &factoring.Remittance{ID: "facrem_1", Status: factoring.RemittanceStatusPending}
	require.NoError(t, remittance.MarkParsed(staging(), lines))
	return remittance
}

func TestMarkParsedCountsWhatNeedsAPerson(t *testing.T) {
	t.Parallel()

	item := pendingCandidate("facsit_a", "INV-1", 100_000, 90_000, 3_000).Item
	matched := fundingLine("INV-1", 90_000, nil)
	matched.MatchTo(item, factoring.RemittanceMatchedByAuto, 95, factoring.MatchReasonInvoiceAmount)
	suggested := fundingLine("INV-2", 45_000, nil)
	suggested.Suggest([]*factoring.RemittanceSuggestion{{ItemID: "facsit_b", Score: 40}}, "")
	broken := &factoring.RemittanceLine{ID: "facrml_x", MatchStatus: factoring.RemittanceLineError, Error: "bad"}

	remittance := parsedRemittance(t, matched, suggested, broken)

	assert.Equal(t, factoring.RemittanceStatusParsed, remittance.Status)
	assert.Equal(t, 3, remittance.LineCount)
	assert.Equal(t, 1, remittance.MatchedCount)
	assert.Equal(t, 2, remittance.ExceptionCount)
	assert.Equal(t, int64(135_000), remittance.TotalCashMinor, "an unreadable row adds nothing")
	assert.Equal(t, pulid.ID("facrem_1"), matched.RemittanceID)
	assert.Equal(t, factoring.RemittanceLineSuggested, suggested.MatchStatus)
	assert.Equal(t, 40, suggested.Score)
}

func TestMarkAppliedLeavesTheRemittanceOpenUntilEveryLineIsSettled(t *testing.T) {
	t.Parallel()

	booked := fundingLine("INV-1", 90_000, nil)
	booked.MatchTo(&factoring.SubmissionItem{ID: "facsit_a"}, factoring.RemittanceMatchedByAuto, 95, factoring.MatchReasonInvoiceAmount)
	open := fundingLine("INV-2", 45_000, nil)
	remittance := parsedRemittance(t, booked, open)

	booked.MarkApplied("facres_1")
	remittance.MarkApplied(sentAt, "usr_1")
	assert.Equal(t, factoring.RemittanceStatusPartiallyApplied, remittance.Status)
	assert.Equal(t, 1, remittance.AppliedCount)

	require.Error(t, remittance.Discard(sentAt, "Wrong file"),
		"part of it is booked, so the rest is ignored rather than the file discarded")

	line, err := remittance.EditableLine(open.ID)
	require.NoError(t, err)
	require.NoError(t, line.Ignore("Factor fee adjustment, booked by hand"))
	remittance.SettleIfDone()
	assert.Equal(t, factoring.RemittanceStatusApplied, remittance.Status)

	_, err = remittance.EditableLine(open.ID)
	require.Error(t, err, "an applied remittance is closed")
}

func TestEditableLineRefusesABookedLine(t *testing.T) {
	t.Parallel()

	booked := fundingLine("INV-1", 90_000, nil)
	booked.MatchTo(&factoring.SubmissionItem{ID: "facsit_a"}, factoring.RemittanceMatchedByAuto, 95, factoring.MatchReasonInvoiceAmount)
	booked.MarkApplied("facres_1")
	remittance := parsedRemittance(t, booked, fundingLine("INV-2", 1, nil))
	remittance.MarkApplied(sentAt, "usr_1")

	_, err := remittance.EditableLine(booked.ID)
	require.Error(t, err)
	_, err = remittance.EditableLine("facrml_missing")
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
}

func TestStagingAgainIsRefusedOnceAnythingIsBooked(t *testing.T) {
	t.Parallel()

	remittance := &factoring.Remittance{Status: factoring.RemittanceStatusPartiallyApplied}
	require.Error(t, remittance.MarkNeedsMapping(staging()))
	require.Error(t, remittance.MarkFailed("", "", "boom", sentAt))

	fresh := &factoring.Remittance{Status: factoring.RemittanceStatusFailed}
	require.NoError(t, fresh.MarkNeedsMapping(staging()))
	assert.Equal(t, factoring.RemittanceStatusNeedsMapping, fresh.Status)
	assert.Empty(t, fresh.Error)
}

func TestRequireApplicableNeedsAMatchedLine(t *testing.T) {
	t.Parallel()

	remittance := parsedRemittance(t, fundingLine("INV-1", 1, nil))
	require.Error(t, remittance.RequireApplicable())

	remittance.Lines[0].MatchTo(&factoring.SubmissionItem{ID: "facsit_a"}, factoring.RemittanceMatchedByPerson, 0, factoring.MatchReasonChosenByPerson)
	require.NoError(t, remittance.RequireApplicable())
}

func TestDiscardNeedsAReason(t *testing.T) {
	t.Parallel()

	remittance := parsedRemittance(t, fundingLine("INV-1", 1, nil))
	require.Error(t, remittance.Discard(sentAt, "  "))
	require.NoError(t, remittance.Discard(sentAt, "Duplicate of yesterday's file"))
	assert.Equal(t, factoring.RemittanceStatusDiscarded, remittance.Status)
}

func TestSetKindReopensTheLineForMatching(t *testing.T) {
	t.Parallel()

	line := fundingLine("INV-1", 90_000, nil)
	line.MatchTo(&factoring.SubmissionItem{ID: "facsit_a"}, factoring.RemittanceMatchedByAuto, 95, factoring.MatchReasonInvoiceAmount)
	require.NoError(t, line.SetKind(factoring.RemittanceLineReserveRelease))
	assert.Equal(t, factoring.RemittanceLineUnmatched, line.MatchStatus)
	assert.True(t, line.SubmissionItemID.IsNil())
	require.Error(t, line.SetKind("Bonus"))
}
