package customfield

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomFieldValue)(nil)

type CustomFieldValue struct {
	bun.BaseModel `bun:"table:custom_field_values,alias:cfv" json:"-"`
	keel.Table    `keel:"custom_field_values"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:cfv_"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk" keel:"business_unit_id,pk,tenant"`
	DefinitionID   pulid.ID `json:"definitionId"   bun:"definition_id,type:VARCHAR(100),notnull" keel:"definition_id"`
	ResourceType   string   `json:"resourceType"   bun:"resource_type,type:VARCHAR(100),notnull" keel:"resource_type"`
	ResourceID     string   `json:"resourceId"     bun:"resource_id,type:VARCHAR(100),notnull" keel:"resource_id"`
	Value          any      `json:"value"          bun:"value,type:JSONB,notnull" keel:"value,jsonb"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull" keel:"updated_at,autoupdate"`

	Definition *CustomFieldDefinition `json:"definition,omitempty" bun:"rel:belongs-to,join:definition_id=id" keel:"belongs:definition_id"`
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

func (v *CustomFieldValue) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.DefinitionID,
			validation.Required.Error("Custom field definition is required"),
		),
		// A value with no owner is unreachable: nothing would ever load it and
		// nothing would ever delete it.
		validation.Field(&v.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxResourceTypeLength).
				Error("Resource type cannot be longer than 100 characters"),
		),
		validation.Field(&v.ResourceID,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxResourceIDLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		validation.Field(&v.Value, validation.NotNil.Error("Value is required")),
	))
}

const (
	maxResourceTypeLength = 100
	maxResourceIDLength   = 100
)
