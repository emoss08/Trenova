/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

// EmailProfileFieldConfig defines filterable and sortable fields for email profiles
var EmailProfileFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"name":         true,
		"status":       true,
		"providerType": true,
		"fromAddress":  true,
		"host":         true,
		"isDefault":    true,
	},
	SortableFields: map[string]bool{
		"name":         true,
		"status":       true,
		"providerType": true,
		"isDefault":    true,
		"fromAddress":  true,
		"createdAt":    true,
		"updatedAt":    true,
	},
	FieldMap: map[string]string{
		"name":         "name",
		"status":       "status",
		"providerType": "provider_type",
		"fromAddress":  "from_address",
		"host":         "host",
		"isDefault":    "is_default",
		"createdAt":    "created_at",
		"updatedAt":    "updated_at",
	},
	EnumMap: map[string]bool{
		"status":         true,
		"providerType":   true,
		"authType":       true,
		"encryptionType": true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{},
}

func BuildEmailProfileListOptions(
	filter *ports.QueryOptions,
	additionalOpts *ListEmailProfileRequest,
) *ListEmailProfileRequest {
	return &ListEmailProfileRequest{
		Filter: filter,
	}
}

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
	Filter          *ports.QueryOptions `json:"filter"          query:"filter"`
	ExcludeInactive bool                `json:"excludeInactive" query:"excludeInactive"`
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

	// MarkAsSent marks an email as sent
	MarkAsSent(ctx context.Context, queueID pulid.ID, messageID string) error

	// MarkAsFailed marks an email as failed
	MarkAsFailed(ctx context.Context, queueID pulid.ID, errorMessage string) error

	// IncrementRetryCount increments the retry count for an email
	IncrementRetryCount(ctx context.Context, queueID pulid.ID) error
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
