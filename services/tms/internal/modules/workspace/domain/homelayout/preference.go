package homelayout

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Preference)(nil)
)

// Preference is one user's home screen document. A row exists only once the
// user has changed something; until then they resolve against a preset.
type Preference struct {
	keel.Table `keel:"home_preferences"`

	ID             pulid.ID  `json:"id" keel:"id,pk,autoid:hpf_"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID  `json:"userId" keel:"user_id"`
	Preferences    *Document `json:"preferences" keel:"preferences,jsonb"`
	Version        int64     `json:"version" keel:"version,version"`
	CreatedAt      int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty" keel:"belongs:user_id"`
}

func (p *Preference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.UserID, validation.Required.Error("User is required")),
		validation.Field(&p.Preferences,
			validation.Required.Error("Preferences document is required"),
		),
	))

	if p.Preferences != nil {
		p.Preferences.Validate(multiErr)
	}
}

func (p *Preference) GetID() pulid.ID { return p.ID }

func (p *Preference) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Preference) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Preference) GetTableName() string { return "home_preferences" }
