package recordcase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVerdict_SplitsTextHighlightsAndCitations(t *testing.T) {
	segments := recordcase.ParseVerdict(
		"Marcus has ==2h 10m of driving left== [2] and is 215 miles out[1].",
		4,
	)

	assert.Equal(t, []recordcase.Segment{
		{Text: "Marcus has "},
		{Text: "2h 10m of driving left", Highlight: true},
		{Cite: 2},
		{Text: " and is 215 miles out"},
		{Cite: 1},
		{Text: "."},
	}, segments)
}

func TestParseVerdict_DropsACitationOfAnExhibitThatDoesNotExist(t *testing.T) {
	segments := recordcase.ParseVerdict("The door closes at 3:00 PM[7]. Acme is asking[2].", 3)

	assert.Equal(t, []recordcase.Segment{
		{Text: "The door closes at 3:00 PM. Acme is asking"},
		{Cite: 2},
		{Text: "."},
	}, segments)
}

func TestParseVerdict_CitesAnExhibitOnceForARunOfClaimsThatRestOnIt(t *testing.T) {
	segments := recordcase.ParseVerdict(
		"Both cards expired 5d ago [1]. The record flags this as ==critical== [1]. "+
			"Dispatch still has her on a load [2]. Both are expired [1].",
		2,
	)

	assert.Equal(t, []recordcase.Segment{
		{Text: "Both cards expired 5d ago. The record flags this as "},
		{Text: "critical", Highlight: true},
		{Cite: 1},
		{Text: ". Dispatch still has her on a load"},
		{Cite: 2},
		{Text: ". Both are expired"},
		{Cite: 1},
		{Text: "."},
	}, segments)
}

func TestParseVerdict_PutsACitationBeforeTheFullStopItFollows(t *testing.T) {
	segments := recordcase.ParseVerdict("Both cards expired 5 days ago.[1] She is still scheduled.", 1)

	assert.Equal(t, []recordcase.Segment{
		{Text: "Both cards expired 5 days ago"},
		{Cite: 1},
		{Text: ". She is still scheduled."},
	}, segments)
}

func TestParseVerdict_DealsStackedCitationsBackOverTheClausesTheyBelongTo(t *testing.T) {
	segments := recordcase.ParseVerdict(
		"Invoice INV-6 for $2,531.25 is unpaid. It is 14 days past due, having been due Oct 27 [1][2].",
		2,
	)

	assert.Equal(t, []recordcase.Segment{
		{Text: "Invoice INV-6 for $2,531.25 is unpaid. It is 14 days past due"},
		{Cite: 1},
		{Text: ", having been due Oct 27"},
		{Cite: 2},
		{Text: "."},
	}, segments)
}

func TestParseVerdict_LeavesStackedCitationsThatHaveNoClauseOfTheirOwn(t *testing.T) {
	segments := recordcase.ParseVerdict("The load is late [1][2].", 2)

	assert.Equal(t, []recordcase.Segment{
		{Text: "The load is late"},
		{Cite: 1},
		{Cite: 2},
		{Text: "."},
	}, segments)
}

func TestParseVerdict_KeepsBracketsThatAreNotCitations(t *testing.T) {
	segments := recordcase.ParseVerdict("The note says [see BOL] and stops there.", 2)

	require.Len(t, segments, 1)
	assert.Equal(t, "The note says [see BOL] and stops there.", segments[0].Text)
}

func TestParseVerdict_CollapsesWhitespaceAndBoundsItsLength(t *testing.T) {
	long := ""
	for range 200 {
		long += "late  again\n"
	}

	segments := recordcase.ParseVerdict(long, 0)

	require.Len(t, segments, 1)
	assert.LessOrEqual(t, len([]rune(segments[0].Text)), recordcase.MaxVerdictLength)
	assert.NotContains(t, segments[0].Text, "  ")
	assert.NotContains(t, segments[0].Text, "\n")
}

func TestReadingText_IsTheVerdictWithoutItsMarks(t *testing.T) {
	reading := &recordcase.Reading{
		Segments: recordcase.ParseVerdict("He has ==2h left==[2] today.", 2),
	}

	assert.Equal(t, "He has 2h left today.", reading.Text())
}

func TestReadingFingerprint_IgnoresTheOrderSignalsAreReadIn(t *testing.T) {
	due := int64(1_800_003_600)
	late := recordcase.Signal{
		Kind: "EtaAtRisk", SourceID: "shp_1", Severity: "Critical", Title: "Projected late",
		DueAt: &due,
	}
	asking := recordcase.Signal{
		Kind: "InboundMessage", SourceID: "im_1", Severity: "Warning", Title: "ETA?",
	}

	first := recordcase.ReadingFingerprint("In transit", []recordcase.Signal{late, asking})
	second := recordcase.ReadingFingerprint("In transit", []recordcase.Signal{asking, late})

	assert.Equal(t, first, second)
	assert.Len(t, first, 32)
}

func TestReadingFingerprint_ChangesWithWhatIsOpenAndWhereTheRecordStands(t *testing.T) {
	due := int64(1_800_003_600)
	later := due + 2*3600
	late := recordcase.Signal{
		Kind: "EtaAtRisk", SourceID: "shp_1", Severity: "Critical", Title: "Projected late",
		DueAt: &due,
	}
	base := recordcase.ReadingFingerprint("In transit", []recordcase.Signal{late})

	worse := late
	worse.Severity = "Warning"
	moved := late
	moved.DueAt = &later

	assert.NotEqual(t, base, recordcase.ReadingFingerprint("Delivered", []recordcase.Signal{late}))
	assert.NotEqual(
		t,
		base,
		recordcase.ReadingFingerprint("In transit", []recordcase.Signal{worse}),
	)
	assert.NotEqual(
		t,
		base,
		recordcase.ReadingFingerprint("In transit", []recordcase.Signal{moved}),
	)
	assert.NotEqual(t, base, recordcase.ReadingFingerprint("In transit", nil))
}

