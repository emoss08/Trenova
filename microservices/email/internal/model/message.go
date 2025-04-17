package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Type represents the message type
type Type string

const (
	// TypeEmailSend is the type for sending an email
	TypeEmailSend Type = "email.send"
)

// Message is the message that is sent to the email service
type Message struct {
	ID          string    `json:"id" bun:"id,pk"`
	Type        Type      `json:"type" bun:"type"`
	EntityID    string    `json:"entityId" bun:"entity_id"`
	EntityType  string    `json:"entityType" bun:"entity_type"`
	TenantID    string    `json:"tenantId" bun:"tenant_id"`
	RequestedAt time.Time `json:"requestedAt" bun:"requested_at"`
	Payload     any       `json:"payload" bun:"payload,type:jsonb"`
}

// EmailPayload is the payload for sending an email
type EmailPayload struct {
	Template    string            `json:"template"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Data        map[string]any    `json:"data,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// EmailAttachment is an attachment for an email
type EmailAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // Base64 encoded content
	ContentType string `json:"contentType"`
}

// EmailStatus is the status of an email
type EmailStatus string

const (
	// EmailStatusPending is the status for a pending email
	EmailStatusPending EmailStatus = "pending"
	// EmailStatusSent is the status for a sent email
	EmailStatusSent EmailStatus = "sent"
	// EmailStatusFailed is the status for a failed email
	EmailStatusFailed EmailStatus = "failed"
	// EmailStatusRetrying is the status for an email that is being retried
	EmailStatusRetrying EmailStatus = "retrying"
)

// Email is the database model for an email
type Email struct {
	ID         string      `json:"id" bun:"id,pk"`
	TenantID   string      `json:"tenantId" bun:"tenant_id"`
	Status     EmailStatus `json:"status" bun:"status"`
	RetryCount int         `json:"retryCount" bun:"retry_count"`
	ErrorMsg   string      `json:"errorMsg,omitempty" bun:"error_msg"`
	MessageID  string      `json:"messageId" bun:"message_id"`
	Template   string      `json:"template" bun:"template"`
	Subject    string      `json:"subject" bun:"subject"`
	To         []string    `json:"to" bun:"to,array"`
	Cc         []string    `json:"cc,omitempty" bun:"cc,array"`
	Bcc        []string    `json:"bcc,omitempty" bun:"bcc,array"`
	CreatedAt  time.Time   `json:"createdAt" bun:"created_at"`
	UpdatedAt  time.Time   `json:"updatedAt" bun:"updated_at"`
	SentAt     *time.Time  `json:"sentAt,omitempty" bun:"sent_at"`
	Data       any         `json:"data,omitempty" bun:"data,type:jsonb"`
}

// GetHTMLBody returns the HTML body of the email
func (e *Email) GetHTMLBody() string {
	// This would normally be processed through a template engine
	// Here we're just returning a simple HTML version using the data
	if e.Data == nil {
		return fmt.Sprintf("<html><body><h1>%s</h1><p>No content available</p></body></html>", e.Subject)
	}

	// Format the data as a simple HTML table
	dataMap, ok := e.Data.(map[string]interface{})
	if !ok {
		// Try to convert from JSON if it's stored as a string or other format
		dataBytes, err := json.Marshal(e.Data)
		if err != nil {
			return fmt.Sprintf("<html><body><h1>%s</h1><p>Error formatting email data</p></body></html>", e.Subject)
		}

		var newDataMap map[string]interface{}
		if err := json.Unmarshal(dataBytes, &newDataMap); err != nil {
			return fmt.Sprintf("<html><body><h1>%s</h1><p>Email data format error</p></body></html>", e.Subject)
		}
		dataMap = newDataMap
	}

	// Skip attachments in the HTML content
	delete(dataMap, "attachments")

	// Build a simple HTML representation
	var htmlContent string
	htmlContent += fmt.Sprintf("<html><body><h1>%s</h1><table>", e.Subject)

	for key, value := range dataMap {
		htmlContent += fmt.Sprintf("<tr><td><strong>%s</strong></td><td>%v</td></tr>", key, value)
	}

	htmlContent += "</table></body></html>"
	return htmlContent
}
