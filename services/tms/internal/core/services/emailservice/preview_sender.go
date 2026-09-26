package emailservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

var _ services.EmailSenderResolver = (*Service)(nil)

// ResolveSender reads the profile Send would use and the recipients it would
// refuse, and sends nothing.
func (s *Service) ResolveSender(
	ctx context.Context,
	req *services.SendEmailRequest,
) (*services.EmailSender, error) {
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}

	suppressed, err := s.suppressedRecipients(ctx, req.TenantInfo, req.To, false)
	if err != nil {
		return nil, err
	}

	return &services.EmailSender{
		ProfileID:  profile.ID,
		Email:      senderEmail(req, profile),
		Name:       profile.SenderName,
		ReplyTo:    profile.ReplyToEmail,
		Suppressed: suppressed,
	}, nil
}

func senderEmail(req *services.SendEmailRequest, profile *email.Profile) string {
	if from := strings.TrimSpace(req.FromEmail); from != "" {
		return from
	}

	return profile.SenderEmail
}

// suppressedRecipients lists the recipients the suppression list refuses.
// Send stops at the first one; a preview names every one.
func (s *Service) suppressedRecipients(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recipients []string,
	firstOnly bool,
) ([]string, error) {
	var suppressed []string
	for _, recipient := range recipients {
		refused, err := s.repo.HasSuppression(ctx, tenantInfo, recipient)
		if err != nil {
			return nil, err
		}
		if !refused {
			continue
		}
		suppressed = append(suppressed, recipient)
		if firstOnly {
			return suppressed, nil
		}
	}

	return suppressed, nil
}
