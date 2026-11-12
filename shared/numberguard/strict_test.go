package numberguard_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/stretchr/testify/assert"
)

func TestCheckSpelled_RefusesAFigureWrittenAsAWordThatNoEvidenceHolds(t *testing.T) {
	supported := numberguard.SupportedFromText("14 days past due", "2 credentials expired")

	check := numberguard.CheckSpelled(
		"The truck left the window by over three hours and the account is two weeks late.",
		supported,
	)

	assert.False(t, check.OK)
	assert.Equal(t, []string{"three"}, check.Unsupported)
}

func TestCheckSpelled_PassesWordsTheEvidenceHoldsAndLeavesOneAlone(t *testing.T) {
	supported := numberguard.SupportedFromText("2 credentials expired")

	assert.True(t, numberguard.CheckSpelled("Her two credentials lapsed.", supported).OK)
	assert.True(t, numberguard.CheckSpelled("No one is assigned to this one.", supported).OK)
	assert.True(t, numberguard.CheckSpelled("", supported).OK)
}

func TestCheckEvery_RefusesASmallFigureTheEvidenceDoesNotHold(t *testing.T) {
	supported := numberguard.SupportedFromText(
		"Delivery window 5:36–7:36 PM, departed 10:46 PM",
		"2h 10m of driving left",
		"expired 5d ago",
	)

	late := numberguard.CheckEvery("The truck left 3 hours after the window closed.", supported)
	assert.False(t, late.OK)
	assert.Equal(t, []string{"3"}, late.Unsupported)

	assert.True(t, numberguard.CheckEvery(
		"He has 2h 10m of driving left and both cards expired 5 days ago, at 10:46 PM.",
		supported,
	).OK)
}
