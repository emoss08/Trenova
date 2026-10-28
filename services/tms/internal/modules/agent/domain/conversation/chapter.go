package conversation

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	ChapterTitleRunes    = 60
	MaxChaptersPerThread = 200
	ChapterIDPrefix      = "achp_"
	chapterTableName     = "assistant_thread_chapters"
)

var (
	_ validationframework.TenantedEntity = (*Chapter)(nil)

	chapterMarkup = regexp.MustCompile("[*_`#>\\[\\]]+")
	chapterSpaces = regexp.MustCompile(`\s+`)
)

type ChapterTitleSource string

const (
	ChapterTitleModel    = ChapterTitleSource("Model")
	ChapterTitleQuestion = ChapterTitleSource("Question")
	ChapterTitlePerson   = ChapterTitleSource("Person")
	ChapterTitleSchedule = ChapterTitleSource("Schedule")
)

func AllChapterTitleSources() []ChapterTitleSource {
	return []ChapterTitleSource{
		ChapterTitleModel,
		ChapterTitleQuestion,
		ChapterTitlePerson,
		ChapterTitleSchedule,
	}
}

func (s ChapterTitleSource) String() string { return string(s) }

func (s ChapterTitleSource) IsValid() bool {
	return slices.Contains(AllChapterTitleSources(), s)
}

func (s ChapterTitleSource) Renamable() bool {
	return s != ChapterTitlePerson
}

type Chapter struct {
	keel.Table `keel:"assistant_thread_chapters"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:achp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ThreadID    pulid.ID           `json:"threadId"    keel:"thread_id"`
	TurnID      pulid.ID           `json:"turnId"      keel:"turn_id"`
	Title       string             `json:"title"       keel:"title"`
	TitleSource ChapterTitleSource `json:"titleSource" keel:"title_source"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	StartSequence *int64 `json:"startSequence" keel:"start_sequence,computed"`
	StartedAt     int64  `json:"startedAt"     keel:"started_at,computed,nullzero"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (c *Chapter) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.ThreadID, validation.Required.Error("Conversation is required")),
		validation.Field(&c.TurnID, validation.Required.Error("Turn is required")),
	))
	ValidateChapterTitle("title", c.Title, multiErr)
	if !c.TitleSource.IsValid() {
		multiErr.Add("titleSource", errortypes.ErrInvalid, "Unknown chapter title source")
	}
}

func ValidateChapterTitle(field, title string, multiErr *errortypes.MultiError) {
	switch {
	case strings.TrimSpace(title) == "":
		multiErr.Add(field, errortypes.ErrRequired, "A chapter needs a title")
	case utf8.RuneCountInString(title) > ChapterTitleRunes:
		multiErr.Add(field, errortypes.ErrInvalid, "A chapter title can be at most 60 characters")
	}
}

func CleanChapterTitle(raw string) string {
	title := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, raw)
	title = chapterMarkup.ReplaceAllString(title, "")
	title = chapterSpaces.ReplaceAllString(strings.TrimSpace(title), " ")
	title = strings.Trim(title, `"'“”‘’ `)
	title = strings.TrimRight(title, ".:;,!-–— ")
	if lower := strings.ToLower(title); strings.HasPrefix(lower, "chapter:") {
		title = strings.TrimSpace(title[len("chapter:"):])
	}
	if title == "" {
		return ""
	}
	first, size := utf8.DecodeRuneInString(title)
	title = string(unicode.ToUpper(first)) + title[size:]

	return stringutils.Ellipsize(title, ChapterTitleRunes)
}

func ChapterTitleFrom(question string) string {
	if title := CleanChapterTitle(question); title != "" {
		return title
	}

	return "Conversation"
}

func (c *Chapter) GetID() pulid.ID { return c.ID }

func (c *Chapter) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *Chapter) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *Chapter) GetTableName() string { return chapterTableName }
