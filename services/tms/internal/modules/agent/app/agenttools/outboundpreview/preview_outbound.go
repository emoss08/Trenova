package outboundpreview

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
)

// sensitiveTextScanner finds values in free text that look like something
// nobody should mail: an account number, a key, a personal identifier.
type sensitiveTextScanner interface {
	MaskText(text string) (string, bool)
}

// outboundScanner reads text a write would send outside the organization.
// Detection needs no key; the masking the audit log applies is not used.
var outboundScanner sensitiveTextScanner = auditservice.NewSensitiveDataManager(
	config.EncryptionConfig{},
)

func WarnSensitiveContent(preview *agent.ToolPreview, texts ...string) {
	for _, text := range texts {
		if _, found := outboundScanner.MaskText(text); found {
			toolpreview.Warn(preview, agent.PreviewWarningSensitiveContent,
				"The message contains text that looks sensitive, such as an account "+
					"number, a key or a personal identifier. Read it before it goes out.")

			return
		}
	}
}

// resolveEmailSender names who an email would go out as, and warns when the
// suppression list would refuse a recipient, because the send would then
// fail. A deployment without a resolver names no sender.
func ResolveEmailSender(
	ctx context.Context,
	senders emailports.EmailSenderResolver,
	req *emailports.SendEmailRequest,
) (*emailports.EmailSender, error) {
	if senders == nil {
		return nil, nil //nolint:nilnil // a deployment without a resolver names no sender
	}

	return senders.ResolveSender(ctx, req)
}

func WarnSuppressedRecipients(preview *agent.ToolPreview, sender *emailports.EmailSender) {
	if sender == nil || len(sender.Suppressed) == 0 {
		return
	}

	listed := strings.Join(sender.Suppressed, ", ")
	if sender.Refused {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail,
			"The email would not be sent: every recipient ("+listed+
				") is on the organization's suppression list.",
			sender.Suppressed...)
		return
	}
	toolpreview.Warn(preview, agent.PreviewWarningRecipientSkipped,
		listed+" is on the organization's suppression list and would be skipped.",
		sender.Suppressed...)
}

func EmailMessagePreview(
	sender *emailports.EmailSender,
	message *agent.MessagePreview,
) *agent.MessagePreview {
	message.Channel = agent.MessageChannelEmail
	if sender != nil {
		message.From = sender.Address()
	}

	return message
}
