package shipmentbrief

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeKeepsTheSpacingBetweenSegments(t *testing.T) {
	t.Parallel()

	brief := &Brief{Segments: []Segment{
		{Text: "3 loads deliver today"},
		{Text: " and "},
		{Text: strings.Repeat("é", MaxSegmentLength+10)},
	}}
	brief.Normalize()

	assert.Equal(t, " and ", brief.Segments[1].Text)
	assert.Equal(t, MaxSegmentLength, utf8.RuneCountInString(brief.Segments[2].Text))
}

func TestNormalizeCountsOpenIssuesFromTheFacts(t *testing.T) {
	t.Parallel()

	brief := &Brief{Facts: Facts{Late: 2, Uncovered: 3, OpenSuggestions: 4, Detention: 9}}
	brief.Normalize()

	assert.Equal(t, 9, brief.OpenIssues)
	assert.NotNil(t, brief.Segments)
	assert.NotNil(t, brief.Wording)
}

func TestValidateRefusesAnUnknownTrigger(t *testing.T) {
	t.Parallel()

	brief := &Brief{
		OrganizationID: "org_a",
		BusinessUnitID: "bu_a",
		BriefDate:      "2026-10-07",
		Generation:     1,
		Trigger:        Trigger("Hourly"),
		GeneratedAt:    1,
	}
	multiErr := errortypes.NewMultiError()
	brief.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	brief.Trigger = TriggerCleared
	multiErr = errortypes.NewMultiError()
	brief.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestWordingFitsOnlyTheTextItReworded(t *testing.T) {
	t.Parallel()

	wording := Wording{
		Title:  "Send it out",
		Reason: "Closes at 13:30.",
		Basis:  WordingBasis("Tender A → B", "Pickup closes at 13:30.", []string{"$3590"}),
	}

	assert.True(t, wording.Fits("Tender A → B", "Pickup closes at 13:30.", []string{"$3590"}))
	assert.False(t, wording.Fits("Tender A → B", "Pickup closes at 14:00.", []string{"$3590"}))
	assert.False(t, wording.Fits("Tender A → B", "Pickup closes at 13:30.", nil))
	assert.False(t, Wording{Basis: wording.Basis}.Fits(
		"Tender A → B", "Pickup closes at 13:30.", []string{"$3590"},
	))
}
