package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// EmailService handles email sending operations
type EmailService interface {
	// SendEmail sends an email immediately
	SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error)

	// SendTemplatedEmail sends an email using a template
	SendTemplatedEmail(
		ctx context.Context,
		req *SendTemplatedEmailRequest,
	) (*SendEmailResponse, error)

	// QueueEmail adds an email to the queue for later processing
	QueueEmail(ctx context.Context, queue *email.Queue) (*email.Queue, error)

	// ProcessEmailQueue processes pending emails in the queue
	ProcessEmailQueue(ctx context.Context) error

	// TestEmailProfile tests an email profile configuration
	TestEmailProfile(
		ctx context.Context,
		req repositories.GetEmailProfileByIDRequest,
	) (*TestEmailProfileResponse, error)

	// LogEmailEvent logs an email event (open, click, bounce, etc.)
	LogEmailEvent(ctx context.Context, log *email.Log) error

	// GetEmailStatus retrieves the status of a queued email
	GetEmailStatus(ctx context.Context, queueID pulid.ID) (*EmailStatusResponse, error)

	// RetryFailedEmail retries a failed email
	RetryFailedEmail(ctx context.Context, queueID pulid.ID) error

	// CancelScheduledEmail cancels a scheduled email
	CancelScheduledEmail(ctx context.Context, queueID pulid.ID) error
}

// EmailProfileService manages email configuration profiles
type EmailProfileService interface {
	// Create creates a new email profile
	Create(
		ctx context.Context,
		profile *email.Profile,
		userID pulid.ID,
	) (*email.Profile, error)

	// Update updates an existing email profile
	Update(ctx context.Context, profile *email.Profile) (*email.Profile, error)

	// Get retrieves an email profile by ID
	Get(ctx context.Context, req repositories.GetEmailProfileByIDRequest) (*email.Profile, error)

	// List retrieves a list of email profiles
	List(
		ctx context.Context,
		req *repositories.ListEmailProfileRequest,
	) (*ports.ListResult[*email.Profile], error)

	// Delete deletes an email profile
	Delete(ctx context.Context, req repositories.DeleteEmailProfileRequest) error

	// SetDefault sets a profile as the default for the organization
	SetDefault(ctx context.Context, req repositories.GetEmailProfileByIDRequest) error

	// GetDefault retrieves the default email profile for an organization
	GetDefault(
		ctx context.Context,
		req repositories.GetEmailProfileByIDRequest,
	) (*email.Profile, error)
}

// EmailTemplateService manages email templates
type EmailTemplateService interface {
	// Create creates a new email template
	Create(ctx context.Context, template *email.Template) (*email.Template, error)

	// Update updates an existing email template
	Update(ctx context.Context, template *email.Template) (*email.Template, error)

	// Get retrieves an email template by ID
	Get(ctx context.Context, id pulid.ID) (*email.Template, error)

	// GetBySlug retrieves an email template by slug
	GetBySlug(ctx context.Context, slug string, organizationID pulid.ID) (*email.Template, error)

	// List retrieves a list of email templates
	List(
		ctx context.Context,
		filter *ports.QueryOptions,
	) (*ports.ListResult[*email.Template], error)

	// Delete deletes an email template
	Delete(ctx context.Context, id pulid.ID) error

	// PreviewTemplate previews a template with sample data
	PreviewTemplate(
		ctx context.Context,
		id pulid.ID,
		variables map[string]any,
	) (*PreviewTemplateResponse, error)

	// ValidateVariables validates variables against a template's schema
	ValidateVariables(ctx context.Context, templateID pulid.ID, variables map[string]any) error

	// RenderTemplate renders a template with the given variables
	RenderTemplate(
		ctx context.Context,
		template *email.Template,
		variables map[string]any,
	) (*RenderedTemplate, error)
}

// EmailQueueService manages email queue operations
type EmailQueueService interface {
	// Create creates a new email queue entry
	Create(ctx context.Context, queue *email.Queue) (*email.Queue, error)

	// Update updates an email queue entry
	Update(ctx context.Context, queue *email.Queue) (*email.Queue, error)

	// Get retrieves an email queue entry by ID
	Get(ctx context.Context, id pulid.ID) (*email.Queue, error)

	// List retrieves a list of email queue entries
	List(
		ctx context.Context,
		filter *ports.QueryOptions,
	) (*ports.ListResult[*email.Queue], error)

	// GetPending retrieves pending emails to process
	GetPending(ctx context.Context, limit int) ([]*email.Queue, error)

	// GetScheduled retrieves scheduled emails that are due
	GetScheduled(ctx context.Context, limit int) ([]*email.Queue, error)

	// MarkAsSent marks an email as sent
	MarkAsSent(ctx context.Context, queueID pulid.ID, messageID string) error

	// MarkAsFailed marks an email as failed
	MarkAsFailed(ctx context.Context, queueID pulid.ID, errorMessage string) error

	// IncrementRetryCount increments the retry count for a failed email
	IncrementRetryCount(ctx context.Context, queueID pulid.ID) error
}

