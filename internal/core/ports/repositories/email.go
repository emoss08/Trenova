package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetEmailProfileByIDRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	ProfileID  pulid.ID
	ExpandData bool
}

type DeleteEmailProfileRequest struct {
	ProfileID pulid.ID
	OrgID     pulid.ID
	BuID      pulid.ID
}

type ListEmailProfileRequest struct {
	Filter *ports.QueryOptions `json:"filter" query:"filter"`
}

// EmailProfileRepository handles email profile persistence
type EmailProfileRepository interface {
	// Create creates a new email profile
	Create(ctx context.Context, profile *email.Profile) (*email.Profile, error)

	// Update updates an existing email profile
	Update(ctx context.Context, profile *email.Profile) (*email.Profile, error)

	// Get retrieves an email profile by ID
	Get(ctx context.Context, req GetEmailProfileByIDRequest) (*email.Profile, error)

	// List retrieves a list of email profiles
	List(
		ctx context.Context,
		req *ListEmailProfileRequest,
	) (*ports.ListResult[*email.Profile], error)

	// Delete deletes an email profile
	Delete(ctx context.Context, req DeleteEmailProfileRequest) error

	// GetDefault retrieves the default email profile for an organization
	GetDefault(ctx context.Context, orgID pulid.ID, buID pulid.ID) (*email.Profile, error)
}

// EmailTemplateRepository handles email template persistence
type EmailTemplateRepository interface {
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
}

// EmailQueueRepository handles email queue persistence
type EmailQueueRepository interface {
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
}

// EmailLogRepository handles email log persistence
type EmailLogRepository interface {
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
		filter *ports.QueryOptions,
	) (*ports.ListResult[*email.Log], error)
}
