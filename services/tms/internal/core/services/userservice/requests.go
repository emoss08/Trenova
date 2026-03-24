package userservice

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UpdateMySettingsRequest struct {
	Timezone      string                 `json:"timezone"`
	TimeFormat    domaintypes.TimeFormat `json:"timeFormat"`
	ProfilePicURL string                 `json:"profilePicUrl"`
	ThumbnailURL  string                 `json:"thumbnailUrl"`
}

func (r *UpdateMySettingsRequest) Validate() error {
	me := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(
			&r.Timezone,
			validation.Required.Error("Timezone is required"),
		),
		validation.Field(
			&r.TimeFormat,
			validation.Required.Error("Time format is required"),
			validation.In(
				domaintypes.TimeFormat12Hour,
				domaintypes.TimeFormat24Hour,
			).Error("Time format must be either 12-hour or 24-hour"),
		),
		validation.Field(
			&r.ProfilePicURL,
			validation.Length(0, 255).Error("Profile picture URL must be 255 characters or fewer"),
		),
		validation.Field(
			&r.ThumbnailURL,
			validation.Length(0, 255).Error("Thumbnail URL must be 255 characters or fewer"),
		),
	)

	me.AddOzzoError(err)
	if me.HasErrors() {
		return me
	}

	return nil
}

type ChangeMyPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (r *ChangeMyPasswordRequest) Validate() error {
	me := errortypes.NewMultiError()

	if r.CurrentPassword == "" {
		me.Add("currentPassword", errortypes.ErrRequired, "Current password is required")
	}
	if r.NewPassword == "" {
		me.Add("newPassword", errortypes.ErrRequired, "New password is required")
	}
	if r.ConfirmPassword == "" {
		me.Add("confirmPassword", errortypes.ErrRequired, "Password confirmation is required")
	}
	if r.NewPassword != "" && r.ConfirmPassword != "" && r.NewPassword != r.ConfirmPassword {
		me.Add("confirmPassword", errortypes.ErrInvalid, "Password confirmation must match the new password")
	}
	if r.CurrentPassword != "" && r.NewPassword != "" && r.CurrentPassword == r.NewPassword {
		me.Add("newPassword", errortypes.ErrInvalid, "New password must be different from the current password")
	}

	if me.HasErrors() {
		return me
	}

	return nil
}
