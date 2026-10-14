package agentprofile

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Guidance)(nil)
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
	case GuidanceStatusActive, GuidanceStatusSuggested, GuidanceStatusRetired, GuidanceStatusDismissed:
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
	bun.BaseModel `bun:"table:agent_profile_guidance,alias:aprg" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	Subject     Subject        `json:"subject"     bun:"subject,type:VARCHAR(20),notnull"`
	UserID      *pulid.ID      `json:"userId"      bun:"user_id,type:VARCHAR(100),nullzero"`
	Content     string         `json:"content"     bun:"content,type:TEXT,notnull"`
	Status      GuidanceStatus `json:"status"      bun:"status,type:VARCHAR(20),notnull,default:'Active'"`
	Source      GuidanceSource `json:"source"      bun:"source,type:VARCHAR(20),notnull"`
	CreatedByID pulid.ID       `json:"createdById" bun:"created_by_id,type:VARCHAR(100),notnull"`
	ThreadID    *pulid.ID      `json:"threadId"    bun:"thread_id,type:VARCHAR(100),nullzero"`
	RetiredByID *pulid.ID      `json:"retiredById" bun:"retired_by_id,type:VARCHAR(100),nullzero"`
	RetiredAt   *int64         `json:"retiredAt"   bun:"retired_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	CreatedBy    *tenant.User         `bun:"rel:belongs-to,join:created_by_id=id"    json:"createdBy,omitempty"`
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

func (g *Guidance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("aprg_")
		}
		if g.Status == "" {
			g.Status = GuidanceStatusActive
		}
		g.CreatedAt = now
		g.UpdatedAt = now
	case *bun.UpdateQuery:
		g.UpdatedAt = now
	}

	return nil
}
