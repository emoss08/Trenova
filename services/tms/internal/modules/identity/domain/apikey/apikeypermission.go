package apikey

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Permission struct {
	keel.Table `keel:"api_key_permissions"`

	ID             pulid.ID               `json:"id"             keel:"id,pk,autoid:akp_"`
	APIKeyID       pulid.ID               `json:"apiKeyId"       keel:"api_key_id"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" keel:"business_unit_id,tenant"`
	OrganizationID pulid.ID               `json:"organizationId" keel:"organization_id,tenant"`
	Resource       string                 `json:"resource"       keel:"resource"`
	Operations     []permission.Operation `json:"operations"     keel:"operations"`
	DataScope      permission.DataScope   `json:"dataScope"      keel:"data_scope"`
	CreatedAt      int64                  `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64                  `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (p *Permission) BeforeUpdate(_ context.Context) error {
	p.Operations = permission.Normalize(p.Operations)

	return nil
}

func (p *Permission) BeforeCreate(_ context.Context) error {
	p.Operations = permission.Normalize(p.Operations)
	if p.DataScope == "" {
		p.DataScope = permission.DataScopeOrganization
	}

	return nil
}

func (p *Permission) BeforeSave(_ context.Context) error {
	p.Operations = permission.Normalize(p.Operations)
	return nil
}

func (p *Permission) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.APIKeyID, validation.Required.Error("API key is required")),
		validation.Field(&p.Resource,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxResourceLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		// A grant with no operations authorizes nothing, and one naming an
		// operation the system does not define can never be evaluated.
		validation.Field(&p.Operations,
			validation.Required.Error("At least one operation is required"),
			validation.Each(domainvalidation.ValidEnum[permission.Operation](
				"Operation is not one this system defines",
			)),
		),
		validation.Field(&p.DataScope,
			validation.Required.Error("Data scope is required"),
			domainvalidation.ValidEnum[permission.DataScope]("Data scope is invalid"),
		),
	))
}

const maxResourceLength = 100
