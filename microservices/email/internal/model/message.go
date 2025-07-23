// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package model

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/bytedance/sonic"
)

// TemplateRenderer is an interface for rendering templates
type TemplateRenderer interface {
	RenderTemplate(name string, data any) (string, error)
	RenderInlineTemplate(content string, data any) (string, error)
}

// Type represents the message type
type Type string

const (
	// TypeEmailSend is the type for sending an email
	TypeEmailSend Type = "email.send"
)

// Message is the message that is sent to the email service
type Message struct {
	ID          string    `json:"id"          bun:"id,pk"`
	Type        Type      `json:"type"        bun:"type"`
	EntityID    string    `json:"entityId"    bun:"entity_id"`
	EntityType  string    `json:"entityType"  bun:"entity_type"`
	TenantID    string    `json:"tenantId"    bun:"tenant_id"`
	RequestedAt time.Time `json:"requestedAt" bun:"requested_at"`
	Payload     any       `json:"payload"     bun:"payload,type:jsonb"`
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
	ID         string      `json:"id"                 bun:"id,pk"`
	TenantID   string      `json:"tenantId"           bun:"tenant_id"`
	Status     EmailStatus `json:"status"             bun:"status"`
	RetryCount int         `json:"retryCount"         bun:"retry_count"`
	ErrorMsg   string      `json:"errorMsg,omitempty" bun:"error_msg"`
	MessageID  string      `json:"messageId"          bun:"message_id"`
	Template   string      `json:"template"           bun:"template"`
	Subject    string      `json:"subject"            bun:"subject"`
	To         []string    `json:"to"                 bun:"to,array"`
	Cc         []string    `json:"cc,omitempty"       bun:"cc,array"`
	Bcc        []string    `json:"bcc,omitempty"      bun:"bcc,array"`
	CreatedAt  time.Time   `json:"createdAt"          bun:"created_at"`
	UpdatedAt  time.Time   `json:"updatedAt"          bun:"updated_at"`
	SentAt     *time.Time  `json:"sentAt,omitempty"   bun:"sent_at"`
	Data       any         `json:"data,omitempty"     bun:"data,type:jsonb"`
}

// GetHTMLBody returns the HTML body of the email
func (e *Email) GetHTMLBody() string {
	// Create template data structure
	templateData := prepareTemplateData(e)

	// Create a simple template with proper HTML structure
	const baseTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .content { padding: 20px; background-color: #f9f9f9; border-radius: 5px; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #999; }
        table { width: 100%; border-collapse: collapse; }
        table td { padding: 8px; border-bottom: 1px solid #ddd; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Subject}}</h1>
        </div>
        <div class="content">
            {{if .DataItems}}
                <table>
                {{range .DataItems}}
                    <tr>
                        <td><strong>{{.Key}}</strong></td>
                        <td>{{.Value}}</td>
                    </tr>
                {{end}}
                </table>
            {{else}}
                <p>No content available</p>
            {{end}}
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Trenova. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	// Initialize the template
	tmpl, err := template.New("email").Parse(baseTemplate)
	if err != nil {
		return fmt.Sprintf("<html><body><h1>%s</h1><p>Template error: %s</p></body></html>",
			e.Subject, err.Error())
	}

	// Render the template
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, templateData); err != nil {
		return fmt.Sprintf(
			"<html><body><h1>%s</h1><p>Template execution error: %s</p></body></html>",
			e.Subject,
			err.Error(),
		)
	}

	return buffer.String()
}

// KeyValue represents a key-value pair for template data
type KeyValue struct {
	Key   string
	Value string
}

// TemplateData contains data for the email template
type TemplateData struct {
	Subject   string
	Year      int
	DataItems []KeyValue
}

// prepareTemplateData extracts and formats data for the template
func prepareTemplateData(e *Email) TemplateData {
	data := TemplateData{
		Subject: e.Subject,
		Year:    time.Now().Year(),
	}

	if e.Data == nil {
		return data
	}

	dataMap, ok := e.Data.(map[string]any)
	if !ok {
		// Try to convert from JSON if it's stored as a string or other format
		dataBytes, err := sonic.Marshal(e.Data)
		if err == nil {
			var newDataMap map[string]any
			if err = sonic.Unmarshal(dataBytes, &newDataMap); err == nil {
				dataMap = newDataMap
			}
		}
	}

	if dataMap != nil {
		// Skip attachments in the HTML content
		delete(dataMap, "attachments")

		// Convert map to slices for template
		for key, value := range dataMap {
			data.DataItems = append(data.DataItems, KeyValue{
				Key:   key,
				Value: fmt.Sprintf("%v", value),
			})
		}
	}

	return data
}

// RenderHTMLBody renders the email body using the provided template service
func (e *Email) RenderHTMLBody(templateService TemplateRenderer) (string, error) {
	// Use the Template field to determine which template to render
	if e.Template == "" {
		// If no template specified, use GetHTMLBody as fallback
		return e.GetHTMLBody(), nil
	}

	if e.Template == "custom" && e.Data != nil {
		// For custom templates, check if there's an inline template string in the data
		dataMap, ok := e.Data.(map[string]any)
		if ok {
			if inlineTemplate, has := dataMap["template_html"].(string); has &&
				inlineTemplate != "" {
				// Render the inline template with the data
				return templateService.RenderInlineTemplate(inlineTemplate, dataMap)
			}
		}
		// If no inline template found, fall back to GetHTMLBody
		return e.GetHTMLBody(), nil
	}

	// For named templates, use the template service to render it
	return templateService.RenderTemplate(e.Template, e.Data)
}
