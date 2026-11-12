package casereading

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/recordcaseservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/stretchr/testify/assert"
)

func composed() *recordcaseservice.Reading {
	return &recordcaseservice.Reading{
		Label:  "SHP-1",
		Title:  "Dallas, TX → Chicago, IL",
		Status: "In transit · projected late",
		Exhibits: []recordcase.Exhibit{
			{Number: 1, Title: "Where it is", Claims: []string{"215 miles to Chicago"}},
			{
				Number: 2,
				Title:  "John Smith's hours",
				Claims: []string{"John Smith has 2h 10m of driving left of 11h"},
			},
		},
		Signals: []string{"Projected late on SHP-1"},
	}
}

func TestGuarded_KeepsASentenceThatCitesOnlyWhatTheEvidenceSays(t *testing.T) {
	supported := supportedBy(composed())

	assert.Equal(t,
		"He has 2h 10m of driving left and 215 miles to go.",
		guarded(new("  He has 2h 10m of driving\nleft and 215 miles to go.  "), supported),
	)
}

func TestGuarded_DropsAFigureNobodyComputed(t *testing.T) {
	supported := supportedBy(composed())

	assert.Empty(t, guarded(new("The missed door costs $250."), supported))
}

func TestGuarded_DropsTextThatCarriesTheAnswersOwnStructure(t *testing.T) {
	supported := supportedBy(composed())

	assert.Empty(t, guarded(new(`It misses the window.','ifNothingChanges':"}`), supported))
	assert.Empty(t, guarded(new("It misses the window [2]."), supported))
	assert.Empty(t, guarded(new("==The card expires in 12 days==, so he may not drive."), supported))
	assert.Empty(t, guarded(new("He may not drive that load.</br>"), supported))
	assert.Empty(t, guarded(new(strings.Repeat("late ", 80)), supported))
	assert.Empty(t, guarded(new("   "), supported))
}

func TestUnmarked_IsTheVerdictAsTheGuardReadsIt(t *testing.T) {
	plain := unmarked("He has ==2h 10m of driving left==[2] and is 215 miles out[1].")

	assert.NotContains(t, plain, "==")
	assert.NotContains(t, plain, "[")
	assert.True(t, numberguard.CheckNumbers(plain, supportedBy(composed())).OK,
		"a citation's number is never read as a figure the verdict claims")
}

func TestBuildContext_NumbersEachExhibitAndWithholdsThePlansDetail(t *testing.T) {
	reading := composed()
	reading.Noun = "shipment"
	reading.Proposal = "Move the delivery to 1:00 PM"

	context := buildContext(reading)

	assert.Len(t, context.Sections, 1)
	content := context.Sections[0].Content
	assert.False(t, context.Sections[0].Trusted)
	assert.Contains(t, content, "Record: shipment SHP-1 (Dallas, TX → Chicago, IL)")
	assert.Contains(t, content, "[1] Where it is")
	assert.Contains(t, content, "[2] John Smith's hours")
	assert.Contains(t, content, "  - John Smith has 2h 10m of driving left of 11h")
	assert.Contains(t, content, "Do not describe the plan.")
}

func TestMarked_MarksTheDecidingFactWhereTheVerdictAlreadySaysIt(t *testing.T) {
	verdict := "He has 2h 10m of driving left [2] and is 215 miles out [1]."

	assert.Equal(t,
		"He has ==2h 10m of driving left== [2] and is 215 miles out [1].",
		marked(verdict, " 2H 10m of driving left. ", nil),
	)
	assert.Equal(t,
		verdict,
		marked("He has ==2h 10m== of driving left [2] and is 215 miles out [1].", "", nil),
	)
	assert.Equal(t, verdict, marked(verdict, "3 hours of driving left", nil))
}

func TestMarked_FallsBackToTheLongestFigureTheEvidenceWroteTheSameWay(t *testing.T) {
	verdict := "John Smith has 2h 10m of driving left [2] and is 215 miles out [1]."
	evidence := []string{"215 miles to Chicago", "John Smith has 2h 10m of driving left of 11h"}

	assert.Equal(t,
		"==John Smith has 2h 10m of driving left== [2] and is 215 miles out [1].",
		marked(verdict, "the hours he has left", evidence),
	)
	assert.Equal(t,
		"The door is shut [1].",
		marked("The door is shut [1].", "", []string{"The door is shut"}),
	)
}

func TestGuarded_KeepsOneSentenceAndDropsWhatTheModelLeftAfterIt(t *testing.T) {
	supported := numberguard.SupportedFromText("12 days", "75.00 USD")

	for written, kept := range map[string]string{
		"Her endorsement must be renewed first.) \u30c4":                              "Her endorsement must be renewed first.",
		"He would be dispatched while it is close to lapsing.... ','ifNothingChanges": "He would be dispatched while it is close to lapsing.",
		"The balance stays unpaid. ENDOFJSON":                                         "The balance stays unpaid.",
		"The amount is already accruing.although the stop was on time.":               "The amount is already accruing.",
		"Acme owes 75.00 USD so far.":                                                 "Acme owes 75.00 USD so far.",
		"Nothing in the facts says what happens if it lapses.":                        "",
		"no full stop anywhere":                                                       "no full stop anywhere",
	} {
		assert.Equal(t, kept, guarded(new(written), supported), written)
	}
	assert.Empty(t, guarded(nil, supported))
}

// A verdict the model let trail off is not a finding: it is refused and asked
// for again, and a sentence that ends is kept.
func TestTrailsOff_RefusesAVerdictLeftUnfinished(t *testing.T) {
	t.Parallel()

	assert.True(t, trailsOff("Its delivery window closed Sun 2:36 PM..."))
	assert.True(t, trailsOff("Its delivery window closed Sun 2:36 PM… "))
	assert.False(t, trailsOff("Its delivery window closed Sun 2:36 PM PST with no arrival [2]."))
}

/*
What a customer or a carrier wrote is the one kind of evidence a reading may
not pass over. A verdict that cites every message is kept; one that cites the
conditions and leaves a message out is asked for again, naming the exhibit.
*/
func TestUnreadMessages_NamesEachMessageTheVerdictPassedOver(t *testing.T) {
	t.Parallel()

	exhibits := []recordcase.Exhibit{
		{Number: 1, Kind: recordcase.ExhibitRoute},
		{Number: 2, Kind: recordcase.ExhibitMessage},
		{Number: 3, Kind: recordcase.ExhibitFacts},
		{Number: 4, Kind: recordcase.ExhibitMessage},
	}
	conditionsOnly := []recordcase.Segment{{Text: "No proof of delivery"}, {Cite: 3}}
	everyMessage := []recordcase.Segment{{Cite: 2}, {Text: " and "}, {Cite: 4}}

	assert.Equal(t, []int{2, 4}, unreadMessages(conditionsOnly, exhibits))
	assert.Empty(t, unreadMessages(everyMessage, exhibits))
	assert.Contains(t, unreadCorrection([]int{2, 4}), "[2], [4]")
}
