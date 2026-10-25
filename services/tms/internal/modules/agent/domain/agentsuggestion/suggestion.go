package agentsuggestion

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Suggestion)(nil)
)

type Suggestion struct {
	keel.Table `keel:"agent_suggestions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:agsg_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	UserID           pulid.ID  `json:"userId"           keel:"user_id"`
	SuggestedAgentID pulid.ID  `json:"suggestedAgentId" keel:"suggested_agent_id"`
	CurrentAgentID   *pulid.ID `json:"currentAgentId"   keel:"current_agent_id"`
	Surface          Surface   `json:"surface"          keel:"surface"`
	Outcome          Outcome   `json:"outcome"          keel:"outcome"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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
