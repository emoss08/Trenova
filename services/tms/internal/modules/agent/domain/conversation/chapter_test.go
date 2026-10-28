package conversation

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCleanChapterTitle_WritesAPlainSentenceCaseTitle(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw  string
		want string
	}{
		"bold and italic markup": {
			raw:  "**Acme** _invoice_ dispute",
			want: "Acme invoice dispute",
		},
		"a heading with code": {
			raw:  "## Late loads on the `Dallas` lane",
			want: "Late loads on the Dallas lane",
		},
		"a quote with a link": {
			raw:  "> [Detention] policy setup",
			want: "Detention policy setup",
		},
		"straight quotes": {
			raw:  `"acme invoice dispute"`,
			want: "Acme invoice dispute",
		},
		"single quotes": {
			raw:  "'fuel card review'",
			want: "Fuel card review",
		},
		"smart double quotes": {
			raw:  "“Driver hours for Tuesday”",
			want: "Driver hours for Tuesday",
		},
		"smart single quotes": {
			raw:  "‘Fuel card review’",
			want: "Fuel card review",
		},
		"a full stop": {
			raw:  "Acme invoice dispute.",
			want: "Acme invoice dispute",
		},
		"a colon": {
			raw:  "Late loads:",
			want: "Late loads",
		},
		"a semicolon": {
			raw:  "Late loads;",
			want: "Late loads",
		},
		"a comma": {
			raw:  "Late loads,",
			want: "Late loads",
		},
		"exclamations": {
			raw:  "Late loads!!",
			want: "Late loads",
		},
		"a trailing hyphen": {
			raw:  "Late loads -",
			want: "Late loads",
		},
		"a trailing en dash": {
			raw:  "Late loads –",
			want: "Late loads",
		},
		"a trailing em dash": {
			raw:  "Late loads —",
			want: "Late loads",
		},
		"a run of trailing punctuation": {
			raw:  "Late loads...!",
			want: "Late loads",
		},
		"a chapter prefix": {
			raw:  "Chapter: Acme invoice dispute",
			want: "Acme invoice dispute",
		},
		"an upper-case chapter prefix": {
			raw:  "CHAPTER: late loads",
			want: "Late loads",
		},
		"a lower-case chapter prefix": {
			raw:  "chapter:detention policy",
			want: "Detention policy",
		},
		"tabs and line breaks": {
			raw:  "Acme\tinvoice\n\n  dispute",
			want: "Acme invoice dispute",
		},
		"control characters": {
			raw:  "Acme\u0000invoice\u0007dispute",
			want: "Acme invoice dispute",
		},
		"surrounding space": {
			raw:  "   late loads   ",
			want: "Late loads",
		},
		"a first rune outside ASCII": {
			raw:  "élan freight audit",
			want: "Élan freight audit",
		},
		"a title already clean": {
			raw:  "Acme invoice dispute",
			want: "Acme invoice dispute",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, CleanChapterTitle(tc.raw))
		})
	}
}

func TestCleanChapterTitle_IsEmptyWhenNothingIsLeft(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "   ", "**", "``", `""`, "“”", "...", "—", "\n\t"} {
		assert.Empty(t, CleanChapterTitle(raw), "%q", raw)
	}
}

func TestCleanChapterTitle_EllipsizesPastSixtyRunes(t *testing.T) {
	t.Parallel()

	exact := strings.Repeat("a", ChapterTitleRunes)
	assert.Equal(t, "A"+strings.Repeat("a", ChapterTitleRunes-1), CleanChapterTitle(exact),
		"a title of exactly sixty runes is kept whole")

	long := CleanChapterTitle("Late loads " + strings.Repeat("x", 60))
	assert.Equal(t, "Late loads "+strings.Repeat("x", 48)+"…", long)
	assert.Equal(t, ChapterTitleRunes, utf8.RuneCountInString(long))

	atSpace := CleanChapterTitle(strings.Repeat("a", 58) + " bbbbbbbbbb")
	assert.Equal(t, "A"+strings.Repeat("a", 57)+"…", atSpace,
		"the cut does not leave a space before the ellipsis")

	wide := CleanChapterTitle(strings.Repeat("é", 70))
	assert.Equal(t, "É"+strings.Repeat("é", 58)+"…", wide, "runes are counted, not bytes")
}

