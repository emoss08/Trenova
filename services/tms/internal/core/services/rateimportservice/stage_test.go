package rateimportservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sheet(rows ...[]string) *pkgrateimport.Sheet {
	return &pkgrateimport.Sheet{
		Headers:      []string{"Origin State", "Destination State", "Rate Type", "Rate"},
		Rows:         rows,
		FirstDataRow: 2,
	}
}

// standardTemplates stands in for the organization's seeded formula templates,
// which is what staging resolves a sheet's rate types against. Existing rules
// price through the same per-mile template, so a lane the sheet restates
// identically compares as unchanged.
var standardTemplates = map[string]pulid.ID{
	formulatemplate.StandardFlatRate: pulid.MustNew("ft_"),
	formulatemplate.StandardPerMile:  pulid.MustNew("ft_"),
}

func resolveTemplate(name string) (pulid.ID, bool) {
	id, ok := standardTemplates[name]

	return id, ok
}

func existingRule(laneKey, rate string) *rateagreement.RateAgreementRule {
	templateID := standardTemplates[formulatemplate.StandardPerMile]

	return &rateagreement.RateAgreementRule{
		ID:                pulid.MustNew("ragr_"),
		LaneKey:           laneKey,
		Label:             laneKey,
		FormulaTemplateID: &templateID,
		Rate:              decimal.NewNullDecimal(decimal.RequireFromString(rate)),
	}
}

// Every row is read, and the ones that would not read carry their reason. A
// sheet that stopped at the first bad row would take as many uploads to fix as
// it has mistakes.
func TestStage_ReadsEveryRowAndKeepsTheOnesThatFailed(t *testing.T) {
	t.Parallel()

	staged, err := stage(sheet(
		[]string{"IL", "GA", "PerMile", "2.35"},
		[]string{"TX", "CA", "per furlong", "1.90"},
		[]string{"OH", "PA", "PerMile", "2.00"},
	), nil, resolveTemplate)

	require.NoError(t, err)
	require.Len(t, staged.Rows, 3)
	assert.Empty(t, staged.Rows[0].Error)
	assert.NotEmpty(t, staged.Rows[1].Error)
	assert.Empty(t, staged.Rows[2].Error)
	assert.Equal(t, 1, staged.ErrorCount)
}

// The row number has to be the line in the person's own spreadsheet, or every
// error message sends them off to count lines.
func TestStage_NumbersRowsAsTheyAppearInTheFile(t *testing.T) {
	t.Parallel()

	staged, err := stage(sheet(
		[]string{"IL", "GA", "PerMile", "2.35"},
		[]string{"TX", "CA", "PerMile", "1.90"},
	), nil, resolveTemplate)

	require.NoError(t, err)
	assert.Equal(t, 2, staged.Rows[0].RowNumber)
	assert.Equal(t, 3, staged.Rows[1].RowNumber)
}

// A row that would not read contributes nothing to the diff. Counting it as a
// removal would tell somebody a lane is disappearing when the truth is that one
// line of their sheet has a typo.
func TestStage_AFailedRowDoesNotLookLikeARemoval(t *testing.T) {
	t.Parallel()

	staged, err := stage(
		sheet([]string{"IL", "GA", "per furlong", "2.35"}),
		[]*rateagreement.RateAgreementRule{existingRule("ST:IL>ST:GA", "2.10")},
		resolveTemplate,
	)

	require.NoError(t, err)

	kinds := make([]pkgrateimport.ChangeKind, 0, len(staged.Changes))
	for _, change := range staged.Changes {
		kinds = append(kinds, change.Kind)
	}

	assert.NotContains(t, kinds, pkgrateimport.ChangeKindRemoved,
		"a typo in one row must not read as a lane being withdrawn")
}

func TestStage_DiffsWhatReadAgainstTheContract(t *testing.T) {
	t.Parallel()

	staged, err := stage(
		sheet([]string{"IL", "GA", "PerMile", "2.35"}),
		[]*rateagreement.RateAgreementRule{existingRule("ST:IL>ST:GA", "2.10")},
		resolveTemplate,
	)

	require.NoError(t, err)
	require.NotNil(t, staged.Summary)
	assert.Equal(t, 1, staged.Summary.Changed)
}

// A sheet whose columns cannot be placed is refused before any row is read:
// staging thousands of rows against a mapping that cannot work wastes
// everybody's time and fills the table with rows nobody will look at.
func TestStage_ASheetWithNoUsableMappingIsRefusedUpFront(t *testing.T) {
	t.Parallel()

	unusable := &pkgrateimport.Sheet{
		Headers:      []string{"Sales Rep", "Notes"},
		Rows:         [][]string{{"Dana", "call first"}},
		FirstDataRow: 2,
	}

	_, err := stage(unusable, nil, resolveTemplate)

	require.Error(t, err)
}

