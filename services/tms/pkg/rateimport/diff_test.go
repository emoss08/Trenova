package rateimport_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rule(laneKey, rate string) *rateagreement.RateAgreementRule {
	return &rateagreement.RateAgreementRule{
		ID:          pulid.MustNew("ragr_"),
		LaneKey:     laneKey,
		Label:       laneKey,
		RatingBasis: rateagreement.RatingBasisPerMile,
		Rate:        decimal.NewNullDecimal(decimal.RequireFromString(rate)),
	}
}

func kindsByLane(changes []rateimport.Change) map[string]rateimport.ChangeKind {
	byLane := make(map[string]rateimport.ChangeKind, len(changes))
	for _, change := range changes {
		byLane[change.LaneKey] = change.Kind
	}

	return byLane
}

// A lane the sheet introduces is new business. Somebody reviewing a dry run
// wants to see it before it starts pricing shipments.
func TestDiff_ALaneOnlyInTheSheetIsAdded(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(nil, []*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")})

	require.Len(t, changes, 1)
	assert.Equal(t, rateimport.ChangeKindAdded, changes[0].Kind)
}

// A lane in the contract that the sheet does not mention would stop pricing on
// commit. That is the single most dangerous thing a rate sheet can do quietly,
// so it is reported as loudly as a change.
func TestDiff_ALaneOnlyInTheContractIsRemoved(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff([]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")}, nil)

	require.Len(t, changes, 1)
	assert.Equal(t, rateimport.ChangeKindRemoved, changes[0].Kind)
}

func TestDiff_ALaneWithADifferentRateIsChanged(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.35")},
	)

	require.Len(t, changes, 1)
	assert.Equal(t, rateimport.ChangeKindChanged, changes[0].Kind)
}

// A monthly sheet is mostly the same lanes at the same rates. Reporting every
// one of them as a change would bury the handful that moved.
func TestDiff_ALaneAtTheSameRateIsUnchanged(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
	)

	require.Len(t, changes, 1)
	assert.Equal(t, rateimport.ChangeKindUnchanged, changes[0].Kind)
}

// "This went up" is not reviewable. "This went from 2.10 to 2.35" is.
func TestDiff_AChangedLaneSaysWhatMovedAndFromWhatToWhat(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.35")},
	)

	require.Len(t, changes[0].Fields, 1)
	assert.Equal(t, "rate", changes[0].Fields[0].Field)
	assert.Equal(t, "2.1", changes[0].Fields[0].Before)
	assert.Equal(t, "2.35", changes[0].Fields[0].After)
}

// Several terms can move on one lane, and somebody approving a sheet needs all
// of them rather than whichever was compared first.
func TestDiff_AChangedLaneReportsEveryTermThatMoved(t *testing.T) {
	t.Parallel()

	existing := rule("ST:il>ST:ga", "2.10")
	existing.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString("450"))

	incoming := rule("ST:il>ST:ga", "2.35")
	incoming.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString("500"))
	incoming.RatingBasis = rateagreement.RatingBasisFlat

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{existing},
		[]*rateagreement.RateAgreementRule{incoming},
	)

	fields := make([]string, 0, len(changes[0].Fields))
	for _, moved := range changes[0].Fields {
		fields = append(fields, moved.Field)
	}

	assert.ElementsMatch(t, []string{"rate", "minCharge", "ratingBasis"}, fields)
}

// A term the contract had and the sheet drops is a change, not an absence. A
// minimum charge silently disappearing is exactly the kind of thing a dry run
// exists to catch.
func TestDiff_ATermTheSheetDropsIsReportedAsAChange(t *testing.T) {
	t.Parallel()

	existing := rule("ST:il>ST:ga", "2.10")
	existing.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString("450"))

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{existing},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
	)

	require.Len(t, changes[0].Fields, 1)
	assert.Equal(t, "minCharge", changes[0].Fields[0].Field)
	assert.Equal(t, "450", changes[0].Fields[0].Before)
	assert.Empty(t, changes[0].Fields[0].After)
}

