package agenttoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/shopspring/decimal"
)

// sensitiveTextScanner finds values in free text that look like something
// nobody should mail: an account number, a key, a personal identifier.
type sensitiveTextScanner interface {
	MaskText(text string) (string, bool)
}

// The record fields previews name, as the tools' parameters spell them.
const (
	previewFieldCustomerID     = "customerId"
	previewFieldDefinitionID   = "definitionId"
	previewFieldPaymentDate    = "paymentDate"
	previewFieldServiceTypeID  = "serviceTypeId"
	previewFieldShipmentID     = "shipmentId"
	previewFieldShipmentMoveID = "shipmentMoveId"
	previewFieldShipmentTypeID = "shipmentTypeId"
	previewFieldSubject        = "subject"
	previewFieldTractorTypeID  = "tractorTypeId"
	previewFieldTrailerTypeID  = "trailerTypeId"
)

// outboundScanner reads text a write would send outside the organization.
// Detection needs no key; the masking the audit log applies is not used.
var outboundScanner sensitiveTextScanner = auditservice.NewSensitiveDataManager(
	config.EncryptionConfig{},
)

func warnSensitiveContent(preview *agent.ToolPreview, texts ...string) {
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
func resolveEmailSender(
	ctx context.Context,
	senders serviceports.EmailSenderResolver,
	req *serviceports.SendEmailRequest,
) (*serviceports.EmailSender, error) {
	if senders == nil {
		return nil, nil //nolint:nilnil // a deployment without a resolver names no sender
	}

	return senders.ResolveSender(ctx, req)
}

func warnSuppressedRecipients(preview *agent.ToolPreview, sender *serviceports.EmailSender) {
	if sender == nil || len(sender.Suppressed) == 0 {
		return
	}

	toolpreview.Warn(preview, agent.PreviewWarningWouldFail,
		"The email would not be sent: "+strings.Join(sender.Suppressed, ", ")+
			" is on the organization's suppression list.",
		sender.Suppressed...)
}

func emailMessagePreview(
	sender *serviceports.EmailSender,
	message *agent.MessagePreview,
) *agent.MessagePreview {
	message.Channel = agent.MessageChannelEmail
	if sender != nil {
		message.From = sender.Address()
	}

	return message
}

func pinnedVersion(version int64) *int64 {
	return &version
}

func knownAmount(amount decimal.Decimal) decimal.NullDecimal {
	return decimal.NewNullDecimal(amount)
}
