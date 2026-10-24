package agentprofile

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Guidance)(nil)
)

const (
	MaxGuidanceChars = 500
	// MaxActiveGuidance bounds what one profile is written from, so the prompt
	// that writes it stays small and a long list cannot drown the facts.
	MaxActiveGuidance = 50
)

// GuidanceStatus is where a line stands. Active lines shape the next version;
// a Suggested line was offered about the organization by someone who may not
// manage shared memory, or by an agent, and waits for an administrator to
// approve it or dismiss it; Retired and Dismissed lines are kept for the record.
type GuidanceStatus string

const (
	GuidanceStatusActive    = GuidanceStatus("Active")
	GuidanceStatusSuggested = GuidanceStatus("Suggested")
	GuidanceStatusRetired   = GuidanceStatus("Retired")
	GuidanceStatusDismissed = GuidanceStatus("Dismissed")
)

func (s GuidanceStatus) IsValid() bool {
	switch s {
	case GuidanceStatusActive,
		GuidanceStatusSuggested,
		GuidanceStatusRetired,
		GuidanceStatusDismissed:
		return true
	default:
		return false
	}
}

// Closed reports a line nobody reads any more: retired after shaping the
// profile, or dismissed before it ever did.
func (s GuidanceStatus) Closed() bool {
	return s == GuidanceStatusRetired || s == GuidanceStatusDismissed
}

func AllGuidanceStatuses() []GuidanceStatus {
	return []GuidanceStatus{
		GuidanceStatusActive,
		GuidanceStatusSuggested,
		GuidanceStatusRetired,
		GuidanceStatusDismissed,
	}
}

// GuidanceSource says who worded the line: a person typing it, or an agent
// writing down what a person said in a conversation.
type GuidanceSource string

const (
	GuidanceSourcePerson = GuidanceSource("Person")
	GuidanceSourceAgent  = GuidanceSource("Agent")
)

func (s GuidanceSource) IsValid() bool {
	return s == GuidanceSourcePerson || s == GuidanceSourceAgent
}

func AllGuidanceSources() []GuidanceSource {
	return []GuidanceSource{GuidanceSourcePerson, GuidanceSourceAgent}
}

type Guidance struct {
	keel.Table `keel:"agent_profile_guidance"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:aprg_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Subject     Subject        `json:"subject" keel:"subject"`
	UserID      *pulid.ID      `json:"userId" keel:"user_id"`
	Content     string         `json:"content" keel:"content"`
	Status      GuidanceStatus `json:"status" keel:"status"`
	Source      GuidanceSource `json:"source" keel:"source"`
	CreatedByID pulid.ID       `json:"createdById" keel:"created_by_id"`
	ThreadID    *pulid.ID      `json:"threadId" keel:"thread_id"`
	RetiredByID *pulid.ID      `json:"retiredById" keel:"retired_by_id"`
	RetiredAt   *int64         `json:"retiredAt" keel:"retired_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	CreatedBy    *tenant.User         `json:"createdBy,omitempty" keel:"belongs:created_by_id"`
}

func (g *Guidance) Validate(multiErr *errortypes.MultiError) {
	g.Content = strings.TrimSpace(g.Content)

	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&g.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&g.Subject,
			validation.Required.Error("Subject is required"),
			domainvalidation.ValidEnum[Subject]("Subject is invalid"),
		),
		validation.Field(&g.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[GuidanceStatus]("Status is invalid"),
		),
		validation.Field(&g.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[GuidanceSource]("Source is invalid"),
		),
		validation.Field(&g.CreatedByID, validation.Required.Error("Who gave it is required")),
		validation.Field(&g.Content,
			validation.Required.Error("Guidance says something"),
			validation.RuneLength(1, MaxGuidanceChars).
				Error("Guidance is at most 500 characters"),
		),
	))

	if (g.Subject == SubjectUser) != (g.UserID != nil && g.UserID.IsNotNil()) {
		multiErr.Add("userId", errortypes.ErrInvalid, "Only guidance about a person names them")
	}
	if g.Status.Closed() != (g.RetiredAt != nil) {
		multiErr.Add(
			"retiredAt",
			errortypes.ErrInvalid,
			"Only retired or dismissed guidance says when",
		)
	}
}

func (g *Guidance) Retire(by pulid.ID, at int64) {
	g.Status = GuidanceStatusRetired
	g.RetiredByID = &by
	g.RetiredAt = &at
}

// Approve makes an offered line one the next version honours.
func (g *Guidance) Approve() { g.Status = GuidanceStatusActive }

// Dismiss closes an offered line without it ever shaping the profile; who
// closed it and when are kept as for a retired line.
func (g *Guidance) Dismiss(by pulid.ID, at int64) {
	g.Status = GuidanceStatusDismissed
	g.RetiredByID = &by
	g.RetiredAt = &at
}

func (g *Guidance) IsActive() bool { return g != nil && g.Status == GuidanceStatusActive }

func (g *Guidance) IsSuggested() bool { return g != nil && g.Status == GuidanceStatusSuggested }

func (g *Guidance) GetID() pulid.ID { return g.ID }

func (g *Guidance) GetOrganizationID() pulid.ID { return g.OrganizationID }

func (g *Guidance) GetBusinessUnitID() pulid.ID { return g.BusinessUnitID }

func (g *Guidance) GetTableName() string { return "agent_profile_guidance" }

func (g *Guidance) GetCreatedAt() int64 { return g.CreatedAt }

func (g *Guidance) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aprg",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "content", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (g *Guidance) BeforeCreate(_ context.Context) error {
	if g.Status == "" {
		g.Status = GuidanceStatusActive
	}

	return nil
}
