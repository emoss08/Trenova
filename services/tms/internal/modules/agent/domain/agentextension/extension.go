package agentextension

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Extension struct {
	keel.Table `keel:"agent_extensions"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:aext_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	Type           Type           `json:"type"           keel:"type"`
	Enabled        bool           `json:"enabled"        keel:"enabled"`
	Availability   Availability   `json:"availability"   keel:"availability"`
	Configuration  map[string]any `json:"configuration"  keel:"configuration,jsonb"`
	EnabledByID    pulid.ID       `json:"enabledById"    keel:"enabled_by_id,nullzero"`
	EnabledAt      *int64         `json:"enabledAt"      keel:"enabled_at"`
	Version        int64          `json:"version"        keel:"version,version"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`

	EnabledBy *tenant.User `json:"enabledBy,omitempty" keel:"belongs:enabled_by_id"`
}

func (e *Extension) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[Type]("Type is not an extension this system supports"),
		),
		validation.Field(
			&e.Availability,
			validation.Required.Error("Availability is required"),
			domainvalidation.ValidEnum[Availability](
				"Availability must be AllAgents or SelectedAgents",
			),
		),
		validation.Field(&e.EnabledByID,
			validation.When(
				e.Enabled,
				validation.Required.Error("An enabled extension must record who enabled it"),
			),
		),
	))
}

func (e *Extension) Ready(spec Spec) bool {
	return e != nil && e.Enabled && spec.Configured(e.Configuration)
}

func (e *Extension) GrantsEveryAgent() bool {
	return e != nil && e.Enabled && e.Availability == AvailabilityAllAgents
}