// A contract with no minimum charge and one with a minimum of zero are
// different contracts, and reading absence as zero would hide a sheet that
// introduces a floor of nothing.
func TestDiff_ATermTheSheetAddsAtZeroIsStillAChange(t *testing.T) {
	t.Parallel()

	incoming := rule("ST:il>ST:ga", "2.10")
	incoming.MinCharge = decimal.NewNullDecimal(decimal.Zero)

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
		[]*rateagreement.RateAgreementRule{incoming},
	)

	require.Len(t, changes[0].Fields, 1)
	assert.Equal(t, "minCharge", changes[0].Fields[0].Field)
	assert.Empty(t, changes[0].Fields[0].Before)
	assert.Equal(t, "0", changes[0].Fields[0].After)
}

// Two rows in one sheet for the same lane is a mistake in the sheet, and
// importing both would leave the contract with two rules the engine has to
// break a tie between.
func TestDiff_TwoSheetRowsForOneLaneAreReportedAsADuplicate(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(nil, []*rateagreement.RateAgreementRule{
		rule("ST:il>ST:ga", "2.10"),
		rule("ST:il>ST:ga", "2.35"),
	})

	kinds := make([]rateimport.ChangeKind, 0, len(changes))
	for _, change := range changes {
		kinds = append(kinds, change.Kind)
	}

	assert.Contains(t, kinds, rateimport.ChangeKindDuplicate)
}

// The dry run is read top to bottom, and the order has to be the order somebody
// checks in: what disappears first, then what is new, then what moved.
func TestDiff_IsOrderedWithTheRiskiestChangesFirst(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{
			rule("ST:il>ST:ga", "2.10"),
			rule("ST:tx>ST:ca", "1.90"),
			rule("ST:oh>ST:pa", "2.00"),
		},
		[]*rateagreement.RateAgreementRule{
			rule("ST:il>ST:ga", "2.35"),
			rule("ST:oh>ST:pa", "2.00"),
			rule("ST:fl>ST:ny", "2.50"),
		},
	)

	require.Len(t, changes, 4)
	assert.Equal(t, rateimport.ChangeKindRemoved, changes[0].Kind)
	assert.Equal(t, rateimport.ChangeKindAdded, changes[1].Kind)
	assert.Equal(t, rateimport.ChangeKindChanged, changes[2].Kind)
	assert.Equal(t, rateimport.ChangeKindUnchanged, changes[3].Kind)
}

// The summary is what somebody reads before deciding whether to read the rest.
func TestSummarize_CountsEachKind(t *testing.T) {
	t.Parallel()

	summary := rateimport.Summarize(rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10"), rule("ST:tx>ST:ca", "1.90")},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.35"), rule("ST:fl>ST:ny", "2.50")},
	))

	assert.Equal(t, 1, summary.Added)
	assert.Equal(t, 1, summary.Changed)
	assert.Equal(t, 1, summary.Removed)
	assert.Equal(t, 0, summary.Unchanged)
}

// A sheet that changes nothing is worth saying out loud: it usually means
// somebody uploaded last month's file again.
func TestSummarize_ASheetThatChangesNothingSaysSo(t *testing.T) {
	t.Parallel()

	summary := rateimport.Summarize(rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10")},
	))

	assert.False(t, summary.ChangesAnything())
	assert.Equal(t, 1, summary.Unchanged)
}

func TestDiff_MatchesLanesByKeyRatherThanByPosition(t *testing.T) {
	t.Parallel()

	changes := rateimport.Diff(
		[]*rateagreement.RateAgreementRule{rule("ST:il>ST:ga", "2.10"), rule("ST:tx>ST:ca", "1.90")},
		[]*rateagreement.RateAgreementRule{rule("ST:tx>ST:ca", "1.90"), rule("ST:il>ST:ga", "2.10")},
	)

	assert.Equal(t, map[string]rateimport.ChangeKind{
		"ST:il>ST:ga": rateimport.ChangeKindUnchanged,
		"ST:tx>ST:ca": rateimport.ChangeKindUnchanged,
	}, kindsByLane(changes))
}
