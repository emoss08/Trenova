package emailservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

var (
	ErrRetryableSend    = serviceports.ErrRetryableEmailSend
	ErrNonRetryableSend = serviceports.ErrNonRetryableEmailSend
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
}

type ProviderAttachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

type SendProviderResponse struct {
	ProviderMessageID string
}
