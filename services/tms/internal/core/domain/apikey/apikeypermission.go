package apikey

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
