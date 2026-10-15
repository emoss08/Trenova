package agentsuggestion

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Suggestion)(nil)
	_ validationframework.TenantedEntity = (*Suggestion)(nil)
)

type Suggestion struct {
	bun.BaseModel `bun:"table:agent_suggestions,alias:agsg" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	UserID           pulid.ID  `json:"userId"           bun:"user_id,type:VARCHAR(100),notnull"`
	SuggestedAgentID pulid.ID  `json:"suggestedAgentId" bun:"suggested_agent_id,type:VARCHAR(100),notnull"`
	CurrentAgentID   *pulid.ID `json:"currentAgentId"   bun:"current_agent_id,type:VARCHAR(100),nullzero"`
	Surface          Surface   `json:"surface"          bun:"surface,type:VARCHAR(20),notnull"`
	Outcome          Outcome   `json:"outcome"          bun:"outcome,type:VARCHAR(20),notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (s *Suggestion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&s.UserID, validation.Required.Error("User is required")),
		validation.Field(&s.SuggestedAgentID,
			validation.Required.Error("The agent suggested is required"),
		),
		validation.Field(&s.Surface,
			validation.Required.Error("Surface is required"),
			domainvalidation.ValidEnum[Surface]("Surface is invalid"),
		),
		validation.Field(&s.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[Outcome]("Outcome is invalid"),
		),
	))

	if s.CurrentAgentID != nil && *s.CurrentAgentID == s.SuggestedAgentID {
		multiErr.Add(
			"suggestedAgentId",
			errortypes.ErrInvalid,
			"An agent is suggested in place of another one",
		)
	}
}

func (s *Suggestion) GetID() pulid.ID { return s.ID }

func (s *Suggestion) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *Suggestion) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *Suggestion) GetTableName() string { return "agent_suggestions" }

func (s *Suggestion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("agsg_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