// EmailLogService manages email delivery logs
type EmailLogService interface {
	// Create creates a new email log entry
	Create(ctx context.Context, log *email.Log) (*email.Log, error)

	// Get retrieves an email log entry by ID
	Get(ctx context.Context, id pulid.ID) (*email.Log, error)

	// GetByQueueID retrieves logs for a specific queue entry
	GetByQueueID(ctx context.Context, queueID pulid.ID) ([]*email.Log, error)

	// GetByMessageID retrieves a log by provider message ID
	GetByMessageID(ctx context.Context, messageID string) (*email.Log, error)

	// List retrieves a list of email logs
	List(
		ctx context.Context,
		filter *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*email.Log], error)
}

// Request and Response types

type SendEmailRequest struct {
	OrganizationID pulid.ID               `json:"organizationId"`
	BusinessUnitID pulid.ID               `json:"businessUnitId"`
	UserID         pulid.ID               `json:"userId"`
	ProfileID      *pulid.ID              `json:"profileId,omitempty"` // Use default if not provided
	To             []string               `json:"to"`
	CC             []string               `json:"cc,omitempty"`
	BCC            []string               `json:"bcc,omitempty"`
	Subject        string                 `json:"subject"`
	HTMLBody       string                 `json:"htmlBody"`
	TextBody       string                 `json:"textBody,omitempty"`
	Attachments    []email.AttachmentMeta `json:"attachments,omitempty"`
	Priority       email.Priority         `json:"priority,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
}

func (r *SendEmailRequest) Validate() error {
	return validation.ValidateStruct(
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&r.To,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.Email.Error("Invalid email address in To field")),
		),
		validation.Field(
			&r.CC,
			validation.Each(is.Email.Error("Invalid email address in CC field")),
		),
		validation.Field(
			&r.BCC,
			validation.Each(is.Email.Error("Invalid email address in BCC field")),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 500).Error("Subject must be between 1 and 500 characters"),
		),
		// validation.Field(&r.HTMLBody,
		// 	validation.Required.Error("HTML body is required"),
		// 	validation.Length(1, 10485760).Error("HTML body must not exceed 10MB"),
		// ),
		validation.Field(&r.TextBody,
			validation.Length(0, 5242880).Error("Text body must not exceed 5MB"),
		),
	)
}

type SendTemplatedEmailRequest struct {
	OrganizationID pulid.ID               `json:"organizationId"`
	BusinessUnitID pulid.ID               `json:"businessUnitId"`
	ProfileID      *pulid.ID              `json:"profileId,omitempty"`
	TemplateID     pulid.ID               `json:"templateId"`
	To             []string               `json:"to"`
	CC             []string               `json:"cc,omitempty"`
	BCC            []string               `json:"bcc,omitempty"`
	Variables      map[string]any         `json:"variables"`
	Attachments    []email.AttachmentMeta `json:"attachments,omitempty"`
	Priority       email.Priority         `json:"priority,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
}

func (r *SendTemplatedEmailRequest) Validate() error {
	return validation.ValidateStruct(
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&r.TemplateID, validation.Required.Error("Template ID is required")),
		validation.Field(&r.To,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.Email.Error("Invalid email address in To field")),
		),
		validation.Field(
			&r.CC,
			validation.Each(is.Email.Error("Invalid email address in CC field")),
		),
		validation.Field(
			&r.BCC,
			validation.Each(is.Email.Error("Invalid email address in BCC field")),
		),
		validation.Field(
			&r.Variables,
			validation.Required.Error("Template variables are required"),
		),
	)
}

type SendEmailResponse struct {
	QueueID   pulid.ID `json:"queueId"`
	MessageID string   `json:"messageId,omitempty"`
	Status    string   `json:"status"`
}

type TestEmailProfileResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type EmailStatusResponse struct {
	QueueID      pulid.ID          `json:"queueId"`
	Status       email.QueueStatus `json:"status"`
	SentAt       *int64            `json:"sentAt,omitempty"`
	ScheduledAt  *int64            `json:"scheduledAt,omitempty"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	RetryCount   int               `json:"retryCount"`
	Logs         []*email.Log      `json:"logs,omitempty"`
}

type PreviewTemplateResponse struct {
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlBody"`
	TextBody string `json:"textBody"`
}

type RenderedTemplate struct {
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlBody"`
	TextBody string `json:"textBody"`
}
