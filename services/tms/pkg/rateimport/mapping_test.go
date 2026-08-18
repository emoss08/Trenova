package rateimport_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A rate sheet arrives however the person who made it named their columns, and
// most of them are recognisable. Making somebody map twenty columns by hand
// when nineteen are obvious is the reason bulk import goes unused.
func TestGuessMapping_MatchesTheObviousHeaders(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{
		"Origin State",
		"Destination State",
		"Rate",
		"Minimum Charge",
	})

	assert.Equal(t, 0, mapping[rateimport.FieldOriginValue])
	assert.Equal(t, 1, mapping[rateimport.FieldDestinationValue])
	assert.Equal(t, 2, mapping[rateimport.FieldRate])
	assert.Equal(t, 3, mapping[rateimport.FieldMinCharge])
}

// Spreadsheets come out of other systems with whatever spacing and casing that
// system used, and none of it means anything.
func TestGuessMapping_IgnoresCaseSpacingAndPunctuation(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"  ORIGIN_STATE  ", "dest. state", "RATE"})

	assert.Equal(t, 0, mapping[rateimport.FieldOriginValue])
	assert.Equal(t, 1, mapping[rateimport.FieldDestinationValue])
	assert.Equal(t, 2, mapping[rateimport.FieldRate])
}

// A header the guesser does not recognise is reported rather than dropped.
// Silently ignoring a column is how a sheet imports looking complete while a
// discount nobody noticed never made it in.
func TestGuessMapping_ReportsHeadersItCouldNotPlace(t *testing.T) {
	t.Parallel()

	_, unmatched := rateimport.GuessMapping([]string{"Origin State", "Sales Rep", "Rate"})

	assert.Equal(t, []string{"Sales Rep"}, unmatched)
}

// A sheet with two columns claiming the same field is ambiguous, and guessing
// which one was meant would price a tariff off the wrong column. The first wins
// and the second is reported as unplaced so somebody decides.
func TestGuessMapping_TwoColumnsClaimingOneFieldKeepsTheFirst(t *testing.T) {
	t.Parallel()

	// Both of these are names the rate field answers to, so this is a genuine
	// collision rather than one recognised header and one stranger.
	mapping, unmatched := rateimport.GuessMapping([]string{"Rate", "Linehaul"})

	assert.Equal(t, 0, mapping[rateimport.FieldRate])
	assert.Equal(t, []string{"Linehaul"}, unmatched)
}

// The scope columns are what turn "GA" into a state lane rather than a city
// one, and a sheet that names them is telling the importer something it cannot
// infer.
func TestGuessMapping_RecognisesTheScopeColumns(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Origin Type", "Destination Type"})

	assert.Equal(t, 0, mapping[rateimport.FieldOriginScope])
	assert.Equal(t, 1, mapping[rateimport.FieldDestinationScope])
}

func TestGuessMapping_RecognisesTheDateColumns(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Effective Date", "Expires"})

	assert.Equal(t, 0, mapping[rateimport.FieldEffectiveFrom])
	assert.Equal(t, 1, mapping[rateimport.FieldEffectiveTo])
}

func TestGuessMapping_RecognisesTheBasisColumn(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Rate Type"})

	assert.Equal(t, 0, mapping[rateimport.FieldRatingBasis])
}

func TestGuessMapping_AnEmptyHeaderRowMapsNothing(t *testing.T) {
	t.Parallel()

	mapping, unmatched := rateimport.GuessMapping(nil)

	assert.Empty(t, mapping)
	assert.Empty(t, unmatched)
}

// A blank column in the middle of a sheet is padding, not a field somebody
// forgot to name, and reporting it as unplaced would train people to ignore
// the list.
func TestGuessMapping_BlankHeadersAreNotReportedAsUnplaced(t *testing.T) {
	t.Parallel()

	_, unmatched := rateimport.GuessMapping([]string{"Rate", "", "   "})

	assert.Empty(t, unmatched)
}

// Whether a mapping is usable is a different question from whether every column
// was placed: a sheet can carry extra columns and still import.
func TestMapping_IsUsableOnceItCanLocateALaneAndAPrice(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Origin State", "Destination State", "Rate"})

	require.NoError(t, mapping.Validate())
}

func TestMapping_WithoutARateColumnCannotPriceAnything(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Origin State", "Destination State"})

	require.ErrorIs(t, mapping.Validate(), rateimport.ErrNoRateColumn)
}

// A sheet naming only one end of the lane is missing half of what a rule needs,
// and importing it would write lanes that match everything on the other side.
func TestMapping_WithoutBothEndsOfTheLaneIsRefused(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Origin State", "Rate"})

	require.ErrorIs(t, mapping.Validate(), rateimport.ErrNoDestinationColumn)
}
