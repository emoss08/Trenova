package emailservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
)

var (
	ErrRetryableSend    = emailports.ErrRetryableEmailSend
	ErrNonRetryableSend = emailports.ErrNonRetryableEmailSend
)

type ProviderSender interface {
	Provider() email.Provider
	IntegrationType() integration.Type
	Send(context.Context, SendProviderRequest) (*SendProviderResponse, error)
}

type SendProviderRequest struct {
	Message SendProviderMessage
	Config  map[string]string
}

type SendProviderMessage struct {
	IdempotencyKey string
	From           string
	ReplyTo        string
	To             []string
	CC             []string
	BCC            []string
	Subject        string
	HTML           string
	Text           string
	Attachments    []ProviderAttachment
	Headers        map[string]string
	OpenTracking   bool
}

type ProviderAttachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

type SendProviderResponse struct {
	ProviderMessageID string
}
