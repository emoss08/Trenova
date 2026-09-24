package aifeedback

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validFeedback() *Feedback {
	return &Feedback{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		UserID:            pulid.MustNew("usr_"),
		TargetType:        TargetAssistantMessage,
		TargetID:          pulid.MustNew("amsg_"),
		FingerprintSource: FingerprintAtRating,
		Rating:            RatingNegative,
		Reasons:           []Reason{ReasonInaccurate, ReasonIncomplete},
		Comment:           "The ETA was a day off",
		PatternKey:        PatternKey([]string{"get_shipment"}, ReasonInaccurate, "Shipment"),
	}
}

func fieldsOf(t *testing.T, entity *Feedback) []string {
	t.Helper()

	me := errortypes.NewMultiError()
	entity.Validate(me)
	fields := make([]string, 0, len(me.Errors))
	for _, err := range me.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestFeedbackValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Feedback)
		want   []string
	}{
		{name: "a complete thumbs down passes", mutate: func(*Feedback) {}},
		{
			name: "a thumbs up takes only positive reasons",
			mutate: func(f *Feedback) {
				f.Rating = RatingPositive
				f.Reasons = []Reason{ReasonHelpful, ReasonInaccurate}
			},
			want: []string{"reasons[1]"},
		},
		{
			name:   "a thumbs down refuses a positive reason",
			mutate: func(f *Feedback) { f.Reasons = []Reason{ReasonSavedTime} },
			want:   []string{"reasons[0]"},
		},
		{
			name:   "a reason may not be listed twice",
			mutate: func(f *Feedback) { f.Reasons = []Reason{ReasonUnsafe, ReasonUnsafe} },
			want:   []string{"reasons[1]"},
		},
		{
			name:   "an unknown reason is refused",
			mutate: func(f *Feedback) { f.Reasons = []Reason{"Rude"} },
			want:   []string{"reasons[0]"},
		},
		{
			name:   "no reasons is allowed",
			mutate: func(f *Feedback) { f.Reasons = nil },
		},
		{
			name:   "a rating is a thumb, not a score",
			mutate: func(f *Feedback) { f.Rating = 5 },
			want:   []string{"rating"},
		},
		{
			name:   "a zero rating is refused",
			mutate: func(f *Feedback) { f.Rating = 0; f.Reasons = nil },
			want:   []string{"rating"},
		},
		{
			name:   "a comment is at most a thousand characters",
			mutate: func(f *Feedback) { f.Comment = strings.Repeat("é", MaxCommentRunes+1) },
			want:   []string{"comment"},
		},
		{
			name:   "a comment of exactly a thousand multibyte characters passes",
			mutate: func(f *Feedback) { f.Comment = strings.Repeat("é", MaxCommentRunes) },
		},
		{
			name: "a briefing section names its part",
			mutate: func(f *Feedback) {
				f.TargetType = TargetBriefingSection
			},
			want: []string{"targetPart"},
		},
		{
			name: "a whole answer names no part",
			mutate: func(f *Feedback) {
				f.TargetPart = "dispatch"
			},
			want: []string{"targetPart"},
		},
		{
			name:   "an unknown target type is refused",
			mutate: func(f *Feedback) { f.TargetType = "Email" },
			want:   []string{"targetType"},
		},
		{
			name:   "the rater is required",
			mutate: func(f *Feedback) { f.UserID = pulid.Nil },
			want:   []string{"userId"},
		},
		{
			name:   "a pattern key is a sha-256 hex digest",
			mutate: func(f *Feedback) { f.PatternKey = "abc" },
			want:   []string{"patternKey"},
		},
		{
			name:   "the fingerprint source must be known",
			mutate: func(f *Feedback) { f.FingerprintSource = "Guess" },
			want:   []string{"fingerprintSource"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validFeedback()
			tt.mutate(entity)
			fields := fieldsOf(t, entity)
			if len(tt.want) == 0 {
				assert.Empty(t, fields)
				return
			}
			for _, field := range tt.want {
				assert.Contains(t, fields, field)
			}
		})
	}
}

func TestReasonSides(t *testing.T) {
	t.Parallel()

	for _, reason := range NegativeReasons() {
		assert.True(t, reason.MatchesRating(RatingNegative), reason)
		assert.False(t, reason.MatchesRating(RatingPositive), reason)
	}
	for _, reason := range PositiveReasons() {
		assert.True(t, reason.MatchesRating(RatingPositive), reason)
		assert.False(t, reason.MatchesRating(RatingNegative), reason)
	}
	require.Len(t, AllReasons(), len(NegativeReasons())+len(PositiveReasons()))
}

func TestPatternKey(t *testing.T) {
	t.Parallel()

	key := PatternKey(
		[]string{"get_shipment", "get_shipment", "list_moves"},
		ReasonInaccurate,
		"Shipment",
	)
	assert.Len(t, key, PatternKeyLength)
	assert.Equal(t, key,
		PatternKey([]string{"get_shipment", "list_moves"}, ReasonInaccurate, "Shipment"),
		"a tool called twice in a row is one step of the pattern")
	assert.NotEqual(t, key,
		PatternKey([]string{"list_moves", "get_shipment"}, ReasonInaccurate, "Shipment"),
		"the order of the tools matters")
	assert.NotEqual(t, key,
		PatternKey([]string{"get_shipment", "list_moves"}, ReasonIncomplete, "Shipment"))
	assert.NotEqual(t, key,
		PatternKey([]string{"get_shipment", "list_moves"}, ReasonInaccurate, "Customer"))
}
