package permission

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Role)(nil)
	_ domaintypes.PostgresSearchable = (*Role)(nil)
)

type CoreResponsibility string

const (
	CoreResponsibilityBilling    = CoreResponsibility("Billing")
	CoreResponsibilityOperations = CoreResponsibility("Operations")
	CoreResponsibilityFinance    = CoreResponsibility("Finance")
	CoreResponsibilityLeadership = CoreResponsibility("Leadership")
)

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r" json:"-"`
	keel.Table    `keel:"roles"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                 pulid.ID           `json:"id"                 bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:rol_"`
	BusinessUnitID     pulid.ID           `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID           `json:"organizationId"     bun:"organization_id,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`
	Name               string             `json:"name"               bun:"name,type:VARCHAR(255),notnull" keel:"name"`
	Description        string             `json:"description"        bun:"description,type:TEXT" keel:"description,nullzero"`
	CoreResponsibility CoreResponsibility `json:"coreResponsibility" bun:"core_responsibility,type:VARCHAR(50),nullzero" keel:"core_responsibility,nullzero"`
	ParentRoleIDs      []pulid.ID         `json:"parentRoleIds"      bun:"parent_role_ids,type:TEXT[],array" keel:"parent_role_ids"`
	MaxSensitivity     FieldSensitivity   `json:"maxSensitivity"     bun:"max_sensitivity,type:VARCHAR(20),notnull,default:'internal'" keel:"max_sensitivity"`
	IsSystem           bool               `json:"isSystem"           bun:"is_system" keel:"is_system"`
	CreatedBy          pulid.ID           `json:"createdBy"          bun:"created_by,type:VARCHAR(100)" keel:"created_by,nullzero"`
	CreatedAt          int64              `json:"createdAt"          bun:"created_at,notnull" keel:"created_at,autocreate"`
	UpdatedAt          int64              `json:"updatedAt"          bun:"updated_at,notnull" keel:"updated_at,autoupdate"`

	Permissions []*ResourcePermission `json:"permissions,omitempty" bun:"rel:has-many,join:id=role_id" keel:"many:role_id"`
}

func (r *Role) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rol_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *Role) GetID() pulid.ID {
	return r.ID
}

func (r *Role) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *Role) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *Role) GetTableName() string {
	return "roles"
}

func (r *Role) GetBusinessUnitID() pulid.ID {
	return ""
}

func (r *Role) Validate(multiErr *errortypes.MultiError) {
	if err := validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
	); err != nil {
		multiErr.AddOzzoError(err)
	}

	// The grants arrive with the role and are written with it, so a malformed
	// one is reported against the role rather than failing at insert.
	for i, permission := range r.Permissions {
		if permission == nil {
			continue
		}
		permission.Validate(multiErr.WithIndex("permissions", i))
	}
}

func (r *Role) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "r",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
