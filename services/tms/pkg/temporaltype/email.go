package temporaltype

import (
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pulid"
)

// SendEmailPayload is the input for email sending workflows
type SendEmailPayload struct {
	OrganizationID pulid.ID                  `json:"organizationId"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId"`
	UserID         pulid.ID                  `json:"userId"`
	ProfileID      *pulid.ID                 `json:"profileId,omitempty"`
	To             []string                  `json:"to"`
	CC             []string                  `json:"cc,omitempty"`
	BCC            []string                  `json:"bcc,omitempty"`
	Subject        string                    `json:"subject"`
	HTMLBody       string                    `json:"htmlBody"`
	TextBody       string                    `json:"textBody,omitempty"`
	Priority       email.Priority            `json:"priority,omitempty"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
	Attachments    []services.AttachmentMeta `json:"attachments,omitempty"`
}

// SendTemplatedEmailPayload is the input for templated email workflows
type SendTemplatedEmailPayload struct {
	OrganizationID pulid.ID                   `json:"organizationId"`
	BusinessUnitID pulid.ID                   `json:"businessUnitId"`
	UserID         pulid.ID                   `json:"userId"`
	ProfileID      *pulid.ID                  `json:"profileId,omitempty"`
	TemplateKey    services.SystemTemplateKey `json:"templateKey"`
	To             []string                   `json:"to"`
	CC             []string                   `json:"cc,omitempty"`
	BCC            []string                   `json:"bcc,omitempty"`
	Variables      map[string]any             `json:"variables"`
	Priority       email.Priority             `json:"priority,omitempty"`
	Metadata       map[string]any             `json:"metadata,omitempty"`
	Attachments    []services.AttachmentMeta  `json:"attachments,omitempty"`
}

// EmailResult is the result returned from email workflows
type EmailResult struct {
	MessageID    string         `json:"messageId"`
	Status       string         `json:"status"`
	ProviderType string         `json:"providerType"`
	SentAt       int64          `json:"sentAt"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}
