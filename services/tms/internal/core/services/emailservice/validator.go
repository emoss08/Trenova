package emailservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateProfile(_ context.Context, profile *email.Profile) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if profile == nil {
		multiErr.Add("", errortypes.ErrRequired, "Email profile is required")
		return multiErr
	}
	profile.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (v *Validator) ValidateSend(_ context.Context, req *services.SendEmailRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrRequired, "Email request is required")
		return multiErr
	}
	if req.Purpose == "" {
		req.Purpose = email.PurposeGeneral
	}
	if !email.IsValidPurpose(req.Purpose) {
		multiErr.Add("purpose", errortypes.ErrInvalid, "Invalid email purpose")
	}
	if len(req.To) == 0 {
		multiErr.Add("to", errortypes.ErrRequired, "At least one recipient is required")
	}
	for i, recipient := range req.To {
		if !strings.Contains(strings.TrimSpace(recipient), "@") {
			multiErr.WithIndex("to", i).Add("", errortypes.ErrInvalid, "Recipient email is invalid")
		}
	}
	if strings.TrimSpace(req.Subject) == "" {
		multiErr.Add("subject", errortypes.ErrRequired, "Subject is required")
	}
	if strings.TrimSpace(req.HTML) == "" && strings.TrimSpace(req.Text) == "" {
		multiErr.Add("body", errortypes.ErrRequired, "HTML or text body is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
