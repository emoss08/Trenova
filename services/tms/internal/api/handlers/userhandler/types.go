package userhandler

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
)

type SwitchOrganizationRequest struct {
	OrganizationID string `json:"organizationId" binding:"required"`
}

type ReplaceOrganizationMembershipsRequest struct {
	OrganizationIDs []string `json:"organizationIds"`
}

type UpdateMySettingsRequest struct {
	Timezone   string                 `json:"timezone"`
	TimeFormat domaintypes.TimeFormat `json:"timeFormat"`
	Locale     string                 `json:"locale"`
}

type ProfilePictureURLResponse struct {
	URL string `json:"url"`
}

type ChangeMyPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}
