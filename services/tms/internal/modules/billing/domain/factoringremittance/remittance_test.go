package factoringremittance_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staging() *factoringremittance.RemittanceStaging {
	return &factoringremittance.RemittanceStaging{
		DocumentID:  pulid.ID("doc_1"),
		FileName:    "rts-10-07.csv",
		Format:      factoringremittance.RemittanceFormatCsv,
		DefaultKind: factoringremittance.RemittanceLineFunding,
		Headers:     []string{"Invoice", "Advance"},
		Mapping:     map[string]int{"invoiceNumber": 0, "advance": 1},
		At:          sentAt,
	}
}

func parsedRemittance(
	t *testing.T,
	lines ...*factoringremittance.RemittanceLine,
) *factoringremittance.Remittance {
	t.Helper()
	remittance := &factoringremittance.Remittance{
		ID:     "facrem_1",
		Status: factoringremittance.RemittanceStatusPending,
	}
	require.NoError(t, remittance.MarkParsed(staging(), lines))
	return remittance
}

func TestMarkParsedCountsWhatNeedsAPerson(t *testing.T) {
	t.Parallel()

	item := pendingCandidate("facsit_a", "INV-1", 100_000, 90_000, 3_000).Item
	matched := fundingLine("INV-1", 90_000, nil)
	matched.MatchTo(
		item,
		factoringremittance.RemittanceMatchedByAuto,
		95,
		factoringremittance.MatchReasonInvoiceAmount,
	)
	suggested := fundingLine("INV-2", 45_000, nil)
	suggested.Suggest(
		[]*factoringremittance.RemittanceSuggestion{{ItemID: "facsit_b", Score: 40}},
		"",
	)
	broken := &factoringremittance.RemittanceLine{
		ID:          "facrml_x",
		MatchStatus: factoringremittance.RemittanceLineError,
		Error:       "bad",
	}

	remittance := parsedRemittance(t, matched, suggested, broken)

	assert.Equal(t, factoringremittance.RemittanceStatusParsed, remittance.Status)
	assert.Equal(t, 3, remittance.LineCount)
	assert.Equal(t, 1, remittance.MatchedCount)
	assert.Equal(t, 2, remittance.ExceptionCount)
	assert.Equal(t, int64(135_000), remittance.TotalCashMinor, "an unreadable row adds nothing")
	assert.Equal(t, pulid.ID("facrem_1"), matched.RemittanceID)
	assert.Equal(t, factoringremittance.RemittanceLineSuggested, suggested.MatchStatus)
	assert.Equal(t, 40, suggested.Score)
}

func TestMarkAppliedLeavesTheRemittanceOpenUntilEveryLineIsSettled(t *testing.T) {
	t.Parallel()

	booked := fundingLine("INV-1", 90_000, nil)
	booked.MatchTo(
		&factoringsubmission.SubmissionItem{ID: "facsit_a"},
		factoringremittance.RemittanceMatchedByAuto,
		95,
		factoringremittance.MatchReasonInvoiceAmount,
	)
	open := fundingLine("INV-2", 45_000, nil)
	remittance := parsedRemittance(t, booked, open)

	booked.MarkApplied("facres_1")
	remittance.MarkApplied(sentAt, "usr_1")
	assert.Equal(t, factoringremittance.RemittanceStatusPartiallyApplied, remittance.Status)
	assert.Equal(t, 1, remittance.AppliedCount)

	require.Error(t, remittance.Discard(sentAt, "Wrong file"),
		"part of it is booked, so the rest is ignored rather than the file discarded")

	line, err := remittance.EditableLine(open.ID)
	require.NoError(t, err)
	require.NoError(t, line.Ignore("Factor fee adjustment, booked by hand"))
	remittance.SettleIfDone()
	assert.Equal(t, factoringremittance.RemittanceStatusApplied, remittance.Status)

	_, err = remittance.EditableLine(open.ID)
	require.Error(t, err, "an applied remittance is closed")
}

func TestEditableLineRefusesABookedLine(t *testing.T) {
	t.Parallel()

	booked := fundingLine("INV-1", 90_000, nil)
	booked.MatchTo(
		&factoringsubmission.SubmissionItem{ID: "facsit_a"},
		factoringremittance.RemittanceMatchedByAuto,
		95,
		factoringremittance.MatchReasonInvoiceAmount,
	)
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

	remittance := &factoringremittance.Remittance{
		Status: factoringremittance.RemittanceStatusPartiallyApplied,
	}
	require.Error(t, remittance.MarkNeedsMapping(staging()))
	require.Error(t, remittance.MarkFailed("", "", "boom", sentAt))

	fresh := &factoringremittance.Remittance{Status: factoringremittance.RemittanceStatusFailed}
	require.NoError(t, fresh.MarkNeedsMapping(staging()))
	assert.Equal(t, factoringremittance.RemittanceStatusNeedsMapping, fresh.Status)
	assert.Empty(t, fresh.Error)
}

func TestRequireApplicableNeedsAMatchedLine(t *testing.T) {
	t.Parallel()

	remittance := parsedRemittance(t, fundingLine("INV-1", 1, nil))
	require.Error(t, remittance.RequireApplicable())

	remittance.Lines[0].MatchTo(
		&factoringsubmission.SubmissionItem{ID: "facsit_a"},
		factoringremittance.RemittanceMatchedByPerson,
		0,
		factoringremittance.MatchReasonChosenByPerson,
	)
	require.NoError(t, remittance.RequireApplicable())
}

func TestDiscardNeedsAReason(t *testing.T) {
	t.Parallel()

	remittance := parsedRemittance(t, fundingLine("INV-1", 1, nil))
	require.Error(t, remittance.Discard(sentAt, "  "))
	require.NoError(t, remittance.Discard(sentAt, "Duplicate of yesterday's file"))
	assert.Equal(t, factoringremittance.RemittanceStatusDiscarded, remittance.Status)
}

func TestSetKindReopensTheLineForMatching(t *testing.T) {
	t.Parallel()

	line := fundingLine("INV-1", 90_000, nil)
	line.MatchTo(
		&factoringsubmission.SubmissionItem{ID: "facsit_a"},
		factoringremittance.RemittanceMatchedByAuto,
		95,
		factoringremittance.MatchReasonInvoiceAmount,
	)
	require.NoError(t, line.SetKind(factoringremittance.RemittanceLineReserveRelease))
	assert.Equal(t, factoringremittance.RemittanceLineUnmatched, line.MatchStatus)
	assert.True(t, line.SubmissionItemID.IsNil())
	require.Error(t, line.SetKind("Bonus"))
}
