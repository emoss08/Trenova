package userhandler

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type SimulatePermissionsRequest struct {
	AddRoleIDs    []pulid.ID `json:"addRoleIds"`
	RemoveRoleIDs []pulid.ID `json:"removeRoleIds"`
}

type SwitchOrganizationRequest struct {
	OrganizationID string `json:"organizationId" binding:"required"`
}

type ReplaceOrganizationMembershipsRequest struct {
	OrganizationIDs []string `json:"organizationIds"`
}

type UpdateMySettingsRequest struct {
	Timezone      string                 `json:"timezone"`
	TimeFormat    domaintypes.TimeFormat `json:"timeFormat"`
	ProfilePicURL string                 `json:"profilePicUrl"`
	ThumbnailURL  string                 `json:"thumbnailUrl"`
}

type ChangeMyPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}
