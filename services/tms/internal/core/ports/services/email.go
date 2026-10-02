package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrRetryableEmailSend    = errors.New("retryable email provider failure")
	ErrNonRetryableEmailSend = errors.New("non-retryable email provider failure")
)

type SendEmailRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ProfileID      pulid.ID              `json:"profileId"`
	Purpose        email.Purpose         `json:"purpose"`
	FromEmail      string                `json:"fromEmail"`
	To             []string              `json:"to"`
	CC             []string              `json:"cc"`
	BCC            []string              `json:"bcc"`
	Subject        string                `json:"subject"`
	HTML           string                `json:"html"`
	Text           string                `json:"text"`
	Attachments    []EmailAttachment     `json:"attachments"`
	Headers        map[string]string     `json:"headers"`
	OpenTracking   bool                  `json:"openTracking"`
	IdempotencyKey string                `json:"idempotencyKey"`
}

type EmailAttachment struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Content     []byte `json:"content,omitempty"`
	ObjectKey   string `json:"objectKey,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type SendPersistedEmailRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	MessageID    pulid.ID              `json:"messageId"`
	HTML         string                `json:"html"`
	Text         string                `json:"text"`
	Headers      map[string]string     `json:"headers"`
	OpenTracking bool                  `json:"openTracking"`
}

type TestEmailProfileRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

type EmailService interface {
	Send(context.Context, *SendEmailRequest) (*email.Message, error)
	SendPersisted(context.Context, *SendPersistedEmailRequest) (*email.Message, error)
}
