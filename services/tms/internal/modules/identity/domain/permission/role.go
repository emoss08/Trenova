package permission

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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
	keel.Table `keel:"roles"`

	ID                 pulid.ID           `json:"id" keel:"id,pk,autoid:rol_"`
	BusinessUnitID     pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name               string             `json:"name" keel:"name"`
	Description        string             `json:"description" keel:"description,nullzero"`
	CoreResponsibility CoreResponsibility `json:"coreResponsibility" keel:"core_responsibility,nullzero"`
	ParentRoleIDs      []pulid.ID         `json:"parentRoleIds" keel:"parent_role_ids"`
	MaxSensitivity     FieldSensitivity   `json:"maxSensitivity" keel:"max_sensitivity"`
	IsSystem           bool               `json:"isSystem" keel:"is_system"`
	CreatedBy          pulid.ID           `json:"createdBy" keel:"created_by,nullzero"`
	CreatedAt          int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	Permissions []*ResourcePermission `json:"permissions,omitempty" keel:"many:role_id"`
}

func (r *Role) BeforeCreate(ctx context.Context) error {
	if r.MaxSensitivity == "" {
		r.MaxSensitivity = SensitivityInternal
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
