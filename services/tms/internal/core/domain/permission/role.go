package permission

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
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

	ID                  pulid.ID           `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID      pulid.ID           `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100)"`
	OrganizationID      pulid.ID           `json:"organizationId"      bun:"organization_id,type:VARCHAR(100)"`
	Name                string             `json:"name"                bun:"name,type:VARCHAR(255),notnull"`
	Description         string             `json:"description"         bun:"description,type:TEXT"`
	CoreResponsibility  CoreResponsibility `json:"coreResponsibility"  bun:"core_responsibility,type:VARCHAR(50),nullzero"`
	ParentRoleIDs       []pulid.ID         `json:"parentRoleIds"       bun:"parent_role_ids,type:TEXT[],array"`
	MaxSensitivity      FieldSensitivity   `json:"maxSensitivity"      bun:"max_sensitivity,type:VARCHAR(20),notnull,default:'internal'"`
	IsSystem            bool               `json:"isSystem"            bun:"is_system,default:false"`
	IsOrgAdmin          bool               `json:"isOrgAdmin"          bun:"is_org_admin,default:false"`
	IsBusinessUnitAdmin bool               `json:"isBusinessUnitAdmin" bun:"is_business_unit_admin,default:false"`
	CreatedBy           pulid.ID           `json:"createdBy"           bun:"created_by,type:VARCHAR(100)"`
	CreatedAt           int64              `json:"createdAt"           bun:"created_at,notnull"`
	UpdatedAt           int64              `json:"updatedAt"           bun:"updated_at,notnull"`

	Permissions []*ResourcePermission `json:"permissions,omitempty" bun:"rel:has-many,join:id=role_id"`
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
