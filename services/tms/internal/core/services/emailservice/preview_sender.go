package emailservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

var _ services.EmailSenderResolver = (*Service)(nil)

// ResolveSender reads the profile Send would use and the recipients it would
// skip, and sends nothing.
func (s *Service) ResolveSender(
	ctx context.Context,
	req *services.SendEmailRequest,
) (*services.EmailSender, error) {
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}

	split, err := s.splitSuppressed(ctx, req)
	if err != nil {
		return nil, err
	}

	return &services.EmailSender{
		ProfileID:  profile.ID,
		Email:      senderEmail(req, profile),
		Name:       profile.SenderName,
		ReplyTo:    profile.ReplyToEmail,
		Suppressed: split.Suppressed,
		Refused:    len(split.To) == 0 && len(req.To) > 0,
	}, nil
}

func senderEmail(req *services.SendEmailRequest, profile *email.Profile) string {
	if from := strings.TrimSpace(req.FromEmail); from != "" {
		return from
	}

	return profile.SenderEmail
}

type suppressionSplit struct {
	To         []string
	CC         []string
	BCC        []string
	Suppressed []string
}

func (s *Service) splitSuppressed(
	ctx context.Context,
	req *services.SendEmailRequest,
) (*suppressionSplit, error) {
	split := &suppressionSplit{}
	seen := make(map[string]bool)
	keep := func(recipients []string) ([]string, error) {
		kept := make([]string, 0, len(recipients))
		for _, recipient := range recipients {
			normalized := stringutils.NormalizeEmailAddress(recipient)
			refused, checked := seen[normalized]
			if !checked {
				var err error
				refused, err = s.repo.HasSuppression(ctx, req.TenantInfo, recipient)
				if err != nil {
					return nil, err
				}
				seen[normalized] = refused
				if refused {
					split.Suppressed = append(split.Suppressed, recipient)
				}
			}
			if !refused {
				kept = append(kept, recipient)
			}
		}
		return kept, nil
	}
	var err error
	if split.To, err = keep(req.To); err != nil {
		return nil, err
	}
	if split.CC, err = keep(req.CC); err != nil {
		return nil, err
	}
	if split.BCC, err = keep(req.BCC); err != nil {
		return nil, err
	}
	return split, nil
}
