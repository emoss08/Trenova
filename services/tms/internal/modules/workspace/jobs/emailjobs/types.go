package emailjobs

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/shared/pulid"
)

type SendEmailResult struct {
	MessageID         pulid.ID            `json:"messageId"`
	ProviderMessageID string              `json:"providerMessageId,omitempty"`
	Status            email.MessageStatus `json:"status"`
	Error             string              `json:"error,omitempty"`
}
