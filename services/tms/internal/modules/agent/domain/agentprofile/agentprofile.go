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
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Profile)(nil)
)

const (
	MaxProfileChars      = 3000
	MaxProfileErrorChars = 1000
	MaxFacts             = 40
	MaxFactKeyChars      = 60
	MaxFactTextChars     = 400
	DigestChars          = 64
)

type Subject string

const (
	SubjectOrganization = Subject("Organization")
	SubjectUser         = Subject("User")
)

func (s Subject) IsValid() bool {
	return s == SubjectOrganization || s == SubjectUser
}

func AllSubjects() []Subject {
	return []Subject{SubjectOrganization, SubjectUser}
}

type Status string

const (
	StatusCurrent    = Status("Current")
	StatusSuperseded = Status("Superseded")
	StatusFailed     = Status("Failed")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusCurrent, StatusSuperseded, StatusFailed:
		return true
	default:
		return false
	}
}

func AllStatuses() []Status {
	return []Status{StatusCurrent, StatusSuperseded, StatusFailed}
}

type Trigger string

const (
	TriggerScheduled = Trigger("Scheduled")
	TriggerRequested = Trigger("Requested")
	TriggerGuidance  = Trigger("Guidance")
)

func (t Trigger) IsValid() bool {
	switch t {
	case TriggerScheduled, TriggerRequested, TriggerGuidance:
		return true
	default:
		return false
	}
}

func AllTriggers() []Trigger {
	return []Trigger{TriggerScheduled, TriggerRequested, TriggerGuidance}
}

// Fact is one thing worked out from the records by a query, never by a model:
// Key names what it measures ("fleet.tractors") and Text says it in a line.
type Fact struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type Facts []Fact

func (f Facts) Normalized() Facts {
	out := make(Facts, 0, min(len(f), MaxFacts))
	seen := make(map[string]struct{}, len(f))
	for _, fact := range f {
		key := stringutils.TruncateRunes(strings.TrimSpace(fact.Key), MaxFactKeyChars)
		text := stringutils.TruncateRunes(strings.TrimSpace(fact.Text), MaxFactTextChars)
		if key == "" || text == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Fact{Key: key, Text: text})
		if len(out) == MaxFacts {
			break
		}
	}

	return out
}

type Profile struct {
	keel.Table `keel:"agent_profiles"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:aprf_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Subject       Subject    `json:"subject" keel:"subject"`
	UserID        *pulid.ID  `json:"userId" keel:"user_id"`
	Status        Status     `json:"status" keel:"status"`
	Trigger       Trigger    `json:"trigger" keel:"trigger"`
	RequestedByID *pulid.ID  `json:"requestedById" keel:"requested_by_id"`
	Content       string     `json:"content" keel:"content,nullzero"`
	Facts         Facts      `json:"facts" keel:"facts,jsonb"`
	GuidanceIDs   []pulid.ID `json:"guidanceIds" keel:"guidance_ids,jsonb"`
	Digest        string     `json:"digest" keel:"digest"`

	Model        string   `json:"model" keel:"model,nullzero"`
	ProviderID   pulid.ID `json:"providerId" keel:"provider_id,nullzero"`
	InputTokens  int      `json:"inputTokens" keel:"input_tokens"`
	OutputTokens int      `json:"outputTokens" keel:"output_tokens"`
	ErrorMessage string   `json:"errorMessage" keel:"error_message,nullzero"`
	CheckedAt    int64    `json:"checkedAt" keel:"checked_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (p *Profile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.Subject,
			validation.Required.Error("Subject is required"),
			domainvalidation.ValidEnum[Subject]("Subject is invalid"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&p.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[Trigger]("Trigger is invalid"),
		),
		validation.Field(&p.Digest,
			validation.Required.Error("Digest is required"),
			validation.Length(DigestChars, DigestChars).Error("Digest is not a SHA-256 hex digest"),
		),
		validation.Field(&p.Content,
			validation.When(p.Status != StatusFailed,
				validation.Required.Error("A profile says something"),
				validation.RuneLength(1, MaxProfileChars).
					Error("A profile is at most 3,000 characters"),
			),
		),
	))

	if (p.Subject == SubjectUser) != (p.UserID != nil && p.UserID.IsNotNil()) {
		multiErr.Add("userId", errortypes.ErrInvalid, "Only a person's profile names the person")
	}
	if (p.Status == StatusFailed) != (strings.TrimSpace(p.ErrorMessage) != "") {
		multiErr.Add("errorMessage", errortypes.ErrInvalid, "Only a failed version says why")
	}
	if len(p.Facts) > MaxFacts {
		multiErr.Add("facts", errortypes.ErrInvalid, "A profile is made from at most 40 facts")
	}
}

// Fail records a version that could not be written. It never becomes what the
// agents are told; the version before it stays current.
func (p *Profile) Fail(message string) {
	p.Status = StatusFailed
	p.Content = ""
	p.ErrorMessage = stringutils.TruncateRunes(strings.TrimSpace(message), MaxProfileErrorChars)
	if p.ErrorMessage == "" {
		p.ErrorMessage = "The profile could not be written."
	}
}

func (p *Profile) IsCurrent() bool { return p != nil && p.Status == StatusCurrent }

func (p *Profile) GetID() pulid.ID { return p.ID }

func (p *Profile) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Profile) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Profile) GetTableName() string { return "agent_profiles" }

func (p *Profile) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Profile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aprf",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "content", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "subject", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (p *Profile) BeforeCreate(_ context.Context) error {
	if p.Facts == nil {
		p.Facts = Facts{}
	}
	if p.GuidanceIDs == nil {
		p.GuidanceIDs = []pulid.ID{}
	}
	if p.CheckedAt == 0 {
		p.CheckedAt = timeutils.NowUnix()
	}

	return nil
}
