// Package briefing is the morning read: what the day looks like before
// anybody has opened a screen.
//
// A briefing is deterministic first and narrated second. The figures are
// gathered from the same services the pages read, stored as facts, and laid
// out in sections; a model is then asked for a headline and a sentence per
// section, and everything it writes is checked against those facts. If the
// model is off, refuses, or cites a number nobody computed, the briefing is
// still complete — it simply wears the deterministic wording. A number on a
// page headed "your day" gets repeated in a stand-up, so it is never the
// model's to invent.
package briefing

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	// MaxSections bounds the page: more blocks than this and nobody reads
	// any of them.
	MaxSections = 10
	// MaxSectionItems bounds one block's lines.
	MaxSectionItems = 12
	// MaxHeadlineLength keeps the headline to a sentence.
	MaxHeadlineLength = 240
	maxBodyLength     = 600
	maxLabelLength    = 120
	maxValueLength    = 80
	maxPathLength     = 500
)

// Item is one line of a section: a figure, what it counts, and the page
// that shows it. Value is already rendered, because the figure a person
// reads and the figure the guard checks must be the same string.
type Item struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Path  string `json:"path,omitempty"`
}

// IsSafePath reports whether the line points inside this application.
func (i Item) IsSafePath() bool {
	return i.Path == "" || stringutils.IsSafeAppPath(i.Path)
}

// Section is one block of the page. Body is the model's sentence about the
// lines; Summary is the deterministic wording it falls back to, so a
// section always reads as something.
type Section struct {
	Key     SectionKey `json:"key"`
	Title   string     `json:"title"`
	Summary string     `json:"summary"`
	Body    string     `json:"body,omitempty"`
	Items   []Item     `json:"items"`
	Path    string     `json:"path,omitempty"`
}

// Read is the sentence a person sees: the model's when it was accepted,
// the deterministic one otherwise.
func (s Section) Read() string {
	if strings.TrimSpace(s.Body) != "" {
		return s.Body
	}

	return s.Summary
}

// Briefing is one role's morning, on one day, for one organization.
type Briefing struct {
	bun.BaseModel `bun:"table:assistant_briefings,alias:abrf" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	RoleKey RoleKey `json:"roleKey" bun:"role_key,type:VARCHAR(20),notnull"`
	// UserID is set only where an organization writes a briefing per person
	// rather than per role. Nil is the role's shared briefing.
	UserID *pulid.ID `json:"userId" bun:"user_id,type:VARCHAR(100),nullzero"`
	// BriefingDate is the organization's local day, as YYYY-MM-DD. It is a
	// date rather than an instant because "today's briefing" means the
	// reader's today, not the server's.
	BriefingDate string   `json:"briefingDate" bun:"briefing_date,type:VARCHAR(10),notnull"`
	RunID        pulid.ID `json:"runId"        bun:"run_id,type:VARCHAR(100),nullzero"`

	Status   Status `json:"status"   bun:"status,type:VARCHAR(20),notnull"`
	Headline string `json:"headline" bun:"headline,type:VARCHAR(240),nullzero"`
	// Sections is the page. Facts is what the figures were drawn from, kept
	// so the guard can be re-run and a reader can be shown the working.
	Sections []Section      `json:"sections" bun:"sections,type:JSONB,notnull,default:'[]'"`
	Facts    map[string]any `json:"facts"    bun:"facts,type:JSONB,notnull,default:'{}'"`
	// Narrated says whether the model's wording was accepted. False means
	// the page is the deterministic one, which is a complete briefing.
	Narrated bool `json:"narrated" bun:"narrated,type:BOOLEAN,notnull,default:false"`
	// FailureReason says why the facts could not be gathered, for a failed
	// briefing.
	FailureReason string `json:"failureReason" bun:"failure_reason,type:TEXT,nullzero"`

	EmailedAt *int64 `json:"emailedAt" bun:"emailed_at,type:BIGINT,nullzero"`
	ReadAt    *int64 `json:"readAt"    bun:"read_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (b *Briefing) GetID() pulid.ID { return b.ID }

func (b *Briefing) GetTableName() string { return "assistant_briefings" }

func (b *Briefing) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *Briefing) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *Briefing) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("brf_")
		}
		if b.CreatedAt == 0 {
			b.CreatedAt = now
		}
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