// Columns nothing was read from are reported, because a column silently ignored
// is how a sheet imports looking complete while a discount nobody noticed never
// made it in.
func TestStage_ReportsTheColumnsItReadNothingFrom(t *testing.T) {
	t.Parallel()

	withExtra := &pkgrateimport.Sheet{
		Headers:      []string{"Origin State", "Destination State", "Rate", "Sales Rep"},
		Rows:         [][]string{{"IL", "GA", "2.35", "Dana"}},
		FirstDataRow: 2,
	}

	staged, err := stage(withExtra, nil, resolveTemplate)

	require.NoError(t, err)
	assert.Equal(t, []string{"Sales Rep"}, staged.UnmappedHeaders)
}

// Trailing blank rows are what a spreadsheet produces, and staging them as
// failures would report a clean sheet as full of errors.
func TestStage_BlankRowsAreNotStagedAtAll(t *testing.T) {
	t.Parallel()

	staged, err := stage(sheet(
		[]string{"IL", "GA", "PerMile", "2.35"},
		[]string{"", "", "", ""},
	), nil, resolveTemplate)

	require.NoError(t, err)
	assert.Len(t, staged.Rows, 1)
	assert.Zero(t, staged.ErrorCount)
}

// Committing must apply what the review showed, which means the rules it
// supersedes are the ones the diff named — not everything the contract has.
func TestCommitPlan_SupersedesOnlyWhatTheDiffSaidWouldChange(t *testing.T) {
	t.Parallel()

	changed := existingRule("ST:IL>ST:GA", "2.10")
	untouched := existingRule("ST:OH>ST:PA", "2.00")

	// The sheet restates OH→PA at exactly its current rate, so it is genuinely
	// untouched rather than simply absent.
	staged, err := stage(
		sheet(
			[]string{"IL", "GA", "PerMile", "2.35"},
			[]string{"OH", "PA", "PerMile", "2.00"},
		),
		[]*rateagreement.RateAgreementRule{changed, untouched},
		resolveTemplate,
	)
	require.NoError(t, err)

	plan := commitPlan(staged)

	assert.Contains(t, plan.SupersededIDs, changed.ID)
	assert.NotContains(t, plan.SupersededIDs, untouched.ID,
		"a lane the sheet restated identically must not be rewritten")
}

// A lane the sheet does not mention is withdrawn on commit, which is what the
// dry run warned about. Leaving it in place would make the review a lie.
func TestCommitPlan_WithdrawsTheLanesTheSheetDropped(t *testing.T) {
	t.Parallel()

	dropped := existingRule("ST:TX>ST:CA", "1.90")

	staged, err := stage(
		sheet([]string{"IL", "GA", "PerMile", "2.35"}),
		[]*rateagreement.RateAgreementRule{dropped},
		resolveTemplate,
	)
	require.NoError(t, err)

	plan := commitPlan(staged)

	assert.Contains(t, plan.SupersededIDs, dropped.ID)
}

// Only the rows that read become rules. Committing a failed row would write a
// lane nobody could price.
func TestCommitPlan_CarriesOnlyTheRowsThatRead(t *testing.T) {
	t.Parallel()

	staged, err := stage(sheet(
		[]string{"IL", "GA", "PerMile", "2.35"},
		[]string{"TX", "CA", "per furlong", "1.90"},
	), nil, resolveTemplate)
	require.NoError(t, err)

	plan := commitPlan(staged)

	require.Len(t, plan.Rules, 1)
	assert.Contains(t, plan.Rules[0].LaneKey, "IL")
}

// A duplicated lane would leave the contract with two rules the engine has to
// break a tie between, which is a coin flip somebody did not ask for.
func TestCommitPlan_DropsADuplicatedLaneRatherThanWritingBoth(t *testing.T) {
	t.Parallel()

	staged, err := stage(sheet(
		[]string{"IL", "GA", "PerMile", "2.35"},
		[]string{"IL", "GA", "PerMile", "2.60"},
	), nil, resolveTemplate)
	require.NoError(t, err)

	plan := commitPlan(staged)

	assert.Len(t, plan.Rules, 1)
}

func TestStagedBatch_KnowsWhetherCommittingWouldDoAnything(t *testing.T) {
	t.Parallel()

	staged, err := stage(
		sheet([]string{"IL", "GA", "PerMile", "2.10"}),
		[]*rateagreement.RateAgreementRule{existingRule("ST:IL>ST:GA", "2.10")},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.False(t, staged.Summary.ChangesAnything())
}
