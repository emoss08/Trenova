package apikey

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Permission struct {
	bun.BaseModel `bun:"table:api_key_permissions,alias:akp" json:"-"`
	keel.Table    `keel:"api_key_permissions"`

	ID             pulid.ID               `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:akp_"`
	APIKeyID       pulid.ID               `json:"apiKeyId"       bun:"api_key_id,type:VARCHAR(100),notnull" keel:"api_key_id"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	OrganizationID pulid.ID               `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	Resource       string                 `json:"resource"       bun:"resource,type:VARCHAR(100),notnull" keel:"resource"`
	Operations     []permission.Operation `json:"operations"     bun:"operations,type:TEXT[],array,notnull" keel:"operations"`
	DataScope      permission.DataScope   `json:"dataScope"      bun:"data_scope,type:VARCHAR(20),notnull,default:'organization'" keel:"data_scope"`
	CreatedAt      int64                  `json:"createdAt"      bun:"created_at,notnull" keel:"created_at,autocreate"`
	UpdatedAt      int64                  `json:"updatedAt"      bun:"updated_at,notnull" keel:"updated_at,autoupdate"`
}

func (p *Permission) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("akp_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
		p.Operations = permission.Normalize(p.Operations)
	case *bun.UpdateQuery:
		p.UpdatedAt = now
		p.Operations = permission.Normalize(p.Operations)
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