func TestReadingFingerprint_HoldsWhileADeadlineMovesWithinTheHour(t *testing.T) {
	due := int64(1_800_003_600)
	nudged := due + 600
	late := recordcase.Signal{Kind: "EtaAtRisk", SourceID: "shp_1", DueAt: &due}
	same := recordcase.Signal{Kind: "EtaAtRisk", SourceID: "shp_1", DueAt: &nudged}

	assert.Equal(t,
		recordcase.ReadingFingerprint("In transit", []recordcase.Signal{late}),
		recordcase.ReadingFingerprint("In transit", []recordcase.Signal{same}),
	)
}

func TestReadingCurrent_IsFalseForNoReadingAndForAnotherFingerprint(t *testing.T) {
	var none *recordcase.Reading
	reading := &recordcase.Reading{Fingerprint: "abc"}

	assert.False(t, none.Current("abc"))
	assert.True(t, reading.Current("abc"))
	assert.False(t, reading.Current("abd"))
}

func option(key string, kind recordcase.OptionKind, recommended bool) *recordcase.Option {
	return &recordcase.Option{Key: key, Kind: kind, Label: key, Recommended: recommended}
}

func TestValidateSet_TakesTwoOrThreeOptionsWithExactlyOneRecommended(t *testing.T) {
	cases := map[string]struct {
		options []*recordcase.Option
		valid   bool
	}{
		"two with one recommended": {
			options: []*recordcase.Option{
				option("relay", recordcase.OptionKindPlan, true),
				option("leave", recordcase.OptionKindLeave, false),
			},
			valid: true,
		},
		"one option": {
			options: []*recordcase.Option{option("relay", recordcase.OptionKindPlan, true)},
		},
		"four options": {
			options: []*recordcase.Option{
				option("a", recordcase.OptionKindPlan, true),
				option("b", recordcase.OptionKindPlan, false),
				option("c", recordcase.OptionKindPlan, false),
				option("d", recordcase.OptionKindLeave, false),
			},
		},
		"none recommended": {
			options: []*recordcase.Option{
				option("relay", recordcase.OptionKindPlan, false),
				option("leave", recordcase.OptionKindLeave, false),
			},
		},
		"two recommended": {
			options: []*recordcase.Option{
				option("relay", recordcase.OptionKindPlan, true),
				option("move", recordcase.OptionKindPlan, true),
			},
		},
		"a key used twice": {
			options: []*recordcase.Option{
				option("relay", recordcase.OptionKindPlan, true),
				option("relay", recordcase.OptionKindPlan, false),
			},
		},
		"two ways of leaving it": {
			options: []*recordcase.Option{
				option("relay", recordcase.OptionKindPlan, true),
				option("leave", recordcase.OptionKindLeave, false),
				option("wait", recordcase.OptionKindLeave, false),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			multiErr := errortypes.NewMultiError()

			recordcase.ValidateSet(tc.options, multiErr)

			assert.Equal(t, tc.valid, !multiErr.HasErrors())
		})
	}
}

func TestOptionValidate_RefusesRecommendingToLeaveItAsItIs(t *testing.T) {
	leave := &recordcase.Option{
		OrganizationID: "org_1",
		BusinessUnitID: "bu_1",
		RecordKind:     "shipment",
		RecordID:       "shp_01JABCDEFGHJKMNPQRSTVWXYZ0",
		Key:            "leave",
		Kind:           recordcase.OptionKindLeave,
		Label:          "Leave it as it is",
		Recommended:    true,
	}
	leave.Normalize()
	multiErr := errortypes.NewMultiError()

	leave.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestChosen_IsTheOnePickedAndOtherwiseTheOneRecommended(t *testing.T) {
	relay := option("relay", recordcase.OptionKindPlan, true)
	move := option("move", recordcase.OptionKindPlan, false)

	assert.Same(t, relay, recordcase.Chosen([]*recordcase.Option{move, relay}))

	move.Chosen = true
	assert.Same(t, move, recordcase.Chosen([]*recordcase.Option{relay, move}))
	assert.Nil(t, recordcase.Chosen(nil))
}

func TestOptionNormalize_DropsAProjectionThatProjectsNothing(t *testing.T) {
	empty := &recordcase.Option{
		Key:        " Relay ",
		Label:      "Relay",
		Projection: &recordcase.Projection{Step: &recordcase.ProjectedStep{Label: "  "}},
	}
	kept := &recordcase.Option{
		Key:        "move",
		Label:      "Move",
		Projection: &recordcase.Projection{Window: " Tue 7:00–8:00 AM "},
	}

	empty.Normalize()
	kept.Normalize()

	assert.Equal(t, "relay", empty.Key)
	assert.Nil(t, empty.Projection)
	assert.Equal(t, recordcase.OptionKindPlan, empty.Kind)
	assert.Equal(t, recordcase.ToneNeutral, empty.Tone)
	require.NotNil(t, kept.Projection)
	assert.Equal(t, "Tue 7:00–8:00 AM", kept.Projection.Window)
}

func TestDraftNote_NamesTheOptionAndAsksForItsPlanOnly(t *testing.T) {
	relay := &recordcase.Option{
		Label:       "Relay at Bloomington",
		Description: "A second driver takes the load at the yard.",
	}

	note := relay.DraftNote()

	assert.Contains(t, note, `"Relay at Bloomington": A second driver takes the load at the yard.`)
	assert.Contains(t, note, "Do not offer the options again")
}
