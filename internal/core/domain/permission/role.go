package permission

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r" json:"-"`

	ID          pulid.ID      `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	Name        string        `json:"name"               bun:"name,type:VARCHAR(100),notnull"`
	Description string        `json:"description"        bun:"description,type:TEXT"`
	RoleType    RoleType      `json:"roleType"           bun:"role_type,type:role_type_enum,notnull"`
	IsSystem    bool          `json:"isSystem"           bun:"is_system,notnull,default:false"`
	Priority    int           `json:"priority"           bun:"priority,notnull,default:0"`
	Status      domain.Status `json:"status"             bun:"status,type:status_enum,notnull,default:'Active'"`
	ExpiresAt   *int64        `json:"expiresAt,omitzero" bun:"expires_at,nullzero"`
	CreatedAt   int64         `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64         `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnitID pulid.ID  `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId"         bun:"organization_id,type:VARCHAR(100)"`
	ParentRoleID   *pulid.ID `json:"parentRoleId,omitempty" bun:"parent_role_id,type:VARCHAR(100),nullzero"`

	Permissions []*Permission  `json:"permissions,omitzero" bun:"m2m:role_permissions,join:Role=Permission"`
	ParentRole  *Role          `json:"parentRole,omitempty" bun:"rel:belongs-to,join:parent_role_id=id"`
	ChildRoles  []*Role        `json:"childRoles,omitempty" bun:"rel:has-many,join:id=parent_role_id"`
	Metadata    map[string]any `json:"metadata,omitempty"   bun:"metadata,type:JSONB,default:'{}'::jsonb"`
}

func (r *Role) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(2, 100).Error("Name must be between 2 and 100 characters"),
		),
		validation.Field(&r.RoleType, validation.Required.Error("Role type is required")),
		validation.Field(&r.Status, validation.Required.Error("Status is required")),
		validation.Field(&r.Priority, validation.Min(0).Error("Priority must be non-negative")),
	)
}

var _ bun.BeforeAppendModelHook = (*Role)(nil)

func (r *Role) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID == "" {
			r.ID = pulid.MustNew("rol_")
		}
	}
	return nil
}

type RolePermission struct {
	bun.BaseModel  `bun:"table:role_permissions,alias:rp" json:"-"`
	BusinessUnitID pulid.ID    `bun:"business_unit_id,pk,type:VARCHAR(100),notnull" json:"businessUnitId"`
	OrganizationID pulid.ID    `bun:"organization_id,pk,type:VARCHAR(100),notnull"  json:"organizationId"`
	RoleID         pulid.ID    `bun:"role_id,pk,type:VARCHAR(100),notnull"          json:"roleId"`
	PermissionID   pulid.ID    `bun:"permission_id,pk,type:VARCHAR(100),notnull"    json:"permissionId"`
	Role           *Role       `bun:"rel:belongs-to,join:role_id=id"                json:"-"`
	Permission     *Permission `bun:"rel:belongs-to,join:permission_id=id"          json:"-"`
}
