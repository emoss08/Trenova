package email

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Queue)(nil)
	_ domain.Validatable        = (*Queue)(nil)
	_ infra.PostgresSearchable  = (*Queue)(nil)
)

// AttachmentMeta represents metadata for an email attachment
type AttachmentMeta struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	ContentID   string `json:"contentId,omitempty"`
}

// Queue represents an email in the sending queue
type Queue struct {
	bun.BaseModel `bun:"table:email_queues,alias:eq" json:"-"`

	ID                pulid.ID         `json:"id"                          bun:"id,type:varchar(100),pk"`
	OrganizationID    pulid.ID         `json:"organizationId"              bun:"organization_id,type:varchar(100),notnull"`
	BusinessUnitID    pulid.ID         `json:"businessUnitId"              bun:"business_unit_id,type:varchar(100),notnull"`
	ProfileID         pulid.ID         `json:"profileId"                   bun:"profile_id,type:varchar(100),notnull"`
	TemplateID        *pulid.ID        `json:"templateId,omitempty"        bun:"template_id,type:varchar(100)"`
	ToAddresses       []string         `json:"toAddresses"                 bun:"to_addresses,type:text[],notnull"`
	CCAddresses       []string         `json:"ccAddresses,omitempty"       bun:"cc_addresses,type:text[]"`
	BCCAddresses      []string         `json:"bccAddresses,omitempty"      bun:"bcc_addresses,type:text[]"`
	Subject           string           `json:"subject"                     bun:"subject,type:text,notnull"`
	HTMLBody          string           `json:"htmlBody"                    bun:"html_body,type:text"`
	TextBody          string           `json:"textBody"                    bun:"text_body,type:text"`
	Attachments       []AttachmentMeta `json:"attachments,omitempty"       bun:"attachments,type:jsonb"`
	Priority          Priority         `json:"priority"                    bun:"priority,type:email_priority_enum,default:'medium'"`
	Status            QueueStatus      `json:"status"                      bun:"status,type:email_queue_status_enum,default:'pending'"`
	ScheduledAt       *int64           `json:"scheduledAt,omitempty"       bun:"scheduled_at,type:bigint"`
	SentAt            *int64           `json:"sentAt,omitempty"            bun:"sent_at,type:bigint"`
	ErrorMessage      string           `json:"errorMessage,omitempty"      bun:"error_message,type:text"`
	RetryCount        int              `json:"retryCount"                  bun:"retry_count,type:integer,default:0"`
	TemplateVariables map[string]any   `json:"templateVariables,omitempty" bun:"template_variables,type:jsonb"`
	Metadata          map[string]any   `json:"metadata,omitempty"          bun:"metadata,type:jsonb"`
	CreatedAt         int64            `json:"createdAt"                   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64            `json:"updatedAt"                   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"businessUnit,omitempty"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"organization,omitempty"`
	Profile      *Profile                   `bun:"rel:belongs-to,join:profile_id=id"       json:"profile,omitempty"`
	Template     *Template                  `bun:"rel:belongs-to,join:template_id=id"      json:"template,omitempty"`
}

// Validate implements the Validatable interface
func (q *Queue) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, q,
		// Basic fields validation
		validation.Field(&q.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&q.ProfileID,
			validation.Required.Error("Email Profile is required"),
		),

		// Recipients validation
		validation.Field(&q.ToAddresses,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.Email.Error("Invalid email address in To field")),
		),
		validation.Field(&q.CCAddresses,
			validation.Each(is.Email.Error("Invalid email address in CC field")),
		),
		validation.Field(&q.BCCAddresses,
			validation.Each(is.Email.Error("Invalid email address in BCC field")),
		),

		// Content validation
		validation.Field(&q.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 500).Error("Subject must be between 1 and 500 characters"),
		),
		validation.Field(&q.HTMLBody,
			validation.When(
				q.TextBody == "",
				validation.Required.Error("Either HTML body or Text body is required"),
			),
			validation.Length(0, 10485760).Error("HTML body must not exceed 10MB"),
		),
		validation.Field(&q.TextBody,
			validation.When(
				q.HTMLBody == "",
				validation.Required.Error("Either HTML body or Text body is required"),
			),
			validation.Length(0, 5242880).Error("Text body must not exceed 5MB"),
		),

		// Queue management validation
		validation.Field(&q.Priority,
			validation.In(
				PriorityHigh,
				PriorityMedium,
				PriorityLow,
			).Error("Priority must be high, medium, or low"),
		),
		validation.Field(&q.Status,
			validation.In(
				QueueStatusPending,
				QueueStatusProcessing,
				QueueStatusSent,
				QueueStatusFailed,
				QueueStatusScheduled,
				QueueStatusCancelled,
			).Error("Invalid queue status"),
		),
		validation.Field(&q.ScheduledAt,
			validation.When(
				q.Status == QueueStatusScheduled,
				validation.Required.Error("Scheduled time is required for scheduled emails"),
				validation.Min(time.Now().Unix()).Error("Scheduled time must be in the future"),
			),
		),
		validation.Field(&q.RetryCount,
			validation.Min(0).Error("Retry count cannot be negative"),
			validation.Max(10).Error("Maximum retry count exceeded"),
		),

		// Attachments validation
		validation.Field(&q.Attachments,
			validation.Each(validation.By(validateAttachment)),
		),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (q *Queue) GetTableName() string {
	return "email_queue"
}

// validateAttachment validates individual attachment metadata
func validateAttachment(value any) error {
	attachment, ok := value.(AttachmentMeta)
	if !ok {
		return errors.NewValidationError(
			"attachments",
			errors.ErrInvalid,
			"Invalid attachment format",
		)
	}

	return validation.ValidateStruct(&attachment,
		validation.Field(&attachment.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, 255).Error("File name must be between 1 and 255 characters"),
		),
		validation.Field(&attachment.ContentType,
			validation.Required.Error("Content type is required"),
			validation.Length(1, 100).Error("Content type must be between 1 and 100 characters"),
		),
		validation.Field(&attachment.Size,
			validation.Required.Error("File size is required"),
			validation.Min(1).Error("File size must be at least 1 byte"),
			validation.Max(26214400).Error("File size must not exceed 25MB"),
		),
		validation.Field(&attachment.URL,
			validation.Required.Error("File URL is required"),
		),
	)
}

// CanRetry returns true if the email can be retried
func (q *Queue) CanRetry() bool {
	return q.Status == QueueStatusFailed && q.RetryCount < 10
}

// IsScheduled returns true if the email is scheduled for future sending
func (q *Queue) IsScheduled() bool {
	return q.Status == QueueStatusScheduled && q.ScheduledAt != nil &&
		*q.ScheduledAt > time.Now().Unix()
}

// GetTotalRecipients returns the total number of recipients
func (q *Queue) GetTotalRecipients() int {
	return len(q.ToAddresses) + len(q.CCAddresses) + len(q.BCCAddresses)
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (q *Queue) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if q.ID.IsNil() {
			q.ID = pulid.MustNew("emq_") // email queue
		}
		q.CreatedAt = now
	case *bun.UpdateQuery:
		q.UpdatedAt = now
	}

	return nil
}

// GetPostgresSearchConfig implements the PostgresSearchable interface
func (q *Queue) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "eq",
		Fields: []infra.PostgresSearchableField{
			{Name: "subject", Weight: "A", Type: infra.PostgresSearchTypeText},
			{Name: "to_addresses", Weight: "B", Type: infra.PostgresSearchTypeArray},
			{Name: "error_message", Weight: "C", Type: infra.PostgresSearchTypeText},
			{Name: "status", Weight: "D", Type: infra.PostgresSearchTypeEnum},
		},
		MinLength:       2,
		MaxTerms:        5,
		UsePartialMatch: true,
	}
}
