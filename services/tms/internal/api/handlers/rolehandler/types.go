package rolehandler

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
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
