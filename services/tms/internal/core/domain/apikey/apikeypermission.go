package apikey

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Permission struct {
	bun.BaseModel `bun:"table:api_key_permissions,alias:akp" json:"-"`

	ID             pulid.ID               `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	APIKeyID       pulid.ID               `json:"apiKeyId"       bun:"api_key_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID               `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	Resource       string                 `json:"resource"       bun:"resource,type:VARCHAR(100),notnull"`
	Operations     []permission.Operation `json:"operations"     bun:"operations,type:TEXT[],array,notnull"`
	DataScope      permission.DataScope   `json:"dataScope"      bun:"data_scope,type:VARCHAR(20),notnull,default:'organization'"`
	CreatedAt      int64                  `json:"createdAt"      bun:"created_at,notnull"`
	UpdatedAt      int64                  `json:"updatedAt"      bun:"updated_at,notnull"`
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
		p.Operations = permission.ExpandWithDependencies(
			permission.NewOperationSet(p.Operations...),
		).ToSlice()
	case *bun.UpdateQuery:
		p.UpdatedAt = now
		p.Operations = permission.ExpandWithDependencies(
			permission.NewOperationSet(p.Operations...),
		).ToSlice()
	}

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
