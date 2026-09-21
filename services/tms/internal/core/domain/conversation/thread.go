package conversation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const maxTitleLength = 200

// Thread is one person's conversation with one agent.
//
// Threads belong to a user rather than to an organization at large. A
// conversation reveals what someone was investigating and what the assistant
// showed them, which can include customer rates and worker records, so it is not
// shared reading material.
type Thread struct {
	bun.BaseModel `bun:"table:assistant_threads,alias:athr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	UserID            pulid.ID `json:"userId"            bun:"user_id,type:VARCHAR(100),notnull"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`

	Title  string       `json:"title"  bun:"title,type:VARCHAR(200),nullzero"`
	Status ThreadStatus `json:"status" bun:"status,type:VARCHAR(50),notnull"`

	// LastMessageAt orders a user's thread list without a join onto messages.
	LastMessageAt int64 `json:"lastMessageAt" bun:"last_message_at,notnull,default:0"`

	// PreferredProviderID is the model the person chose for this conversation.
	// Empty means the organization's priority order decides, which is the
	// default and what every thread did before the picker existed.
	//
	// It is a preference, not a pin: the router tries it first and falls through
	// to the rest if it fails, so a rate-limited free endpoint degrades into a
	// slower answer rather than no answer. Each message records the provider
	// that actually served it, which is how the reader sees the difference.
	PreferredProviderID pulid.ID `json:"preferredProviderId" bun:"preferred_provider_id,type:VARCHAR(100),nullzero"`

	// Origin is where the conversation began; Pinned keeps it at the top of
	// the Desk's rail. SubjectType and SubjectID name the record the
	// conversation is about when it was opened from one, so every turn
	// carries that record's context without the person restating it.
	Origin      ThreadOrigin      `json:"origin"      bun:"origin,type:VARCHAR(20),notnull,default:'Panel'"`
	Pinned      bool              `json:"pinned"      bun:"pinned,type:BOOLEAN,notnull,default:false"`
	SubjectType agent.SubjectType `json:"subjectType" bun:"subject_type,type:VARCHAR(50),nullzero"`
	SubjectID   pulid.ID          `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (t *Thread) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("athr_")
		}
		if t.Status == "" {
			t.Status = ThreadStatusActive
		}
		if t.Origin == "" {
			t.Origin = ThreadOriginPanel
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

func (t *Thread) GetID() pulid.ID { return t.ID }

func (t *Thread) GetTableName() string { return "assistant_threads" }

func (t *Thread) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "athr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (t *Thread) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.UserID, validation.Required.Error("User is required")),
		validation.Field(&t.AgentDefinitionID,
			validation.Required.Error("Agent is required"),
		),
		validation.Field(&t.Title,
			validation.Length(0, maxTitleLength).
				Error("Title cannot be longer than 200 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ThreadStatus]("Status is invalid"),
		),
		validation.Field(&t.Origin,
			validation.Required.Error("Origin is required"),
			domainvalidation.ValidEnum[ThreadOrigin]("Origin is invalid"),
		),
	))

	if t.SubjectType != "" && !t.SubjectType.IsValid() {
		multiErr.Add("subjectType", errortypes.ErrInvalid, "Subject type is invalid")
	}
	if t.SubjectID.IsNotNil() && t.SubjectType == "" {
		multiErr.Add("subjectType", errortypes.ErrRequired, "A subject needs a type")
	}
	if t.SubjectType != "" && t.SubjectID.IsNil() {
		multiErr.Add("subjectId", errortypes.ErrRequired, "A subject type needs an id")
	}
}

// HasSubject reports whether the conversation is about one record.
func (t *Thread) HasSubject() bool {
	return t.SubjectType != "" && t.SubjectID.IsNotNil()
}
