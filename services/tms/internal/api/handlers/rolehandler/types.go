package rolehandler

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/shared/pulid"
)

type AddPermissionRequest struct {
	Resource   string                 `json:"resource"   form:"resource"   binding:"required"`
	Operations []permission.Operation `json:"operations" form:"operations" binding:"required"`
	DataScope  permission.DataScope   `json:"dataScope"  form:"dataScope"  binding:"required"`
}
type AssignRoleRequest struct {
	UserID    pulid.ID `json:"userId"              form:"userId" binding:"required"`
	ExpiresAt *int64   `json:"expiresAt,omitempty"`
}

type UpsertHierarchyEdgeRequest struct {
	SeniorRoleID pulid.ID `json:"seniorRoleId" binding:"required"`
	JuniorRoleID pulid.ID `json:"juniorRoleId" binding:"required"`
}

type SaveConstraintRequest struct {
	ID          pulid.ID                `json:"id,omitempty"`
	Name        string                  `json:"name"                  binding:"required"`
	Description string                  `json:"description,omitempty"`
	Type        rbac.RoleConstraintType `json:"type"                  binding:"required"`
	MaxRoles    int                     `json:"maxRoles"              binding:"required"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	RoleIDs     []pulid.ID              `json:"roleIds"               binding:"required"`
}
