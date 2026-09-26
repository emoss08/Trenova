package services

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// EmailSender is who an email would go out as, and which of its recipients
// the organization's suppression list would refuse, read the way Send reads
// them but without sending anything.
type EmailSender struct {
	ProfileID  pulid.ID
	Email      string
	Name       string
	ReplyTo    string
	Suppressed []string
}

func (s *EmailSender) Address() string {
	if s == nil {
		return ""
	}

	return stringutils.FormatEmailAddress(s.Name, s.Email)
}

// EmailSenderResolver resolves the profile a send would use, so a preview can
// name the sender and the recipients that would be refused.
type EmailSenderResolver interface {
	ResolveSender(ctx context.Context, req *SendEmailRequest) (*EmailSender, error)
}
