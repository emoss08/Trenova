package userhandler

type SwitchOrganizationRequest struct {
	OrganizationID string `json:"organizationId" binding:"required"`
}

type ProfilePictureURLResponse struct {
	URL string `json:"url"`
}

type ChangeMyPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}