func TestChapterTitleFrom_NamesABlankQuestionConversation(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Conversation", ChapterTitleFrom(""))
	assert.Equal(t, "Conversation", ChapterTitleFrom("  **  "))
	assert.Equal(t, "Why is load 1200 late?", ChapterTitleFrom("why is load 1200 late?"))
	assert.Equal(t, "Acme invoice dispute", ChapterTitleFrom("acme invoice dispute."))
}

func validChapter() *Chapter {
	return &Chapter{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ThreadID:       pulid.MustNew("athr_"),
		TurnID:         pulid.MustNew("atrn_"),
		Title:          "Acme invoice dispute",
		TitleSource:    ChapterTitleModel,
	}
}

func TestChapterValidate_AcceptsAWellFormedChapter(t *testing.T) {
	t.Parallel()

	for _, source := range AllChapterTitleSources() {
		chapter := validChapter()
		chapter.TitleSource = source
		multiErr := errortypes.NewMultiError()

		chapter.Validate(multiErr)

		assert.False(t, multiErr.HasErrors(), "%s: %v", source, multiErr.Errors)
	}
}

func TestChapterValidate_ReportsAMissingTitle(t *testing.T) {
	t.Parallel()

	chapter := validChapter()
	chapter.Title = "   "
	multiErr := errortypes.NewMultiError()

	chapter.Validate(multiErr)

	assert.Equal(t, []string{string(errortypes.ErrRequired)},
		validationtest.FieldCodes(multiErr, "title"))
}

func TestChapterValidate_ReportsATitleOverSixtyRunes(t *testing.T) {
	t.Parallel()

	chapter := validChapter()
	chapter.Title = strings.Repeat("é", ChapterTitleRunes+1)
	multiErr := errortypes.NewMultiError()

	chapter.Validate(multiErr)

	assert.Equal(t, []string{string(errortypes.ErrInvalid)},
		validationtest.FieldCodes(multiErr, "title"))

	chapter.Title = strings.Repeat("é", ChapterTitleRunes)
	atLimit := errortypes.NewMultiError()
	chapter.Validate(atLimit)
	assert.Empty(t, validationtest.FieldCodes(atLimit, "title"),
		"sixty runes of two bytes each is still sixty characters")
}

func TestChapterValidate_ReportsAnUnknownSource(t *testing.T) {
	t.Parallel()

	for _, source := range []ChapterTitleSource{"", "Robot"} {
		chapter := validChapter()
		chapter.TitleSource = source
		multiErr := errortypes.NewMultiError()

		chapter.Validate(multiErr)

		assert.Equal(t, []string{string(errortypes.ErrInvalid)},
			validationtest.FieldCodes(multiErr, "titleSource"), "%q", source)
	}
}

func TestChapterValidate_ReportsMissingReferences(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	(&Chapter{Title: "Acme invoice dispute", TitleSource: ChapterTitlePerson}).Validate(multiErr)

	for _, field := range []string{"organizationId", "businessUnitId", "threadId", "turnId"} {
		assert.NotEmpty(t, validationtest.FieldCodes(multiErr, field), field)
	}
}

func TestValidateChapterTitle_ReportsUnderTheGivenField(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	ValidateChapterTitle("name", "", multiErr)

	assert.Equal(t, []string{string(errortypes.ErrRequired)},
		validationtest.FieldCodes(multiErr, "name"))
	assert.Empty(t, validationtest.FieldCodes(multiErr, "title"))
}

func TestChapterTitleSource_OnlyAPersonsTitleIsKept(t *testing.T) {
	t.Parallel()

	assert.False(t, ChapterTitlePerson.Renamable())
	assert.True(t, ChapterTitleModel.Renamable())
	assert.True(t, ChapterTitleQuestion.Renamable())
	assert.True(t, ChapterTitleSchedule.Renamable())
}