// Normalize bounds what a person reads to what the columns hold, and drops
// a line whose path could leave the application. A briefing that runs long
// is cut rather than refused: the morning is not worth failing over.
func (b *Briefing) Normalize() {
	b.Headline = stringutils.Ellipsize(strings.TrimSpace(b.Headline), MaxHeadlineLength)
	if len(b.Sections) > MaxSections {
		b.Sections = b.Sections[:MaxSections]
	}
	for index := range b.Sections {
		section := &b.Sections[index]
		section.Title = stringutils.Ellipsize(strings.TrimSpace(section.Title), maxLabelLength)
		section.Summary = stringutils.Ellipsize(strings.TrimSpace(section.Summary), maxBodyLength)
		section.Body = stringutils.Ellipsize(strings.TrimSpace(section.Body), maxBodyLength)
		if !stringutils.IsSafeAppPath(section.Path) {
			section.Path = ""
		}
		if len(section.Items) > MaxSectionItems {
			section.Items = section.Items[:MaxSectionItems]
		}
		for itemIndex := range section.Items {
			item := &section.Items[itemIndex]
			item.Label = stringutils.Ellipsize(strings.TrimSpace(item.Label), maxLabelLength)
			item.Value = stringutils.Ellipsize(strings.TrimSpace(item.Value), maxValueLength)
			if !item.IsSafePath() {
				item.Path = ""
			}
		}
	}
	if b.Sections == nil {
		b.Sections = []Section{}
	}
	if b.Facts == nil {
		b.Facts = map[string]any{}
	}
}

func (b *Briefing) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&b.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&b.RoleKey,
			validation.Required.Error("Role is required"),
			domainvalidation.ValidEnum[RoleKey]("Role is invalid"),
		),
		validation.Field(&b.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&b.BriefingDate,
			validation.Required.Error("Briefing date is required"),
			validation.Length(10, 10).Error("Briefing date must be a calendar day"),
		),
		validation.Field(&b.Headline,
			validation.Length(0, MaxHeadlineLength).Error("Headline is too long"),
		),
		validation.Field(&b.Sections,
			validation.Length(0, MaxSections).Error("A briefing carries at most {0} sections"),
		),
	))

	if !timeutils.IsCalendarDate(b.BriefingDate) {
		multiErr.Add("briefingDate", errortypes.ErrInvalid, "Briefing date must be YYYY-MM-DD")
	}
	for index := range b.Sections {
		b.validateSection(multiErr, index)
	}
}

func (b *Briefing) validateSection(multiErr *errortypes.MultiError, index int) {
	section := b.Sections[index]
	field := "sections[" + strconv.Itoa(index) + "]"
	if !section.Key.IsValid() {
		multiErr.Add(field+".key", errortypes.ErrInvalid, "Section is not one this page has")
	}
	if strings.TrimSpace(section.Title) == "" {
		multiErr.Add(field+".title", errortypes.ErrRequired, "Section title is required")
	}
	if section.Path != "" && !stringutils.IsSafeAppPath(section.Path) {
		multiErr.Add(field+".path", errortypes.ErrInvalid, "Path must be an application path")
	}
	if len(section.Path) > maxPathLength {
		multiErr.Add(field+".path", errortypes.ErrInvalid, "Path is too long")
	}
	for itemIndex, item := range section.Items {
		if !item.IsSafePath() {
			multiErr.Add(
				field+".items["+strconv.Itoa(itemIndex)+"].path",
				errortypes.ErrInvalid,
				"Path must be an application path",
			)
		}
	}
}
