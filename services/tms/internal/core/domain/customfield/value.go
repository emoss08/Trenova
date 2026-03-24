package customfield

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomFieldValue)(nil)

type CustomFieldValue struct {
	bun.BaseModel `bun:"table:custom_field_values,alias:cfv" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	DefinitionID   pulid.ID `json:"definitionId"   bun:"definition_id,type:VARCHAR(100),notnull"`
	ResourceType   string   `json:"resourceType"   bun:"resource_type,type:VARCHAR(100),notnull"`
	ResourceID     string   `json:"resourceId"     bun:"resource_id,type:VARCHAR(100),notnull"`
	Value          any      `json:"value"          bun:"value,type:JSONB,notnull"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`

	Definition *CustomFieldDefinition `json:"definition,omitempty" bun:"rel:belongs-to,join:definition_id=id"`
}

func (v *CustomFieldValue) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("cfv_")
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}

func (v *CustomFieldValue) GetID() pulid.ID {
	return v.ID
}

func (v *CustomFieldValue) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *CustomFieldValue) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}
